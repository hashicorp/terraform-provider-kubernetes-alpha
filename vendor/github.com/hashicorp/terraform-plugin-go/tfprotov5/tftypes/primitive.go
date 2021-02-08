package tftypes

import "fmt"

var (
	// DynamicPseudoType is a pseudo-type in Terraform's type system that
	// is used as a wildcard type. It indicates that any Terraform type can
	// be used.
	DynamicPseudoType = primitive{name: "DynamicPseudoType"}

	// String is a primitive type in Terraform that represents a UTF-8
	// string of bytes.
	String = primitive{name: "String"}

	// Number is a primitive type in Terraform that represents a real
	// number.
	Number = primitive{name: "Number"}

	// Bool is a primitive type in Terraform that represents a true or
	// false boolean value.
	Bool = primitive{name: "Bool"}
)

var (
	_ Type = primitive{name: "test"}
)

type primitive struct {
	_    []struct{}
	name string
}

func (p primitive) Is(t Type) bool {
	v, ok := t.(primitive)
	if !ok {
		return false
	}
	return p.name == v.name
}

func (p primitive) String() string {
	return "tftypes." + string(p.name)
}

func (p primitive) private() {}

func (p primitive) MarshalJSON() ([]byte, error) {
	switch p.name {
	case String.name:
		return []byte(`"string"`), nil
	case Number.name:
		return []byte(`"number"`), nil
	case Bool.name:
		return []byte(`"bool"`), nil
	case DynamicPseudoType.name:
		return []byte(`"dynamic"`), nil
	}
	return nil, fmt.Errorf("unknown primitive type %q", p)
}
