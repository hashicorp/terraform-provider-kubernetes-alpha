package foundry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/hashicorp/go-cty/cty"
	"github.com/mitchellh/hashstructure"
)

// NewFoundryFromSpecV2 creates a new cty.Type foundry from an OpenAPI v2 spec document
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

	f := foapiv2{&swg, make(map[uint64]cty.Type)}
	d := f.swagger.Definitions
	if d == nil || len(d) == 0 {
		return nil, errors.New("spec has no type information")
	}

	return f, nil
}

// Foundry is a mechanism to construct cty.Type types out of OpenAPI specifications
type Foundry interface {
	GetTypeByID(id string) (cty.Type, error)
}

type foapiv2 struct {
	swagger   *openapi2.Swagger
	typeCache map[uint64]cty.Type
}

// GetTypeById looks up a type by its fully qualified ID in the Definitions sections of
// the OpenAPI spec and returns its nearest cty.Type equivalent
func (f foapiv2) GetTypeByID(id string) (cty.Type, error) {
	swd, ok := f.swagger.Definitions[id]

	if !ok {
		return cty.NilType, errors.New("invalid type identifier")
	}

	if swd == nil {
		return cty.NilType, errors.New("invalid type reference - nil")
	}

	sch, err := f.resolveSchemaRef(swd)
	if err != nil {
		return cty.NilType, fmt.Errorf("failed to resolve schema: %s", err)
	}

	return f.getTypeFromSchema(sch)
}

func (f foapiv2) resolveSchemaRef(ref *openapi3.SchemaRef) (*openapi3.Schema, error) {
	if ref.Value != nil {
		return ref.Value, nil
	}

	rp := strings.Split(ref.Ref, "/")
	sid := rp[len(rp)-1]

	switch sid {
	case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
		t := openapi3.Schema{
			Type: "integer",
		}
		return &t, nil
		// case "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps":
		// 	t := openapi3.Schema{
		// 		Type: "string",
		// 	}
		// 	return &t, nil
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

func (f foapiv2) getTypeFromSchema(elem *openapi3.Schema) (cty.Type, error) {
	if elem == nil {
		return cty.NilType, errors.New("nil type")
	}
	h, herr := hashstructure.Hash(elem, nil)

	var t cty.Type

	// check if type is in cache
	if herr == nil {
		if t, ok := f.typeCache[h]; ok {
			return t, nil
		}
	}

	switch elem.Type {

	case "object":

		switch {
		case elem.Properties != nil && elem.AdditionalProperties == nil:
			// this is a standard OpenAPI object
			atts := make(map[string]cty.Type, len(elem.Properties))
			for p, v := range elem.Properties {
				schema, err := f.resolveSchemaRef(v)
				if err != nil {
					return cty.NilType, fmt.Errorf("failed to resolve schema: %s", err)
				}
				pType, err := f.getTypeFromSchema(schema)
				if err != nil {
					return cty.NilType, err
				}
				atts[p] = pType
			}
			t = cty.Object(atts)
			if herr == nil {
				f.typeCache[h] = t
			}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties != nil:
			// this is how OpenAPI defines associative arrays
			s, err := f.resolveSchemaRef(elem.AdditionalProperties)
			if err != nil {
				return cty.NilType, fmt.Errorf("failed to resolve schema: %s", err)
			}
			pt, err := f.getTypeFromSchema(s)
			if err != nil {
				return cty.NilType, err
			}
			t = cty.Map(pt)
			if herr == nil {
				f.typeCache[h] = t
			}
			return t, nil

		case elem.Properties == nil && elem.AdditionalProperties == nil:
			// this is a strange case, encountered with io.k8s.apimachinery.pkg.apis.meta.v1.FieldsV1
			t = cty.DynamicPseudoType
			if herr == nil {
				f.typeCache[h] = t
			}
			return t, nil

		}

	case "array":
		it, err := f.resolveSchemaRef(elem.Items)
		if err != nil {
			return cty.NilType, fmt.Errorf("failed to resolve schema for items: %s", err)
		}
		t, err := f.getTypeFromSchema(it)
		if err != nil {
			return cty.NilType, err
		}
		t = cty.List(t)
		if herr == nil {
			f.typeCache[h] = t
		}
		return t, nil

	case "string":
		return cty.String, nil

	case "boolean":
		return cty.Bool, nil

	case "number":
		return cty.Number, nil

	case "integer":
		return cty.Number, nil

	case "":
		return cty.DynamicPseudoType, nil

	}
	return cty.NilType, fmt.Errorf("unknown type: %s", elem.Type)
}
