package tftypes

import (
	"encoding/json"
	"sort"
	"strings"
)

// Object is a Terraform type representing an unordered collection of
// attributes, potentially of differing types, each identifiable with a unique
// string name. The number of attributes, their names, and their types are part
// of the type signature for the Object, and so two Objects with different
// attribute names or types are considered to be distinct types.
type Object struct {
	AttributeTypes map[string]Type

	// used to make this type uncomparable
	// see https://golang.org/ref/spec#Comparison_operators
	// this enforces the use of Is, instead
	_ []struct{}
}

// Is returns whether `t` is an Object type or not. If `t` is an instance of
// the Object type and its AttributeTypes property is not nil, it will only
// return true the AttributeTypes are considered the same. To be considered
// equal, the same set of keys must be present in each, and each key's value
// needs to be considered the same type between the two Objects.
func (o Object) Is(t Type) bool {
	v, ok := t.(Object)
	if !ok {
		return false
	}
	if v.AttributeTypes != nil {
		if len(o.AttributeTypes) != len(v.AttributeTypes) {
			return false
		}
		for k, typ := range o.AttributeTypes {
			if _, ok := v.AttributeTypes[k]; !ok {
				return false
			}
			if !typ.Is(v.AttributeTypes[k]) {
				return false
			}
		}
	}
	return ok
}

func (o Object) String() string {
	var res strings.Builder
	res.WriteString("tftypes.Object[")
	keys := make([]string, 0, len(o.AttributeTypes))
	for k := range o.AttributeTypes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for pos, key := range keys {
		if pos != 0 {
			res.WriteString(", ")
		}
		res.WriteString(`"` + key + `":`)
		res.WriteString(o.AttributeTypes[key].String())
	}
	res.WriteString("]")
	return res.String()
}

func (o Object) private() {}

// MarshalJSON returns a JSON representation of the full type signature of `o`,
// including the AttributeTypes.
//
// Deprecated: this is not meant to be called by third-party code.
func (o Object) MarshalJSON() ([]byte, error) {
	attrs, err := json.Marshal(o.AttributeTypes)
	if err != nil {
		return nil, err
	}
	return []byte(`["object",` + string(attrs) + `]`), nil
}
