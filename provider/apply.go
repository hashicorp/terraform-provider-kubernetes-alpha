package provider

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// ApplyResourceChange function
func (s *RawProviderServer) ApplyResourceChange(ctx context.Context, req *tfprotov5.ApplyResourceChangeRequest) (*tfprotov5.ApplyResourceChangeResponse, error) {
	resp := &tfprotov5.ApplyResourceChangeResponse{}
	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine planned resource type",
			Detail:   err.Error(),
		})
		return resp, err
	}

	applyPlannedState, err := req.PlannedState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal planned resource state",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[ApplyResourceChange][Request][PlannedState]\n%s\n", spew.Sdump(applyPlannedState))

	applyPriorState, err := req.PriorState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal prior resource state",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[ApplyResourceChange][Request][PriorState]\n%s\n", spew.Sdump(applyPriorState))

	c, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics,
			&tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to retrieve Kubrentes dynamic client during apply",
				Detail:   err.Error(),
			})
		return resp, err
	}
	m, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics,
			&tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to retrieve Kubrentes RESTMapper client during apply",
				Detail:   err.Error(),
			})
		return resp, err
	}
	var rs dynamic.ResourceInterface

	switch {
	case applyPriorState.IsNull() || (!applyPlannedState.IsNull() && !applyPriorState.IsNull()):
		// Apply resource
		var plannedStateVal map[string]tftypes.Value = make(map[string]tftypes.Value)
		err = applyPlannedState.As(&plannedStateVal)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to extract planned resource state values",
				Detail:   err.Error(),
			})
			return resp, err
		}
		obj, ok := plannedStateVal["object"]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to find object value in planned resource state",
			})
			return resp, err
		}

		gvk, err := GVKFromTftypesObject(&obj, m)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource GVK: %s", err)
		}

		tsch, err := s.TFTypeFromOpenAPI(gvk)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
		}

		pu, err := TFValueToUnstructured(&obj)
		if err != nil {
			return resp, err
		}

		// remove null attributes - the API doesn't appreciate requests that include them
		uo := unstructured.Unstructured{Object: mapRemoveNulls(pu.(map[string]interface{}))}
		rnamespace := uo.GetNamespace()
		rname := uo.GetName()

		gvr, err := GVRFromUnstructured(&uo, m)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource GVR: %s", err)
		}

		ns, err := IsResourceNamespaced(gvk, m)
		if err != nil {
			return resp, err
		}

		if ns {
			rs = c.Resource(gvr).Namespace(rnamespace)
		} else {
			rs = c.Resource(gvr)
		}
		jd, err := uo.MarshalJSON()
		if err != nil {
			return resp, err
		}

		// Call the Kubernetes API to create the new resource
		result, err := rs.Patch(ctx, rname, types.ApplyPatchType, jd, v1.PatchOptions{FieldManager: "Terraform"})
		if err != nil {
			Dlog.Printf("[ApplyResourceChange][Apply] Error: %s\n%s\n", spew.Sdump(err), spew.Sdump(result))
			rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   err.Error(),
					Summary:  fmt.Sprintf("PATCH resource %s failed!\nAPI Error: %s", rn, err),
				})
			return resp, fmt.Errorf("PATCH resource %s failed: %s", rn, err)
		}
		Dlog.Printf("[ApplyResourceChange][Apply] API response:\n%s\n", spew.Sdump(result.Object["spec"].(map[string]interface{})["versions"]))

		newResObject, err := UnstructuredToTFValue(FilterEphemeralFields(result.Object), tsch, tftypes.AttributePath{})
		if err != nil {
			return resp, err
		}
		Dlog.Printf("[ApplyResourceChange][Apply][ObjectResponse]\n%s\n", spew.Sdump(newResObject))

		// TODO: convert waiter to tftypes
		// err = s.waitForCompletion(ctx, applyPlannedState, rs, rname, tsch)
		// if err != nil {
		// 	return resp, err
		// }

		compObj, err := TFValueDeepUnknown(tsch, newResObject)
		if err != nil {
			return resp, err
		}
		plannedStateVal["object"] = TFValueUnknownToNull(compObj)

		newStateVal := tftypes.NewValue(applyPlannedState.Type(), plannedStateVal)
		Dlog.Printf("[ApplyResourceChange][Apply][NewState]\n%s\n", spew.Sdump(newStateVal))

		newResState, err := tfprotov5.NewDynamicValue(newStateVal.Type(), newStateVal)
		if err != nil {
			return resp, err
		}
		Dlog.Printf("[ApplyResourceChange][Create] transformed new state:\n%s", spew.Sdump(newResState))

		resp.NewState = &newResState

	case applyPlannedState.IsNull():
		// Delete the resource
		priorStateVal := make(map[string]tftypes.Value)
		err = applyPriorState.As(&priorStateVal)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to extract prior resource state values",
				Detail:   err.Error(),
			})
			return resp, err
		}
		pco, ok := priorStateVal["object"]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to find object value in prior resource state",
			})
			return resp, err
		}

		pu, err := TFValueToUnstructured(&pco)
		if err != nil {
			return resp, err
		}

		uo := unstructured.Unstructured{Object: pu.(map[string]interface{})}
		gvr, err := GVRFromUnstructured(&uo, m)
		if err != nil {
			return resp, err
		}

		gvk, err := GVKFromTftypesObject(&pco, m)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource GVK: %s", err)
		}

		ns, err := IsResourceNamespaced(gvk, m)
		if err != nil {
			return resp, err
		}

		// tsch, err := s.TFTypeFromOpenAPI(gvk)
		// if err != nil {
		// 	return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
		// }

		rnamespace := uo.GetNamespace()
		rname := uo.GetName()

		if ns {
			rs = c.Resource(gvr).Namespace(rnamespace)
		} else {
			rs = c.Resource(gvr)
		}
		err = rs.Delete(ctx, rname, v1.DeleteOptions{})
		if err != nil {
			rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   err.Error(),
					Summary:  fmt.Sprintf("DELETE resource %s failed: %s", rn, err),
				})
			return resp, fmt.Errorf("DELETE resource %s failed: %s", rn, err)
		}

		// err = s.waitForCompletion(ctx, applyPlannedState, rs, rname, tsch)
		// if err != nil {
		// 	return resp, err
		// }

		resp.NewState = req.PlannedState
	}
	return resp, nil
}
