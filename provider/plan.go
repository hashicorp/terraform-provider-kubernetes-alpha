package provider

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/morph"
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
		return resp, nil
	}
	// Decode proposed resource state
	proposedState, err := req.ProposedNewState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal planned resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[PlanResourceChange]", "[ProposedState]", spew.Sdump(proposedState))

	proposedVal := make(map[string]tftypes.Value)
	err = proposedState.As(&proposedVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract planned resource state from tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	// Decode prior resource state
	priorState, err := req.PriorState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal prior resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[PlanResourceChange]", "[PriorState]", spew.Sdump(priorState))

	priorVal := make(map[string]tftypes.Value)
	err = priorState.As(&priorVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract prior resource state from tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	if proposedState.IsNull() {
		// we plan to delete the resource
		if _, ok := priorVal["object"]; ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid prior state while planning for destroy",
				Detail:   fmt.Sprintf("'object' attribute missing from state: %s", err),
			})
			return resp, nil
		}
		resp.PlannedState = req.ProposedNewState
		return resp, nil
	}

	ppMan, ok := proposedVal["manifest"]
	if !ok {
		matp := tftypes.AttributePath{}.WithAttributeName("manifest")
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Invalid proposed state during planning",
			Detail:    "Missing 'manifest' attribute",
			Attribute: &matp,
		})
		return resp, nil
	}

	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to create K8s RESTMapper client",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	gvk, err := GVKFromTftypesObject(&ppMan, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine GroupVersionResource for manifest",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	objectType, err := s.TFTypeFromOpenAPI(gvk)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
	}
	so := objectType.(tftypes.Object)
	s.logger.Debug("[PlanUpdateResource]", "OAPI type", spew.Sdump(so))

	// Transform the input manifest to adhere to the type model from the OpenAPI spec
	mobj, err := morph.ValueToType(ppMan, objectType, tftypes.AttributePath{})
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to morph manifest to OAPI type",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Debug("[PlanResourceChange]", "morphed manifest", spew.Sdump(mobj))

	completeObj, err := morph.DeepUnknown(objectType, mobj, tftypes.AttributePath{})
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to backfill manifest from OAPI type",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Debug("[PlanResourceChange]", "backfilled manifest", spew.Sdump(completeObj))

	if proposedVal["object"].IsNull() { // plan for Create
		proposedVal["object"] = completeObj
	} else { // plan for Update
		priorObj, ok := priorVal["object"]
		if !ok {
			oatp := tftypes.AttributePath{}.WithAttributeName("object")
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Invalid prior state during planning",
				Detail:    "Missing 'object' attribute",
				Attribute: &oatp,
			})
			return resp, nil
		}
		updatedObj, err := tftypes.Transform(completeObj, func(ap tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			if v.IsKnown() && !v.IsNull() {
				return v, nil
			}
			priorAtrVal, restPath, err := tftypes.WalkAttributePath(priorObj, ap)
			if err != nil {
				return v, ap.NewError(err)
			}
			if len(restPath.Steps) > 0 {
				s.logger.Warn("[PlanResourceChange]", "Unexpected missing attribute at", ap.String(), " + ", restPath.String())
			}
			return priorAtrVal.(tftypes.Value), nil
		})
		if err != nil {
			oatp := tftypes.AttributePath{}.WithAttributeName("object")
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Failed to update proposed state from prior state",
				Detail:    err.Error(),
				Attribute: &oatp,
			})
			return resp, nil
		}

		proposedVal["object"] = updatedObj
	}

	propStateVal := tftypes.NewValue(proposedState.Type(), proposedVal)
	s.logger.Trace("[PlanResourceChange]", "new planned state", spew.Sdump(propStateVal))

	plannedState, err := tfprotov5.NewDynamicValue(propStateVal.Type(), propStateVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to assemble proposed state during plan",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	resp.PlannedState = &plannedState
	return resp, nil
}
