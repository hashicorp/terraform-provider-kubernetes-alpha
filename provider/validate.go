package provider

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ValidateResourceTypeConfig function
func (s *RawProviderServer) ValidateResourceTypeConfig(ctx context.Context, req *tfprotov5.ValidateResourceTypeConfigRequest) (*tfprotov5.ValidateResourceTypeConfigResponse, error) {
	resp := &tfprotov5.ValidateResourceTypeConfigResponse{}
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
		att := tftypes.AttributePath{}.WithAttributeName("manifest")
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Manifest missing from resource configuration",
			Detail:    "A manifest attribute containing a valid Kubernetes resource configuration is required.",
			Attribute: &att,
		})
		return resp, nil
	}
	manObj, err := TFValueToUnstructured(manifest, tftypes.AttributePath{})
	if err != nil {
		att := tftypes.AttributePath{}.WithAttributeName("manifest")
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   `Invalid "manifest" attribute`,
			Detail:    err.Error(),
			Attribute: &att,
		})
		return resp, nil
	}
	manu := unstructured.Unstructured{Object: manObj.(map[string]interface{})}

	s.logger.Trace("[ValidateResourceTypeConfig]", "[Manifest]", spew.Sdump(manu))

	return resp, nil
}
