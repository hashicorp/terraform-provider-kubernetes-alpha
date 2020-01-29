package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alexsomesan/terraform-provider-kubedynamic/tfplugin5"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	install.Install(scheme.Scheme)
}

// RawProviderServer implements the ProviderServer interface as exported from ProtoBuf.
type RawProviderServer struct{}

// GetSchema function
func (s *RawProviderServer) GetSchema(ctx context.Context, req *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	//	Dlog.Printf("[GetSchema][Request]\n%s\n", spew.Sdump(*req))

	resp := &tfplugin5.GetProviderSchema_Response{
		Provider:        GetProviderConfigSchema(),
		ResourceSchemas: GetProviderResourceSchema(),
	}
	return resp, nil
}

// PrepareProviderConfig function
func (s *RawProviderServer) PrepareProviderConfig(ctx context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	resp := &tfplugin5.PrepareProviderConfig_Response{}

	// config, err := msgpack.Unmarshal(req.Config.Msgpack, GetConfigObjectType())
	//// Dlog.Printf("[PrepareProviderConfig][Request][Config]\n%s\n", spew.Sdump(config))
	// if err != nil {
	// 	return resp, err
	// }

	return resp, nil
}

// ValidateResourceTypeConfig function
func (s *RawProviderServer) ValidateResourceTypeConfig(ctx context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	//	Dlog.Printf("[ValidateResourceTypeConfig][Request]\n%s\n", spew.Sdump(*req))

	config := &tfplugin5.ValidateResourceTypeConfig_Response{}
	return config, nil
}

// ValidateDataSourceConfig function
func (s *RawProviderServer) ValidateDataSourceConfig(ctx context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	//	Dlog.Printf("[ValidateDataSourceConfig][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ValidateDataSourceConfig not implemented")
}

// UpgradeResourceState function
func (s *RawProviderServer) UpgradeResourceState(ctx context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	resp := &tfplugin5.UpgradeResourceState_Response{}
	resp.Diagnostics = []*tfplugin5.Diagnostic{}
	jsonMap := map[string]interface{}{}

	sch := GetProviderResourceSchema()
	rt, err := GetObjectTypeFromSchema(sch[req.TypeName])
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(req.RawState.Json, &jsonMap)
	if err != nil {
		resp.Diagnostics = AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	val, err := UnstructuredToCty(jsonMap)
	if err != nil {
		resp.Diagnostics = AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	newStateMP, err := msgpack.Marshal(val, rt)
	if err != nil {
		resp.Diagnostics = AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.UpgradedState = &tfplugin5.DynamicValue{Msgpack: newStateMP}
	return resp, nil
}

// Configure function
func (s *RawProviderServer) Configure(ctx context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	//	Dlog.Printf("[Configure][Request]\n%s\n", spew.Sdump(*req))
	response := &tfplugin5.Configure_Response{}

	providerConfig, err := msgpack.Unmarshal(req.Config.Msgpack, GetConfigObjectType())
	if err != nil {
		return response, err
	}

	configFile := providerConfig.GetAttr("config_file")
	var kubeconfig string

	// if no config specified, try the known default locations
	if configFile.IsNull() || configFile.AsString() == "" {
		h := os.Getenv("HOME")
		if h == "" {
			h = os.Getenv("USERPROFILE") // windows
		}
		if h == "" {
			err := fmt.Errorf("cannot determine HOME path")
			//			Dlog.Printf("[Configure][Kubeconfig] %v.\n", err)
			return response, err
		}
		kubeconfig = filepath.Join(h, ".kube", "config")
	} else {
		kubeconfig = configFile.AsString()
	}

	var clientConfig *rest.Config
	clientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		err = fmt.Errorf("cannot load Kubernetes client config from %s: %s", kubeconfig, err)
		//		Dlog.Printf("[Configure][Kubeconfig] %s.\n", err.Error())
		return response, err
	}
	if logging.IsDebugOrHigher() {
		clientConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return logging.NewTransport("Kubernetes", rt)
		}
	}

	dynClient, errClient := dynamic.NewForConfig(clientConfig)
	if errClient != nil {
		//		Dlog.Printf("[Configure] Error creating dynamic client %v", errClient)
		return response, errClient
	}

	discoClient, errClient := discovery.NewDiscoveryClientForConfig(clientConfig)
	if errClient != nil {
		//		Dlog.Printf("[Configure] Error creating discovery client %v", errClient)
		return response, errClient
	}

	cacher := memory.NewMemCacheClient(discoClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cacher)

	ps := GetProviderState()
	ps[DynamicClient] = dynClient
	ps[DiscoveryClient] = discoClient
	ps[RestMapper] = mapper

	//	Dlog.Printf("[Configure] Successfully created dicovery client.\n")

	return response, nil
}

// ReadResource function
func (s *RawProviderServer) ReadResource(ctx context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	resp := &tfplugin5.ReadResource_Response{}

	currentState, err := UnmarshalResource(req.TypeName, req.GetCurrentState().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from current state: %#v", err)
	}
	if !currentState.Type().HasAttribute("object") {
		return resp, fmt.Errorf("existing state of resource %s has no 'object' attribute", req.TypeName)
	}

	co := currentState.GetAttr("object").GetAttr("value")

	mo, err := CtyObjectToUnstructured(&co)
	if err != nil {
		return resp, fmt.Errorf("failed to convert current state to unstructured: %s", err)
	}
	cu := unstructured.Unstructured{Object: mo}

	ns := cu.GetNamespace()
	rname := cu.GetName()

	client, err := GetDynamicClient()
	if err != nil {
		return resp, err
	}

	cGVR, err := GVRFromCtyUnstructured(&cu)
	if err != nil {
		return resp, err
	}

	rcl := client.Resource(cGVR)

	var fo *unstructured.Unstructured
	if len(ns) > 0 {
		fo, err = rcl.Namespace(ns).Get(rname, v1.GetOptions{})
	} else {
		fo, err = rcl.Get(rname, v1.GetOptions{})
	}
	Dlog.Printf("[ReadResource][Request][API-GET] %s\n", spew.Sdump(*fo))

	// remove status from result, so we don't store it in the state
	delete(fo.Object, "status")
	nobj, err := UnstructuredToCty(fo.Object)
	if err != nil {
		return resp, err
	}
	newstate, err := cty.Transform(currentState, ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object").GetAttr("value"), &nobj))
	if err != nil {
		return resp, err
	}
	newStatePacked, err := MarshalResource(req.TypeName, &newstate)
	if err != nil {
		return resp, err
	}
	resp.NewState = &tfplugin5.DynamicValue{Msgpack: newStatePacked}
	return resp, nil
}

// PlanResourceChange function
func (s *RawProviderServer) PlanResourceChange(ctx context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	resp := &tfplugin5.PlanResourceChange_Response{}

	proposedStateRaw := req.GetProposedNewState()
	proposedState, err := UnmarshalResource(req.TypeName, proposedStateRaw.GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from proposed plan: %#v", err)
	}
	//	Dlog.Printf("[PlanResourceChange][Request][ProposedNewState]\n%s\n", spew.Sdump(proposedState))

	priorState, err := UnmarshalResource(req.TypeName, req.GetPriorState().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from prior state: %#v", err)
	}
	//	Dlog.Printf("[PlanResourceChange][Request][PriorState]\n%s\n", spew.Sdump(priorState))

	// tfconfig, err := UnmarshalResource(req.TypeName, req.GetConfig().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from configuration: %#v", err)
	}
	//	Dlog.Printf("[PlanResourceChange][Request][Configuration]\n%s\n", spew.Sdump(tfconfig))

	if proposedState.IsNull() {
		// this is a delete
		if !priorState.Type().HasAttribute("object") {
			return resp, fmt.Errorf("cannot find existing object state before delete")
		}
		// dobj := priorState.GetAttr("object")
		//		Dlog.Printf("[PlanResourceChange] Resource to be deleted:\n%s", spew.Sdump(dobj))
		resp.PlannedState = req.ProposedNewState
	} else {
		var cobj *cty.Value
		if priorState.IsNull() {
			// no prior state = new resource
			// Dlog.Println("[PlanResourceChange] Resource to be created.")
			m := proposedState.GetAttr("manifest")
			switch req.TypeName {
			case "kubedynamic_yaml_manifest":
				rawRes, _, err := ResourceFromYAMLManifest([]byte(m.AsString()))
				if err != nil {
					return resp, err
				}
				c, err := UnstructuredToCty(rawRes)
				if err != nil {
					return resp, err
				}
				cobj = &c
				//				Dlog.Printf("[PlanResourceChange][PlanCreate] YAML resource %s to be created:\n%s\n", spew.Sdump(*gvr), spew.Sdump(cobj))
			case "kubedynamic_hcl_manifest":
				cobj = &m
				// Dlog.Printf("[PlanResourceChange][PlanCreate] HCL resource to be created:\n%s\n", spew.Sdump(cobj))
			}
			// Dlog.Printf("[PlanResourceChange][PlanCreate] cyt.Object\n%s\n", spew.Sdump(cobj))
			planned, err := cty.Transform(proposedState, ResourceBulkUpdateObjectAttr(cobj))
			if err != nil {
				return resp, err
			}
			// Dlog.Printf("[PlanResourceChange][PlanCreate] Transformed planned state:\n%s\n", spew.Sdump(planned))
			plannedState, err := MarshalResource(req.TypeName, &planned)
			if err != nil {
				// Dlog.Println("[PlanResourceChange][PlanCreate] Failed to marshall planned state after transform.")
				return resp, err
			}
			resp.PlannedState = &tfplugin5.DynamicValue{
				Msgpack: plannedState,
			}
		} else {
			// resource needs an update
			resp.PlannedState = proposedStateRaw
		}
	}

	Dlog.Printf("[PlanResourceChange][Request][PlannedState]\n%s\n", spew.Sdump(resp.PlannedState.Msgpack))
	return resp, nil
}

// ApplyResourceChange function
func (s *RawProviderServer) ApplyResourceChange(ctx context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	resp := &tfplugin5.ApplyResourceChange_Response{}

	applyPlannedState, err := UnmarshalResource(req.TypeName, (*req.PlannedState).Msgpack)
	if err != nil {
		return resp, err
	}

	applyPriorState, err := UnmarshalResource(req.TypeName, (*req.PriorState).Msgpack)
	if err != nil {
		return resp, err
	}
	Dlog.Printf("[ApplyResourceChange][Request][PlannedState]\n%s\n", spew.Sdump(applyPlannedState))
	//	Dlog.Printf("[ApplyResourceChange][Request][PriorState]\n%s\n", spew.Sdump(applyPriorState))
	//	Dlog.Printf("[ApplyResourceChange][Request][PlannedPrivate]\n%s\n", spew.Sdump(req.PlannedPrivate))

	c, err := GetDynamicClient()
	if err != nil {
		if resp.Diagnostics == nil {
			resp.Diagnostics = make([]*tfplugin5.Diagnostic, 1)
		}
		resp.Diagnostics = append(resp.Diagnostics,
			&tfplugin5.Diagnostic{
				Severity: tfplugin5.Diagnostic_ERROR,
				Summary:  err.Error(),
			})
		return resp, err
	}

	switch {
	case applyPriorState.IsNull():
		{ // Create resource
			o := applyPlannedState.GetAttr("object")
			gvr, err := GVRFromCtyObject(&o)
			if err != nil {
				return resp, err
			}
			var rnamespace string
			om := o.GetAttr("metadata")
			if om.Type().HasAttribute("namespace") {
				rnamespace = om.GetAttr("namespace").AsString()
			}
			var r dynamic.ResourceInterface
			if len(rnamespace) > 0 {
				r = c.Resource(gvr).Namespace(rnamespace)
			} else {
				r = c.Resource(gvr)
			}
			mi, err := CtyObjectToUnstructured(&o)
			if err != nil {
				return resp, err
			}
			uo := unstructured.Unstructured{Object: mi}

			// Call the Kubernetes API to create the resource
			result, err := r.Create(&uo, v1.CreateOptions{})
			if err != nil {
				return resp, err
			}
			// remove status from result, so we don't store it in the state
			delete(result.Object, "status")

			newResObject, err := UnstructuredToCty(result.Object)
			if err != nil {
				return resp, err
			}

			newResState, err := cty.Transform(applyPlannedState,
				ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object"), &newResObject),
			)
			if err != nil {
				return resp, err
			}
			Dlog.Printf("[ApplyResourceChange][Create] Transformed new state:\n%s", spew.Sdump(newResState))

			mp, err := MarshalResource(req.TypeName, &newResState)
			if err != nil {
				return resp, err
			}
			resp.NewState = &tfplugin5.DynamicValue{Msgpack: mp}
		}
	case applyPlannedState.IsNull():
		{ // Delete the resource
			if !applyPriorState.Type().HasAttribute("object") {
				return resp, fmt.Errorf("existing state of resource %s has no 'object' attribute", req.TypeName)
			}
			pco := applyPriorState.GetAttr("object").GetAttr("value")
			pu, err := CtyObjectToUnstructured(&pco)
			cu := unstructured.Unstructured{Object: pu}

			rnamespace := cu.GetNamespace()
			rname := cu.GetName()

			gvr, err := GVRFromCtyUnstructured(&cu)
			if err != nil {
				return resp, err
			}
			r := c.Resource(gvr)
			var derr error
			if len(rnamespace) == 0 {
				derr = r.Delete(rname, &v1.DeleteOptions{})
			} else {
				derr = r.Namespace(rnamespace).Delete(rname, &v1.DeleteOptions{})
			}
			if derr != nil {
				return resp, fmt.Errorf("failed to delete resource %s/%s: %s", rnamespace, rname, err)
			}
			//			Dlog.Printf("[ApplyResourceChange][Delete] successfully deleted %s/%s", rnamespace, rname)
			resp.NewState = req.PlannedState
		}
	}

	// resp.NewState = req.PlannedState
	return resp, nil
}

// ImportResourceState function
func (*RawProviderServer) ImportResourceState(ctx context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	// Terraform only gives us the schema name of the resource and an ID string, as passed by the user on the command line.
	// The ID should be a combination of a Kubernetes GRV and a namespace/name type of resource identifier.
	// Without the user supplying the GRV there is no way to fully indentify the resource when making the Get API call to K8s.
	// Presumably the Kubernetes API machinery already has a standard for expressing such a group. We should look there first.
	return nil, status.Errorf(codes.Unimplemented, "method ImportResourceState not implemented")
}

// ReadDataSource function
func (s *RawProviderServer) ReadDataSource(ctx context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	//	Dlog.Printf("[ReadDataSource][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ReadDataSource not implemented")
}

// Stop function
func (s *RawProviderServer) Stop(ctx context.Context, req *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	//	Dlog.Printf("[Stop][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
