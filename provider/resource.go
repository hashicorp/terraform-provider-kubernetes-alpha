package provider

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
)

// ExtractPackedManifest function expands the value of the manifest attribute from a MsgPack plan.
func ExtractPackedManifest(in []byte) (out string, err error) {
	atts := map[string]cty.Type{
		"manifest": cty.String,
		"object":   cty.DynamicPseudoType,
	}

	pptype := cty.Object(atts)

	pplan, err := msgpack.Unmarshal(in, pptype)
	if err != nil {
		Dlog.Printf("[ExtractManifestFromPlan][UnmarshaledPlan] Failed to unmarshal msgpack: %s\n", err.Error())
		return
	}
	if pplan.IsNull() {
		return
	}
	m := pplan.GetAttr("manifest")
	if !m.IsNull() {
		out = m.AsString()
	}
	return
}
