package provider

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// PlanResourceChange function
func (s *RawProviderServer) PlanResourceChange(ctx context.Context, req *tfprotov5.PlanResourceChangeRequest) (*tfprotov5.PlanResourceChangeResponse, error) {
	resp := &tfprotov5.PlanResourceChangeResponse{}

	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine planned resource type",
			Detail:   err.Error(),
		})
		return resp, err
	}
	// Decode proposed resource state
	proposedState, err := req.ProposedNewState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal planned resource state",
			Detail:   err.Error(),
		})
		return resp, err
	}
	proposedVal := make(map[string]tftypes.Value)
	err = proposedState.As(&proposedVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract planned resource state from tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[PlanResourceChange][PriorState]\n%s\n", spew.Sdump(proposedState))

	// Decode prior resource state
	priorState, err := req.PriorState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal prior resource state",
			Detail:   err.Error(),
		})
		return resp, err
	}
	priorVal := make(map[string]tftypes.Value)
	err = priorState.As(&priorVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract prior resource state from tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[PlanResourceChange][PriorState]\n%s\n", spew.Sdump(priorState))

	if proposedState.IsNull() {
		// we plan to delete the resource
		if _, ok := priorVal["object"]; ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to find existing object state before delete",
				Detail:   err.Error(),
			})
			return resp, err
		}
		resp.PlannedState = req.ProposedNewState
		return resp, nil
	}

	ppMan, ok := proposedVal["manifest"]
	if !ok {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Missing 'manifest' attribute",
		})
		return resp, fmt.Errorf("missing 'manifest' attribute")
	}

	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to create K8s RESTMapper client",
			Detail:   err.Error(),
		})
		return resp, err
	}
	gvk, err := GVKFromTftypesObject(&ppMan, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine GroupVersionResource for manifest",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[PlanResourceChange][ProposedNewState] GVK\n%s\n", spew.Sdump(gvk))
	// Dlog.Printf("[PlanResourceChange][ProposedNewState]\n%s\n", spew.Sdump(ppMan))

	objectType, err := s.TFTypeFromOpenAPI(gvk)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
	}
	so := objectType.(tftypes.Object)
	Dlog.Printf("[PlanUpdateResourceLocal] OAPI type: %s\n", spew.Sdump(so.AttributeTypes))

	// Transform the input manifest to adhere to the type model from the OpenAPI spec
	mobj, err := MorphValueToType(ppMan, objectType, tftypes.AttributePath{})
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to morph manifest to OAPI type",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[PlanUpdateResourceLocal] morphed manifest: %s\n", spew.Sdump(mobj))

	completeObj, err := TFValueDeepUnknown(objectType, mobj)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to backfill manifest from OAPI type",
			Detail:   err.Error(),
		})
		return resp, err
	}
	Dlog.Printf("[PlanUpdateResourceLocal] backfilled manifest: %s\n", spew.Sdump(completeObj))

	if proposedVal["object"].IsNull() { // plan for Create
		proposedVal["object"] = completeObj
	} else { // plan for Update
		// TODO: implement update
	}

	propStateVal := tftypes.NewValue(proposedState.Type(), proposedVal)
	Dlog.Printf("[PlanResourceChange] planned state: %s\n", spew.Sdump(propStateVal))

	plannedState, err := tfprotov5.NewDynamicValue(propStateVal.Type(), propStateVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to assemble proposed state during plan",
			Detail:   err.Error(),
		})
		return resp, err
	}
	resp.PlannedState = &plannedState
	return resp, nil
}
