package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// PlanUpdateResourceHCL decides whether to off-load the change planning
// to the API server via a dry-run call or compute the changes locally
func PlanUpdateResourceHCL(ctx context.Context, in *cty.Value) (cty.Value, error) {
	s := GetGlobalState()
	if s[SSPlanning].(bool) {
		return PlanUpdateResourceHCLServerSide(ctx, in)
	}
	return PlanUpdateResourceHCLLocal(ctx, in)
}

// PlanUpdateResourceHCLLocal calculates the state for a new resource based on HCL manifest
func PlanUpdateResourceHCLLocal(ctx context.Context, plan *cty.Value) (cty.Value, error) {
	m := plan.GetAttr("manifest")

	oapi, err := GetOAPIFoundry()
	if err != nil {
		return cty.NilVal, err
	}

	gvk, err := GVKFromCtyObject(&m)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to determine resource GVR: %s", err)
	}

	id, err := OpenAPIPathFromGVK(gvk)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	tsch, err := oapi.GetTypeByID(id)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to get resource type from OpenAPI (ID %s): %s", id, err)
	}

	// Generate the scaffold resource value with all attributes set to cty.UnknownVal
	tobj := DeepUnknownVal(tsch)

	// Fill in the attributes provided in HCL configuration into the scaffold
	nobj, err := cty.Transform(tobj, func(p cty.Path, v cty.Value) (cty.Value, error) {
		nv, err := p.Apply(m)
		if err != nil {
			return v, nil
		}
		return nv, nil
	})

	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to transform object based on OpenAPI: %s", err)
	}

	if plan.GetAttr("object").IsNull() { // plan for Create
		nc, err := cty.Transform(*plan, ResourceBulkUpdateObjectAttr(&nobj))
		if err != nil {
			return cty.NilVal, err
		}
		return nc, nil
	}

	nc, err := cty.Transform(*plan,
		ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object"), &nobj),
	)

	if err != nil {
		return cty.NilVal, err
	}
	return nc, nil
}

// PlanUpdateResourceHCLServerSide calculates the state for a new resource based on HCL manifest
func PlanUpdateResourceHCLServerSide(ctx context.Context, p *cty.Value) (cty.Value, error) {
	m := p.GetAttr("manifest")
	co, err := CtyObjectToUnstructured(&m)
	if err != nil {
		return cty.NilVal, err
	}
	uo := unstructured.Unstructured{Object: co}

	gvr, err := GVRFromCtyUnstructured(&uo)
	if err != nil {
		return cty.NilVal, err
	}

	ns, err := IsResourceNamespaced(gvr)
	if err != nil {
		return cty.NilVal, err
	}

	rnamespace := uo.GetNamespace()
	rname := uo.GetName()

	c, err := GetDynamicClient()
	if err != nil {
		return cty.NilVal, err
	}

	var r dynamic.ResourceInterface

	if ns {
		r = c.Resource(gvr).Namespace(rnamespace)
	} else {
		r = c.Resource(gvr)
	}

	js, err := uo.MarshalJSON()
	if err != nil {
		return cty.NilVal, err
	}

	ro, err := r.Patch(ctx,
		rname,
		types.ApplyPatchType,
		js,
		v1.PatchOptions{
			DryRun:       []string{v1.DryRunAll},
			FieldManager: "Terraform",
		})
	if err != nil {
		rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
		return cty.NilVal, fmt.Errorf("update dry-run for '%s' failed: %s", rn, err)
	}

	rc, err := UnstructuredToCty(FilterEphemeralFields(ro.Object))
	if err != nil {
		return cty.NilVal, err
	}
	np, err := cty.Transform(*p, ResourceBulkUpdateObjectAttr(&rc))
	if err != nil {
		return cty.NilVal, err
	}
	return np, nil
}

// FilterEphemeralFields removes certain fields from an API response object
// which would otherwise cause a false diff
func FilterEphemeralFields(in map[string]interface{}) map[string]interface{} {
	// Remove "status" attribute
	delete(in, "status")

	meta := in["metadata"].(map[string]interface{})

	// Remove "uid", "creationTimestamp", "resourceVersion" as
	// they change with most resource operations
	delete(meta, "uid")
	delete(meta, "creationTimestamp")
	delete(meta, "resourceVersion")
	delete(meta, "generation")
	delete(meta, "selfLink")

	// TODO: we should be filtering API responses based on the contents of 'managedFields'
	// and only retain the attributes for which the manager is Terraform
	delete(meta, "managedFields")

	return in
}
