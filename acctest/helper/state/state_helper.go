package state

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	tfjson "github.com/hashicorp/terraform-json"
)

// Helper wraps a tfjson.State object with some helper functions that use
// github.com/stretchr/testify/assert to reduce boilerplate in test code
type Helper struct {
	t     *testing.T
	state *tfjson.State
}

func getAttributesFromState(state *tfjson.State, resourceAddr string) (interface{}, error) {
	for _, r := range state.Values.RootModule.Resources {
		if r.Address == resourceAddr {
			return r.AttributeValues, nil
		}
	}
	return nil, fmt.Errorf("Could not find resource %q in state", resourceAddr)
}

var errFieldNotFound = fmt.Errorf("Field not found")

// findAttributeValue will return the value of the attribute at the given address in a tree of arrays and maps
func findAttributeValue(in interface{}, address string) (interface{}, error) {
	keys := strings.Split(address, ".")
	key := keys[0]

	var value interface{}
	if index, err := strconv.Atoi(key); err == nil {
		s, ok := in.([]interface{})
		if !ok || index >= len(s) {
			return nil, errFieldNotFound
		}
		value = s[index]
	} else {
		m, ok := in.(map[string]interface{})
		if !ok {
			return nil, errFieldNotFound
		}
		v, ok := m[key]
		if !ok {
			return nil, errFieldNotFound
		}
		value = v
	}

	if len(keys) == 1 {
		return value, nil
	}

	return findAttributeValue(value, strings.Join(keys[1:], "."))
}

// Wrap will wrap a *tfjson.State object with helper functions
func Wrap(t *testing.T, state *tfjson.State) *Helper {
	return &Helper{
		t:     t,
		state: state,
	}
}

// parseStateAddress will parse an address using the same format as `terraform state show`
// and return the resource address (resource_type.name) and attribute address (attribute.subattribute)
func parseStateAddress(address string) (string, string) {
	parts := strings.Split(address, ".")

	var resourceAddress, attributeAddress string
	switch parts[0] {
	case "data":
		resourceAddress = strings.Join(parts[0:3], ".")
		attributeAddress = strings.Join(parts[3:len(parts)], ".")
	default:
		resourceAddress = strings.Join(parts[0:2], ".")
		attributeAddress = strings.Join(parts[2:len(parts)], ".")
	}

	return resourceAddress, attributeAddress
}

// getAttributeValue will get the value at the given address from the state
// using the same format as `terraform state show`
func (s *Helper) getAttributeValue(address string) interface{} {
	resourceAddress, attributeAddress := parseStateAddress(address)
	attrs, err := getAttributesFromState(s.state, resourceAddress)
	if err != nil {
		s.t.Fatal(err)
	}

	value, err := findAttributeValue(attrs, attributeAddress)
	if err != nil {
		s.t.Fatalf("%q does not exist", address)
	}

	return value
}

// AssertAttributeEqual will fail the test if the attribute does not equal expectedValue
func (s *Helper) AssertAttributeEqual(address string, expectedValue interface{}) {
	s.t.Helper()
	assert.Equal(s.t, expectedValue, s.getAttributeValue(address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeNotEqual will fail the test if the attribute is equal to expectedValue
func (s *Helper) AssertAttributeNotEqual(address string, expectedValue interface{}) {
	s.t.Helper()
	assert.NotEqual(s.t, expectedValue, s.getAttributeValue(address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeExists will fail the test if the attribute does not exist
func (s *Helper) AssertAttributeExists(address string) {
	s.t.Helper()
	s.getAttributeValue(address)
}

// AssertAttributeDoesNotExist will fail the test if the attribute exists
func (s *Helper) AssertAttributeDoesNotExist(address string) {
	s.t.Helper()

	resourceAddress, attributeAddress := parseStateAddress(address)
	attrs, err := getAttributesFromState(s.state, resourceAddress)
	if err != nil {
		s.t.Fatal(err)
	}

	_, err = findAttributeValue(attrs, attributeAddress)
	if err == nil {
		s.t.Fatalf("%q exists", address)
	}
}

// AssertAttributeNotEmpty will fail the test if the attribute is not empty
func (s *Helper) AssertAttributeNotEmpty(address string) {
	s.t.Helper()
	assert.NotEmpty(s.t, s.getAttributeValue(address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeEmpty will fail the test if the attribute is empty
func (s *Helper) AssertAttributeEmpty(address string) {
	s.t.Helper()
	assert.NotEmpty(s.t, s.getAttributeValue(address),
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeLen will fail the test if the length of the attribute is not exactly length
func (s *Helper) AssertAttributeLen(address string, length int) {
	s.t.Helper()
	assert.Len(s.t, s.getAttributeValue(address), length,
		fmt.Sprintf("Address: %q", address))
}

// AssertAttributeTrue will fail the test if the attribute is not true
func (s *Helper) AssertAttributeTrue(address string) {
	s.t.Helper()
	v, ok := s.getAttributeValue(address).(bool)
	if !ok {
		s.t.Errorf("%q is not a bool", address)
	} else {
		assert.True(s.t, v, fmt.Sprintf("Address: %q", address))
	}
}

// AssertAttributeFalse will fail the test if the attribute is not false
func (s *Helper) AssertAttributeFalse(address string) {
	s.t.Helper()
	v, ok := s.getAttributeValue(address).(bool)
	if !ok {
		s.t.Errorf("%q is not a bool", address)
	} else {
		assert.False(s.t, v, fmt.Sprintf("Address: %q", address))
	}
}
