package provider

/*
import (
	"github.com/hashicorp/go-cty/cty"
	proto "github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
)

// AppendProtoDiag appends a new diagnostic from a warning string or an error.
// This panics if d is not a string or error.
func AppendProtoDiag(diags []*proto.Diagnostic, d interface{}) []*proto.Diagnostic {
	switch d := d.(type) {
	case cty.PathError:
		ap := PathToAttributePath(d.Path)
		diags = append(diags, &proto.Diagnostic{
			Severity:  proto.Diagnostic_ERROR,
			Summary:   d.Error(),
			Attribute: ap,
		})
	case error:
		diags = append(diags, &proto.Diagnostic{
			Severity: proto.Diagnostic_ERROR,
			Summary:  d.Error(),
		})
	case string:
		diags = append(diags, &proto.Diagnostic{
			Severity: proto.Diagnostic_WARNING,
			Summary:  d,
		})
	case *proto.Diagnostic:
		diags = append(diags, d)
	case []*proto.Diagnostic:
		diags = append(diags, d...)
	}
	return diags
}

// PathToAttributePath takes a cty.Path and converts it to a proto-encoded path.
func PathToAttributePath(p cty.Path) *proto.AttributePath {
	ap := &proto.AttributePath{}
	for _, step := range p {
		switch selector := step.(type) {
		case cty.GetAttrStep:
			ap.Steps = append(ap.Steps, &proto.AttributePath_Step{
				Selector: &proto.AttributePath_Step_AttributeName{
					AttributeName: selector.Name,
				},
			})
		case cty.IndexStep:
			key := selector.Key
			switch key.Type() {
			case cty.String:
				ap.Steps = append(ap.Steps, &proto.AttributePath_Step{
					Selector: &proto.AttributePath_Step_ElementKeyString{
						ElementKeyString: key.AsString(),
					},
				})
			case cty.Number:
				v, _ := key.AsBigFloat().Int64()
				ap.Steps = append(ap.Steps, &proto.AttributePath_Step{
					Selector: &proto.AttributePath_Step_ElementKeyInt{
						ElementKeyInt: v,
					},
				})
			default:
				// We'll bail early if we encounter anything else, and just
				// return the valid prefix.
				return ap
			}
		}
	}
	return ap
}
*/
