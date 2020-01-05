package provider

import (
	"context"
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
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	Dlog.Printf("[GetSchema][Request]\n%s\n", spew.Sdump(*req))

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
	// Dlog.Printf("[PrepareProviderConfig][Request][Config]\n%s\n", spew.Sdump(config))
	// if err != nil {
	// 	return resp, err
	// }

	return resp, nil
}

// ValidateResourceTypeConfig function
func (s *RawProviderServer) ValidateResourceTypeConfig(ctx context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	Dlog.Printf("[ValidateResourceTypeConfig][Request]\n%s\n", spew.Sdump(*req))

	config := &tfplugin5.ValidateResourceTypeConfig_Response{}
	return config, nil
}

// ValidateDataSourceConfig function
func (s *RawProviderServer) ValidateDataSourceConfig(ctx context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	Dlog.Printf("[ValidateDataSourceConfig][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ValidateDataSourceConfig not implemented")
}

// UpgradeResourceState function
func (s *RawProviderServer) UpgradeResourceState(ctx context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	Dlog.Printf("[UpgradeResourceState][Request]\n%s\n", spew.Sdump(*req))
	resp := &tfplugin5.UpgradeResourceState_Response{}
	return resp, nil
}

// Configure function
func (s *RawProviderServer) Configure(ctx context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	Dlog.Printf("[Configure][Request]\n%s\n", spew.Sdump(*req))
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
			Dlog.Printf("[Configure][Kubeconfig] %v.\n", err)
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
		Dlog.Printf("[Configure][Kubeconfig] %s.\n", err.Error())
		return response, err
	}
	if logging.IsDebugOrHigher() {
		clientConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return logging.NewTransport("Kubernetes", rt)
		}
	}

	dynClient, errClient := dynamic.NewForConfig(clientConfig)
	if errClient != nil {
		Dlog.Printf("[Configure] Error creating dynamic client %v", errClient)
		return response, errClient
	}

	discoClient, errClient := discovery.NewDiscoveryClientForConfig(clientConfig)
	if errClient != nil {
		Dlog.Printf("[Configure] Error creating discovery client %v", errClient)
		return response, errClient
	}

	cacher := memory.NewMemCacheClient(discoClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cacher)

	ps := GetProviderState()
	ps[DynamicClient] = dynClient
	ps[DiscoveryClient] = discoClient
	ps[RestMapper] = mapper

	Dlog.Printf("[Configure] Successfully created dicovery client.\n")

	return response, nil
}

// ReadResource function
func (s *RawProviderServer) ReadResource(ctx context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	Dlog.Printf("[ReadResource][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ReadResource not implemented")
}

// PlanResourceChange function
func (s *RawProviderServer) PlanResourceChange(ctx context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	resp := &tfplugin5.PlanResourceChange_Response{}

	proposedState, err := UnmarshalResource(req.TypeName, req.GetProposedNewState().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from proposed plan: %#v", err)
	}
	Dlog.Printf("[PlanResourceChange][Request][ProposedNewState]\n%s\n", spew.Sdump(proposedState))

	priorState, err := UnmarshalResource(req.TypeName, req.GetPriorState().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from prior state: %#v", err)
	}
	Dlog.Printf("[PlanResourceChange][Request][PriorState]\n%s\n", spew.Sdump(priorState))

	tfconfig, err := UnmarshalResource(req.TypeName, req.GetConfig().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from configuration: %#v", err)
	}
	Dlog.Printf("[PlanResourceChange][Request][Configuration]\n%s\n", spew.Sdump(tfconfig))

	if proposedState.IsNull() {
		// this is a delete
		if !priorState.Type().HasAttribute("object") {
			return resp, fmt.Errorf("cannot find existing object state before delete")
		}
		dobj := priorState.GetAttr("object")
		Dlog.Printf("[PlanResourceChange] Resource to be deleted:\n%s", spew.Sdump(dobj))
		resp.PlannedState = req.ProposedNewState
	} else {
		var cobj *cty.Value
		if priorState.IsNull() {
			// no prior state = new resource
			Dlog.Println("[PlanResourceChange] Resource to be created.")
			m := proposedState.GetAttr("manifest")
			switch req.TypeName {
			case "kubedynamic_yaml_manifest":
				rawRes, gvk, err := ResourceFromYAMLManifest([]byte(m.AsString()))
				if err != nil {
					return resp, err
				}
				cobj, err = UnstructuredToCty(rawRes)
				if err != nil {
					return resp, err
				}
				Dlog.Printf("[PlanResourceChange][PlanCreate] YAML resource %s to be created:\n%s\n", spew.Sdump(*gvk), spew.Sdump(cobj))
			case "kubedynamic_hcl_manifest":
				cobj = &m
				gvk := schema.FromAPIVersionAndKind(m.GetAttr("apiVersion").AsString(), m.GetAttr("kind").AsString())
				Dlog.Printf("[PlanResourceChange][PlanCreate] HCL resource %s to be created:\n%s\n", spew.Sdump(gvk), spew.Sdump(cobj))
			}
			Dlog.Printf("[PlanResourceChange][PlanCreate] cyt.Object\n%s\n", spew.Sdump(cobj))
			planned, err := cty.Transform(proposedState, func(path cty.Path, v cty.Value) (cty.Value, error) {
				if path.Equals(cty.GetAttrPath("object")) {
					return *cobj, nil
				}
				return v, nil
			})
			if err != nil {
				return resp, err
			}
			Dlog.Printf("[PlanResourceChange][PlanCreate] Transformed planned state:\n%s\n", spew.Sdump(planned))
			plannedState, err := MarshalResource(req.TypeName, planned)
			if err != nil {
				Dlog.Println("[PlanResourceChange][PlanCreate] Failed to marshall planned state after transform.")
				return resp, err
			}
			resp.PlannedState = &tfplugin5.DynamicValue{
				Msgpack: plannedState,
			}
		} else {
			// resource needs an update
			m := cty.ObjectVal(map[string]cty.Value{
				"manifest": proposedState.GetAttr("manifest"),
				// TODO: replace with actual update logic
				"object": cty.ObjectVal(map[string]cty.Value{}),
			})
			planmsgp, err := msgpack.Marshal(m, m.Type())
			if err != nil {
				return resp, err
			}
			resp.PlannedState.Msgpack = planmsgp
		}
	}

	Dlog.Printf("[PlanResourceChange][Request][PlannedState]\n%s\n", spew.Sdump(resp.PlannedState.Msgpack))
	return resp, nil
}

// ApplyResourceChange function
func (s *RawProviderServer) ApplyResourceChange(ctx context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	resp := &tfplugin5.ApplyResourceChange_Response{}

	applyConfig, err := UnmarshalResource(req.TypeName, (*req.Config).Msgpack)
	if err != nil {
		return resp, err
	}
	Dlog.Printf("[ApplyResourceChange][Request][Config]\n%s\n", spew.Sdump(applyConfig))

	applyPlannedState, err := UnmarshalResource(req.TypeName, (*req.PlannedState).Msgpack)
	if err != nil {
		return resp, err
	}
	Dlog.Printf("[ApplyResourceChange][Request][PlannedState]\n%s\n", spew.Sdump(applyPlannedState))

	applyPriorState, err := UnmarshalResource(req.TypeName, (*req.PriorState).Msgpack)
	if err != nil {
		return resp, err
	}
	Dlog.Printf("[ApplyResourceChange][Request][PriorState]\n%s\n", spew.Sdump(applyPriorState))
	Dlog.Printf("[ApplyResourceChange][Request][PlannedPrivate]\n%s\n", spew.Sdump(req.PlannedPrivate))

	c, err := GetDynamicClient()
	if err != nil {
		d := resp.Diagnostics
		if d == nil {
			d = make([]*tfplugin5.Diagnostic, 1)
		}
		d = append(d, &tfplugin5.Diagnostic{Severity: tfplugin5.Diagnostic_ERROR, Summary: err.Error()})
		resp.Diagnostics = d
		return resp, err
	}

	switch {
	case applyPriorState.IsNull():
		{ // Create resource
			o := applyPlannedState.GetAttr("object")
			gvr, err := GVRFromCtyObject(&o)
			if err != nil {
				Dlog.Printf("[ApplyResourceChange][Create] Failed to discover GVR: %s\n%s", err, spew.Sdump(o))
			}
			r := c.Resource(*gvr)
			mi, err := CtyToUnstructured(&o)
			if err != nil {
				Dlog.Printf("[ApplyResourceChange][Create] failed to convert proposed state (%s) :\n%s",
					err.Error(), spew.Sdump(mi))
			}
			uo := unstructured.Unstructured{Object: mi}
			Dlog.Printf("[ApplyResourceChange][Create] Creating object:\n%s", spew.Sdump(uo))
			ro, err := r.Create(&uo, v1.CreateOptions{})
			if err != nil {
				Dlog.Printf("[ApplyResourceChange][Create] failed to create object: %s\n%s", err, spew.Sdump(ro))
			}
		}
	case applyPlannedState.IsNull():
		{ // Delete
		}
	}

	resp.NewState = req.PlannedState
	return resp, nil
}

// ImportResourceState function
func (*RawProviderServer) ImportResourceState(ctx context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	Dlog.Printf("[ImportResourceState][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ImportResourceState not implemented")
}

// ReadDataSource function
func (s *RawProviderServer) ReadDataSource(ctx context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	Dlog.Printf("[ReadDataSource][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ReadDataSource not implemented")
}

// Stop function
func (s *RawProviderServer) Stop(ctx context.Context, req *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	Dlog.Printf("[Stop][Request]\n%s\n", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
