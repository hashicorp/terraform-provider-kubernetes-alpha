package tftypes

// these functions are based heavily on github.com/zclconf/go-cty
// used under the MIT License
//
// Copyright (c) 2017-2018 Martin Atkins
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"errors"
)

func Walk(val Value, cb func(AttributePath, Value) (bool, error)) error {
	var path AttributePath
	return walk(path, val, cb)
}

func walk(path AttributePath, val Value, cb func(AttributePath, Value) (bool, error)) error {
	shouldContinue, err := cb(path, val)
	if err != nil {
		var ape attributePathError
		if !errors.As(err, &ape) {
			err = path.NewError(err)
		}
		return err
	}
	if !shouldContinue {
		return nil
	}

	if val.IsNull() || !val.IsKnown() {
		return nil
	}

	// TODO(paddy): replace with val.Type() once #58 lands
	ty := val.typ
	switch {
	case ty.Is(List{}), ty.Is(Set{}), ty.Is(Tuple{}):
		var v []Value
		err := val.As(&v)
		if err != nil {
			// should never happen
			return path.NewError(err)
		}
		for pos, el := range v {
			if ty.Is(Set{}) {
				path = path.WithElementKeyValue(el)
			} else {
				path = path.WithElementKeyInt(int64(pos))
			}
			err = walk(path, el, cb)
			if err != nil {
				var ape attributePathError
				if !errors.As(err, &ape) {
					err = path.NewError(err)
				}
				return err
			}
			path = path.WithoutLastStep()
		}
	case ty.Is(Map{}), ty.Is(Object{}):
		v := map[string]Value{}
		err := val.As(&v)
		if err != nil {
			// should never happen
			return err
		}
		for k, el := range v {
			if ty.Is(Map{}) {
				path = path.WithElementKeyString(k)
			} else if ty.Is(Object{}) {
				path = path.WithAttributeName(k)
			}
			err = walk(path, el, cb)
			if err != nil {
				var ape attributePathError
				if !errors.As(err, &ape) {
					err = path.NewError(err)
				}
				return err
			}
			path = path.WithoutLastStep()
		}
	}

	return nil
}

func Transform(val Value, cb func(AttributePath, Value) (Value, error)) (Value, error) {
	var path AttributePath
	return transform(path, val, cb)
}

func transform(path AttributePath, val Value, cb func(AttributePath, Value) (Value, error)) (Value, error) {
	var newVal Value
	// TODO(paddy): change this to val.Type() when #58 lands
	ty := val.typ

	switch {
	case val.IsNull() || !val.IsKnown():
		newVal = val
	case ty.Is(List{}), ty.Is(Set{}), ty.Is(Tuple{}):
		var v []Value
		err := val.As(&v)
		if err != nil {
			return val, err
		}
		if len(v) == 0 {
			newVal = val
		} else {
			elems := make([]Value, 0, len(v))
			for pos, el := range v {
				if ty.Is(Set{}) {
					path = path.WithElementKeyValue(el)
				} else {
					path = path.WithElementKeyInt(int64(pos))
				}
				newEl, err := transform(path, el, cb)
				if err != nil {
					var ape attributePathError
					if !errors.As(err, &ape) {
						err = path.NewError(err)
					}
					return val, err
				}
				elems = append(elems, newEl)
				path = path.WithoutLastStep()
			}
			newVal = NewValue(ty, elems)
		}
	case ty.Is(Map{}), ty.Is(Object{}):
		v := map[string]Value{}
		err := val.As(&v)
		if err != nil {
			return val, err
		}
		if len(v) == 0 {
			newVal = val
		} else {
			elems := map[string]Value{}
			for k, el := range v {
				if ty.Is(Map{}) {
					path = path.WithElementKeyString(k)
				} else {
					path = path.WithAttributeName(k)
				}
				newEl, err := transform(path, el, cb)
				if err != nil {
					var ape attributePathError
					if !errors.As(err, &ape) {
						err = path.NewError(err)
					}
					return val, err
				}
				elems[k] = newEl
				path = path.WithoutLastStep()
			}
			newVal = NewValue(ty, elems)
		}
	default:
		newVal = val
	}
	res, err := cb(path, newVal)
	if err != nil {
		var ape attributePathError
		if !errors.As(err, &ape) {
			err = path.NewError(err)
		}
		return res, err
	}
	if !newVal.Type().Is(ty) {
		return val, path.NewError(errors.New("invalid transform: value changed type"))
	}
	return res, err
}
