package provider

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ReadResource function
func (s *RawProviderServer) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	resp := &tfprotov5.ReadResourceResponse{}
	var resState map[string]tftypes.Value
	var err error
	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine resource type",
			Detail:   err.Error(),
		})
		return resp, err
	}

	currentState, err := req.CurrentState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to decode current state",
			Detail:   err.Error(),
		})
		return resp, err
	}
	if currentState.IsNull() {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to read resource",
			Detail:   "incomplete of missing state",
		})
		return resp, fmt.Errorf("failed to read resource")
	}
	err = currentState.As(&resState)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract resource from current state",
			Detail:   err.Error(),
		})
		return resp, err
	}

	if resState["object"].IsNull() {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("current state of resource %s has no 'object' attribute", req.TypeName),
			Detail:   err.Error(),
		})
		return resp, err
	}
	co := resState["object"]
	cu, err := TFValueToUnstructured(co)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("%s: failed encode 'object' attribute to Unstructured", req.TypeName),
			Detail:   err.Error(),
		})
		return resp, err
	}
	s.logger.Trace("[ReadResource][TFValueToUnstructured]\n%s\n", spew.Sdump(cu))

	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("failed to get RESTMapper client"),
			Detail:   err.Error(),
		})
		return resp, err
	}
	client, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("failed to get Dynamic client"),
			Detail:   err.Error(),
		})
		return resp, err
	}

	uo := unstructured.Unstructured{Object: cu.(map[string]interface{})}
	cGVR, err := GVRFromUnstructured(&uo, rm)
	if err != nil {
		return resp, err
	}
	ns, err := IsResourceNamespaced(uo.GroupVersionKind(), rm)
	if err != nil {
		return resp, err
	}
	rcl := client.Resource(cGVR)

	rnamespace := uo.GetNamespace()
	rname := uo.GetName()

	var ro *unstructured.Unstructured
	if ns {
		ro, err = rcl.Namespace(rnamespace).Get(ctx, rname, v1.GetOptions{})
	} else {
		ro, err = rcl.Get(ctx, rname, v1.GetOptions{})
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resp, nil
		}
		d := tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("Cannot GET resource %s", spew.Sdump(co)),
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, &d)
		return resp, err
	}

	gvk, err := GVKFromTftypesObject(&co, rm)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource GVR: %s", err)
	}

	objectType, err := s.TFTypeFromOpenAPI(gvk)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	fo := FilterEphemeralFields(ro.Object)
	nobj, err := UnstructuredToTFValue(fo, objectType, tftypes.AttributePath{})
	if err != nil {
		return resp, err
	}

	nobj, err = TFValueDeepUnknown(objectType, nobj)
	if err != nil {
		return resp, err
	}

	rawState := make(map[string]tftypes.Value)
	err = currentState.As(&rawState)
	if err != nil {
		return resp, err
	}
	rawState["object"] = TFValueUnknownToNull(nobj)

	nsVal := tftypes.NewValue(currentState.Type(), rawState)
	newState, err := tfprotov5.NewDynamicValue(nsVal.Type(), nsVal)
	if err != nil {
		return resp, err
	}
	resp.NewState = &newState
	return resp, nil
}
