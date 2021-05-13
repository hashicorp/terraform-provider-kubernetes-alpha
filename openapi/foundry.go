package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/mitchellh/hashstructure"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewFoundryFromSpecV2 creates a new tftypes.Type foundry from an OpenAPI v2 spec document
// * spec argument should be a valid OpenAPI v2 JSON document
func NewFoundryFromSpecV2(spec []byte) (Foundry, error) {
	if len(spec) < 6 { // unlikely to be valid json
		return nil, errors.New("empty spec")
	}

	var swg openapi2.T
	err := swg.UnmarshalJSON(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %s", err)
	}

	d := swg.Definitions
	if d == nil || len(d) == 0 {
		return nil, errors.New("spec has no type information")
	}

	f := foapiv2{
		swagger:        &swg,
		typeCache:      sync.Map{},
		gkvIndex:       sync.Map{}, //reverse lookup index from GVK to OpenAPI definition IDs
		recursionDepth: 50,         // arbitrarily large number - a type this deep will likely kill Terraform anyway
		gate:           sync.Mutex{},
	}

	err = f.buildGvkIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to build GVK index when creating new foundry: %s", err)
	}

	return &f, nil
}

// Foundry is a mechanism to construct tftypes out of OpenAPI specifications
type Foundry interface {
	GetTypeByGVK(gvk schema.GroupVersionKind) (tftypes.Type, error)
}

type foapiv2 struct {
	swagger        *openapi2.T
	typeCache      sync.Map
	gkvIndex       sync.Map
	recursionDepth uint64 // a last resort circuit-breaker for run-away recursion - hitting this will make for a bad day
	gate           sync.Mutex
}

// GetTypeByGVK looks up a type by its GVK in the Definitions sections of
// the OpenAPI spec and returns its (nearest) tftypes.Type equivalent
func (f *foapiv2) GetTypeByGVK(gvk schema.GroupVersionKind) (tftypes.Type, error) {
	// the ID string that Swagger / OpenAPI uses to identify the resource
	// e.g. "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
	f.gate.Lock()
	defer f.gate.Unlock()

	id, ok := f.gkvIndex.Load(gvk)
	if !ok {
		return nil, fmt.Errorf("%v resource not found in OpenAPI index", gvk)
	}
	return f.getTypeByID(id.(string))
}

func (f *foapiv2) getTypeByID(id string) (tftypes.Type, error) {
	swd, ok := f.swagger.Definitions[id]

	if !ok {
		return nil, errors.New("invalid type identifier")
	}

	if swd == nil {
		return nil, errors.New("invalid type reference (nil)")
	}

	sch, err := f.resolveSchemaRef(swd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve schema: %s", err)
	}

	return f.getTypeFromSchema(sch, 0)
}

func (f *foapiv2) resolveSchemaRef(ref *openapi3.SchemaRef) (*openapi3.Schema, error) {
	if ref.Value != nil {
		return ref.Value, nil
	}

	rp := strings.Split(ref.Ref, "/")
	sid := rp[len(rp)-1]

	nref, ok := f.swagger.Definitions[sid]

	if !ok {
		return nil, errors.New("schema not found")
	}
	if nref == nil {
		return nil, errors.New("nil schema reference")
	}

	// These are exceptional situations that require non-standard types.
	switch sid {
	case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
		t := openapi3.Schema{
			Type: "",
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps":
		t := openapi3.Schema{
			Type: "",
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.JSONSchemaProps":
		t := openapi3.Schema{
			Type: "",
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.CustomResourceSubresourceStatus":
		t := openapi3.Schema{
			Type: "object",
			AdditionalProperties: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus":
		t := openapi3.Schema{
			Type: "object",
			AdditionalProperties: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		}
		return &t, nil
	case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceDefinitionSpec":
		t, err := f.resolveSchemaRef(nref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve schema: %s", err)
		}
		vs := t.Properties["versions"]
		vs.Value.AdditionalProperties = vs.Value.Items
		vs.Value.Items = nil
		return t, nil
	}

	return f.resolveSchemaRef(nref)
}

func (f *foapiv2) getTypeFromSchema(elem *openapi3.Schema, stackdepth uint64) (tftypes.Type, error) {
	if stackdepth > f.recursionDepth {
		// this is a hack to overcome the inability to express recursion in tftypes
		return nil, errors.New("recursion runaway while generating type from OpenAPI spec")
	}

	if elem == nil {
		return nil, errors.New("cannot convert OpenAPI type (nil)")
	}

	h, herr := hashstructure.Hash(elem, nil)

	var t tftypes.Type

	// check if type is in cache
	if herr == nil {
		if t, ok := f.typeCache.Load(h); ok {
			return t.(tftypes.Type), nil
		}
	}
	switch elem.Type {
	case "string":
		return tftypes.String, nil

	case "boolean":
		return tftypes.Bool, nil

	case "number":
		return tftypes.Number, nil

	case "integer":
		return tftypes.Number, nil

	case "":
		return tftypes.DynamicPseudoType, nil

	case "array":
		switch {
		case elem.Items != nil && elem.AdditionalProperties == nil: // normal array - translates to a tftypes.List
			it, err := f.resolveSchemaRef(elem.Items)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			et, err := f.getTypeFromSchema(it, stackdepth+1)
			if err != nil {
				return nil, err
			}
			t = tftypes.List{ElementType: et}
			if herr == nil {
				f.typeCache.Store(h, t)
			}
			return t, nil
		case elem.AdditionalProperties != nil && elem.Items == nil: // "overriden" array - translates to a tftypes.List
			it, err := f.resolveSchemaRef(elem.AdditionalProperties)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema for items: %s", err)
			}
			et, err := f.getTypeFromSchema(it, stackdepth+1)
			if err != nil {
				return nil, err
			}
			t = tftypes.Tuple{ElementTypes: []tftypes.Type{et}}
			if herr == nil {
				f.typeCache.Store(h, t)
			}
			return t, nil
		}

	case "object":

		switch {
		case elem.Properties != nil && elem.AdditionalProperties == nil:
			// this is a standard OpenAPI object
			atts := make(map[string]tftypes.Type, len(elem.Properties))
			for p, v := range elem.Properties {
				schema, err := f.resolveSchemaRef(v)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve schema: %s", err)
				}
				pType, err := f.getTypeFromSchema(schema, stackdepth+1)
				if err != nil {
					return nil, err
				}
				atts[p] = pType
			}
			t = tftypes.Object{AttributeTypes: atts}
			if herr == nil {
				f.typeCache.Store(h, t)
			}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties != nil:
			// this is how OpenAPI defines associative arrays
			s, err := f.resolveSchemaRef(elem.AdditionalProperties)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema: %s", err)
			}
			pt, err := f.getTypeFromSchema(s, stackdepth+1)
			if err != nil {
				return nil, err
			}
			t = tftypes.Map{AttributeType: pt}
			if herr == nil {
				f.typeCache.Store(h, t)
			}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties == nil:
			// this is a strange case, encountered with io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1 and also io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus
			t = tftypes.DynamicPseudoType
			if herr == nil {
				f.typeCache.Store(h, t)
			}
			return t, nil

		}
	}

	return nil, fmt.Errorf("unknown type: %s", elem.Type)
}

// buildGvkIndex builds the reverse lookup index that associates each GVK
// to its corresponding string key in the swagger.Definitions map
func (f *foapiv2) buildGvkIndex() error {
	for did, dRef := range f.swagger.Definitions {
		def, err := f.resolveSchemaRef(dRef)
		if err != nil {
			return err
		}
		ex, ok := def.Extensions["x-kubernetes-group-version-kind"]
		if !ok {
			continue
		}
		gvk := []schema.GroupVersionKind{}
		gvkRaw, err := ex.(json.RawMessage).MarshalJSON()
		err = json.Unmarshal(gvkRaw, &gvk)
		if err != nil {
			return fmt.Errorf("failed to unmarshall GVK from OpenAPI schema extention: %v", err)
		}
		for i := range gvk {
			f.gkvIndex.Store(gvk[i], did)
		}
	}
	return nil
}
