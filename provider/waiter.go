package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/go-cty/cty"
	oldcty "github.com/zclconf/go-cty/cty"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	hcl "github.com/hashicorp/hcl/v2"
	hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
)

// Waiter is a simple interface to implement a blocking wait operation
type Waiter interface {
	Wait(context.Context) error
}

// NewResourceWaiter constructs an appropriate Waiter using the supplied waitForBlock configuration
func NewResourceWaiter(resource dynamic.ResourceInterface, resourceName string, resourceType cty.Type, waitForBlock cty.Value) (Waiter, error) {
	fields := waitForBlock.GetAttr("fields")

	if !fields.IsNull() || fields.IsKnown() {
		if !fields.Type().IsMapType() {
			return nil, fmt.Errorf(`"fields" should be a map of strings`)
		}

		vm := fields.AsValueMap()
		matchers := []FieldMatcher{}
		for k, v := range vm {
			expr := v.AsString()
			var re *regexp.Regexp
			if expr == "*" {
				// NOTE this is just a shorthand so the user doesn't have to
				// type the expression below all the time
				re = regexp.MustCompile("(.*)?")
			} else {
				var err error
				re, err = regexp.Compile(expr)
				if err != nil {
					return nil, fmt.Errorf("invalid regular expression: %q", expr)
				}
			}

			p, err := FieldPathToCty(k)
			if err != nil {
				return nil, err
			}
			matchers = append(matchers, FieldMatcher{p, re})
		}

		return &FieldWaiter{
			resource,
			resourceName,
			resourceType,
			matchers,
		}, nil
	}

	return &NoopWaiter{}, nil
}

// FieldMatcher contains a cty Path to a field and a regexp to match on it
type FieldMatcher struct {
	path         cty.Path
	valueMatcher *regexp.Regexp
}

// FieldWaiter will wait for a set of fields to be set,
// or have a particular value
type FieldWaiter struct {
	resource      dynamic.ResourceInterface
	resourceName  string
	resourceType  cty.Type
	fieldMatchers []FieldMatcher
}

// Wait blocks until all of the FieldMatchers configured evaluate to true
func (w *FieldWaiter) Wait(ctx context.Context) error {
	return wait(ctx, w.resource, w.resourceName, w.resourceType, func(obj cty.Value) (bool, error) {
		for _, m := range w.fieldMatchers {
			v, err := m.path.Apply(obj)
			if err != nil {
				return false, err
			}

			var s string
			switch v.Type() {
			case cty.String:
				s = v.AsString()
			case cty.Bool:
				s = fmt.Sprintf("%t", v.True())
			case cty.Number:
				f := v.AsBigFloat()
				if f.IsInt() {
					i, _ := f.Int64()
					s = fmt.Sprintf("%d", i)
				} else {
					i, _ := f.Float64()
					s = fmt.Sprintf("%f", i)
				}
			default:
				return true, fmt.Errorf("wait_for: cannot match on type %q", v.Type().FriendlyName())
			}

			Dlog.Printf("matching %#v %q", m.valueMatcher, s)

			if !m.valueMatcher.Match([]byte(s)) {
				return false, nil
			}
		}

		return true, nil
	})
}

// NoopWaiter is a placeholder for when there is nothing to wait on
type NoopWaiter struct{}

// Wait returns immediately
func (w *NoopWaiter) Wait(_ context.Context) error {
	return nil
}

func wait(ctx context.Context, resource dynamic.ResourceInterface, resourceName string, rtype cty.Type, condition func(cty.Value) (bool, error)) error {
	w, err := resource.Watch(ctx, v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", resourceName).String(),
		Watch:         true,
	})
	if err != nil {
		return err
	}

	Dlog.Printf("[ApplyResourceChange][Wait] Waiting until ready...\n")
	for e := range w.ResultChan() {
		if e.Type == watch.Deleted {
			return fmt.Errorf("resource was deleted")
		}

		if e.Type == watch.Error {
			Dlog.Printf("Error when watching: %#v", e.Object)
			return fmt.Errorf("watch error")
		}

		// NOTE The typed API resource is actually returned in the
		// event object but I haven't yet figured out how to convert it
		// to a cty.Value.
		res, err := resource.Get(ctx, resourceName, v1.GetOptions{})
		if err != nil {
			return err
		}

		obj, err := UnstructuredToCty(res.Object, rtype)
		if err != nil {
			return err
		}

		done, err := condition(obj)
		if done {
			return err
		}
	}

	return nil
}

// FieldPathToCty takes a string representation of
// a path to a field in dot/square bracket notation
// and returns a cty.Path
func FieldPathToCty(fieldPath string) (cty.Path, error) {
	t, d := hclsyntax.ParseTraversalAbs([]byte(fieldPath), "", hcl.Pos{Line: 1, Column: 1})
	if d.HasErrors() {
		return nil, fmt.Errorf("invalid field path %q: %s: %s", fieldPath, d[0].Summary, d[0].Detail)
	}

	path := cty.Path{}
	for _, p := range t {
		switch p.(type) {
		case hcl.TraverseRoot:
			path = path.GetAttr(p.(hcl.TraverseRoot).Name)
		case hcl.TraverseIndex:
			indexKey := p.(hcl.TraverseIndex).Key
			indexKeyType := indexKey.Type()
			if indexKeyType.Equals(oldcty.String) {
				path = path.GetAttr(indexKey.AsString())
			} else if indexKeyType.Equals(oldcty.Number) {
				f := indexKey.AsBigFloat()
				if f.IsInt() {
					i, _ := f.Int64()
					path = path.IndexInt(int(i))
				} else {
					return nil, fmt.Errorf("index in field path must be an integer")
				}
			} else {
				return nil, fmt.Errorf("unsupported type in field path: %s", indexKeyType.FriendlyName())
			}
		case hcl.TraverseAttr:
			path = path.GetAttr(p.(hcl.TraverseAttr).Name)
		case hcl.TraverseSplat:
			return nil, fmt.Errorf("splat is not supported")
		}
	}

	return path, nil
}
