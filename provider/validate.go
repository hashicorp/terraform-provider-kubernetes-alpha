package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

// ValidateResourceTypeConfig function
func (s *RawProviderServer) ValidateResourceTypeConfig(ctx context.Context, req *tfprotov5.ValidateResourceTypeConfigRequest) (*tfprotov5.ValidateResourceTypeConfigResponse, error) {
	resp := &tfprotov5.ValidateResourceTypeConfigResponse{}
	requiredKeys := []string{"apiVersion", "kind", "metadata"}
	forbiddenKeys := []string{"status"}

	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine resource type",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	// Decode proposed resource state
	config, err := req.Config.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	att := tftypes.AttributePath{}.WithAttributeName("manifest")

	configVal := make(map[string]tftypes.Value)
	err = config.As(&configVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract resource state from SDK value",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	manifest, ok := configVal["manifest"]
	if !ok {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Manifest missing from resource configuration",
			Detail:    "A manifest attribute containing a valid Kubernetes resource configuration is required.",
			Attribute: &att,
		})
		return resp, nil
	}

	rawManifest := make(map[string]tftypes.Value)
	err = manifest.As(&rawManifest)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   `Failed to extract "manifest" attribute value from resource configuration`,
			Detail:    err.Error(),
			Attribute: &att,
		})
		return resp, nil
	}

	for _, key := range requiredKeys {
		if _, present := rawManifest[key]; !present {
			kp := att.WithAttributeName(key)
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   `Attribute key missing from "manifest" value`,
				Detail:    fmt.Sprintf("'%s' attribute key is missing from manifest configuration", key),
				Attribute: &kp,
			})
		}
	}

	for _, key := range forbiddenKeys {
		if _, present := rawManifest[key]; present {
			kp := att.WithAttributeName(key)
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   `Forbidden attribute key in "manifest" value`,
				Detail:    fmt.Sprintf("'%s' attribute key is not allowed in manifest configuration", key),
				Attribute: &kp,
			})
		}
	}

	return resp, nil
}

func (s *RawProviderServer) validateResourceOnline(manifest *tftypes.Value) (diags []*tfprotov5.Diagnostic) {
	rm, err := s.getRestMapper()
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to create K8s RESTMapper client",
			Detail:   err.Error(),
		})
		return
	}
	gvk, err := GVKFromTftypesObject(manifest, rm)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine GroupVersionResource for manifest",
			Detail:   err.Error(),
		})
		return
	}
	// Validate if the resource requires a namespace and fail the plan with
	// a meaningful error if none is supplied. Ideally this would be done earlier,
	// during 'ValidateResourceTypeConfig', but at that point we don't have access to API credentials
	// and we need them for calling IsResourceNamespaced (uses the discovery API).
	ns, err := IsResourceNamespaced(gvk, rm)
	if err != nil {
		diags = append(diags,
			&tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Detail:   err.Error(),
				Summary:  fmt.Sprintf("Failed to discover scope of resource '%s'", gvk.String()),
			})
		return
	}
	nsPath := tftypes.AttributePath{}.WithAttributeName("metadata").WithAttributeName("namespace")
	nsVal, restPath, err := tftypes.WalkAttributePath(*manifest, nsPath)
	if ns {
		if err != nil || len(restPath.Steps) > 0 {
			diags = append(diags,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   fmt.Sprintf("Resources of type '%s' require a namespace", gvk.String()),
					Summary:  "Namespace required",
				})
			return
		}
		if nsVal.(tftypes.Value).IsNull() {
			diags = append(diags,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   fmt.Sprintf("Namespace for resource '%s' cannot be nil", gvk.String()),
					Summary:  "Namespace required",
				})
		}
		var nsStr string
		err := nsVal.(tftypes.Value).As(&nsStr)
		if nsStr == "" && err == nil {
			diags = append(diags,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   fmt.Sprintf("Namespace for resource '%s' cannot be empty", gvk.String()),
					Summary:  "Namespace required",
				})
		}
	} else {
		if err == nil && len(restPath.Steps) == 0 && !nsVal.(tftypes.Value).IsNull() {
			diags = append(diags,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   fmt.Sprintf("Resources of type '%s' cannot have a namespace", gvk.String()),
					Summary:  "Cluster level resource cannot take namespace",
				})
		}
	}
	return
}

func (s *RawProviderServer) validateEmptyBlocksOnline(object *tftypes.Value) (diags []*tfprotov5.Diagnostic) {
	// Validate if the resource contains empty objects and fail the plan with
	// a meaningful error if none is supplied. Empty object will be swawllowed by
	// the API and replaced with a nil value which then confuses Terraform when
	// it tries to save the state after apply. This ensures users don't end up
	// in that situation, by failing the plan step instead.
	if object == nil {
		return
	}
	tftypes.Walk(*object, func(ap tftypes.AttributePath, v tftypes.Value) (bool, error) {
		vt := v.Type()
		switch {
		case vt.Is(tftypes.Map{AttributeType: tftypes.DynamicPseudoType}): // special case for masking usable empty blocks
			return true, nil
		case vt.Is(tftypes.Object{}) || vt.Is(tftypes.Map{}):
			var vals map[string]tftypes.Value
			err := v.As(&vals)
			if err != nil {
				return false, err
			}
			for i := range vals {
				if !vals[i].IsNull() {
					return true, nil
				}
			}
			diags = append(diags,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   fmt.Sprintf(`Unnecessary empty block at "%s"`, ap.String()),
					Summary:  "Resource cannot have empty blocks",
				})
		case vt.Is(tftypes.List{}):
			var vals []tftypes.Value
			err := v.As(&vals)
			if err != nil {
				return false, err
			}
			for i := range vals {
				if !vals[i].IsNull() {
					return true, nil
				}
			}
			diags = append(diags,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   fmt.Sprintf(`Unnecessary empty block at "%s"`, ap.String()),
					Summary:  "Resource cannot have empty blocks",
				})
		}
		return true, nil
	})
	return
}
