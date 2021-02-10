package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
	"github.com/mitchellh/hashstructure"
)

// NewFoundryFromSpecV2 creates a new tftypes.Type foundry from an OpenAPI v2 spec document
// * spec argument should be a valid OpenAPI v2 JSON document
func NewFoundryFromSpecV2(spec []byte) (Foundry, error) {
	if len(spec) < 6 { // unlikely to be valid json
		return nil, errors.New("empty spec")
	}

	var swg openapi2.Swagger
	err := json.Unmarshal(spec, &swg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %s", err)
	}

	f := foapiv2{
		swagger:        &swg,
		typeCache:      make(map[uint64]tftypes.Type),
		recursionDepth: 50, // arbitrarily large number - a type this big will likely kill Terraform anyway
	}

	d := f.swagger.Definitions

	if d == nil || len(d) == 0 {
		return nil, errors.New("spec has no type information")
	}

	return f, nil
}

// Foundry is a mechanism to construct tftype out of OpenAPI specifications
type Foundry interface {
	GetTypeByID(id string) (tftypes.Type, error)
}

type foapiv2 struct {
	swagger        *openapi2.Swagger
	typeCache      map[uint64]tftypes.Type
	recursionDepth uint64
}

// GetTypeById looks up a type by its fully qualified ID in the Definitions sections of
// the OpenAPI spec and returns its nearest tftypes.Type equivalent
func (f foapiv2) GetTypeByID(id string) (tftypes.Type, error) {
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

func (f foapiv2) resolveSchemaRef(ref *openapi3.SchemaRef) (*openapi3.Schema, error) {
	if ref.Value != nil {
		return ref.Value, nil
	}

	rp := strings.Split(ref.Ref, "/")
	sid := rp[len(rp)-1]

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
	}

	nref, ok := f.swagger.Definitions[sid]

	if !ok {
		return nil, errors.New("schema not found")
	}
	if nref == nil {
		return nil, errors.New("nil schema reference")
	}

	return f.resolveSchemaRef(nref)
}

func (f foapiv2) getTypeFromSchema(elem *openapi3.Schema, stackdepth uint64) (tftypes.Type, error) {
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
		if t, ok := f.typeCache[h]; ok {
			return t, nil
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
			f.typeCache[h] = t
		}
		return t, nil

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
				f.typeCache[h] = t
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
				f.typeCache[h] = t
			}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties == nil:
			// this is a strange case, encountered with io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1 and also io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceSubresourceStatus
			t = tftypes.DynamicPseudoType
			if herr == nil {
				f.typeCache[h] = t
			}
			return t, nil

		}
	}

	return nil, fmt.Errorf("unknown type: %s", elem.Type)
}
