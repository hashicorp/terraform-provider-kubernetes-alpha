package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexsomesan/terraform-provider-raw/tfplugin5"
	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RawProviderServer implements the ProviderServer interface as exported from ProtoBuf.
type RawProviderServer struct{}

// GetSchema function
func (s *RawProviderServer) GetSchema(ctx context.Context, req *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	Dlog.Printf("[GetSchema][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	resp := &tfplugin5.GetProviderSchema_Response{
		Provider:        GetProviderConfigSchema(),
		ResourceSchemas: GetProviderResourceSchema(),
	}
	return resp, nil
}

// PrepareProviderConfig function
func (s *RawProviderServer) PrepareProviderConfig(ctx context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	resp := &tfplugin5.PrepareProviderConfig_Response{}

	config, err := msgpack.Unmarshal(req.Config.Msgpack, GetConfigObjectType())
	Dlog.Printf("[PrepareProviderConfig][Request][Config] >>>>>>\n%s\n<<<<<<", spew.Sdump(config))
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// ValidateResourceTypeConfig function
func (s *RawProviderServer) ValidateResourceTypeConfig(ctx context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	Dlog.Printf("[ValidateResourceTypeConfig][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	config := &tfplugin5.ValidateResourceTypeConfig_Response{}
	return config, nil
}

// ValidateDataSourceConfig function
func (s *RawProviderServer) ValidateDataSourceConfig(ctx context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	Dlog.Printf("[ValidateDataSourceConfig][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ValidateDataSourceConfig not implemented")
}

// UpgradeResourceState function
func (s *RawProviderServer) UpgradeResourceState(ctx context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	Dlog.Printf("[UpgradeResourceState][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method UpgradeResourceState not implemented")
}

// Configure function
func (s *RawProviderServer) Configure(ctx context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	Dlog.Printf("[Configure][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))
	response := &tfplugin5.Configure_Response{}

	providerConfig, err := msgpack.Unmarshal(req.Config.Msgpack, GetConfigObjectType())
	Dlog.Printf("[Configure][Request][Config] >>>>>>\n%s\n<<<<<<", spew.Sdump(providerConfig))
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

	dynClient, errClient := dynamic.NewForConfig(clientConfig)
	if errClient != nil {
		Dlog.Printf("[Configure] Error creating client %v", errClient)
		return response, errClient
	}

	GetProviderState()[DynamicClient] = dynClient
	Dlog.Printf("[Configure] Successfully created dynamic client.\n")

	return response, nil
}

// ReadResource function
func (s *RawProviderServer) ReadResource(ctx context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	Dlog.Printf("[ReadResource][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ReadResource not implemented")
}

// PlanResourceChange function
func (s *RawProviderServer) PlanResourceChange(ctx context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	resp := &tfplugin5.PlanResourceChange_Response{
		PlannedState: &tfplugin5.DynamicValue{},
	}

	Dlog.Printf("[PlanResourceChange][Request][ProposedNewState] >>>>>>\n%s\n<<<<<<", spew.Sdump(msgpack.Unmarshal(req.ProposedNewState.Msgpack, GetPlanObjectType())))

	proposedPacked := req.GetProposedNewState().GetMsgpack()
	priorPacked := req.GetPriorState().GetMsgpack()
	tfconfigPacked := req.GetConfig().GetMsgpack()

	proposedManifest, err := ExtractPackedManifest(proposedPacked)
	if err != nil {
		return resp, fmt.Errorf("Failed to extract manifest from proposed plan: %#v", err)
	}
	Dlog.Printf("[PlanResourceChange][Request][ProposedNewState][Manifest] %s", proposedManifest)

	priorManifest, err := ExtractPackedManifest(priorPacked)
	if err != nil {
		return resp, fmt.Errorf("Failed to extract manifest from prior state: %#v", err)
	}
	Dlog.Printf("[PlanResourceChange][Request][PriorState][Manifest] %s", priorManifest)

	tfconfig, err := ExtractPackedManifest(tfconfigPacked)
	if err != nil {
		return resp, fmt.Errorf("Failed to extract manifest from configuration: %#v", err)
	}
	Dlog.Printf("[PlanResourceChange][Request][Configuration][Manifest] %s", tfconfig)

	var plannedManifest string
	if len(priorManifest) == 0 {
		plannedManifest = proposedManifest
	}

	m := cty.ObjectVal(map[string]cty.Value{
		"manifest": cty.StringVal(plannedManifest),
	})

	planmsgp, err := msgpack.Marshal(m, GetPlanObjectType())
	if err != nil {
		return resp, err
	}
	resp.PlannedState.Msgpack = planmsgp

	Dlog.Printf("[PlanResourceChange][Request][PlannedState] >>>>>>\n%s\n<<<<<<", spew.Sdump(m))

	return resp, nil
}

// ApplyResourceChange function
func (s *RawProviderServer) ApplyResourceChange(ctx context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	applyConfig, err := msgpack.Unmarshal((*req.Config).Msgpack, GetPlanObjectType())
	if err != nil {
		return nil, err
	}
	configManifest := applyConfig.GetAttr("manifest")
	Dlog.Printf("[ApplyResourceChange][Request][Config] >>>>>>\n%s\n<<<<<<", spew.Sdump(configManifest))

	applyPlannedState, err := msgpack.Unmarshal((*req.PlannedState).Msgpack, GetPlanObjectType())
	if err != nil {
		return nil, err
	}
	plannedManifest := applyPlannedState.GetAttr("manifest")
	Dlog.Printf("[ApplyResourceChange][Request][PlannedState] >>>>>>\n%s\n<<<<<<", spew.Sdump(plannedManifest))

	applyPriorState, err := msgpack.Unmarshal((*req.PriorState).Msgpack, GetPlanObjectType())
	if err != nil {
		return nil, err
	}
	applyPriorState.AsValueMap()
	priorManifest := applyPriorState.GetAttr("manifest")
	Dlog.Printf("[ApplyResourceChange][Request][PriorState] >>>>>>\n%s\n<<<<<<", spew.Sdump(priorManifest))

	return nil, status.Errorf(codes.Unimplemented, "method ApplyResourceChange not implemented")
}

// ImportResourceState function
func (*RawProviderServer) ImportResourceState(ctx context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	Dlog.Printf("[ImportResourceState][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ImportResourceState not implemented")
}

// ReadDataSource function
func (s *RawProviderServer) ReadDataSource(ctx context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	Dlog.Printf("[ReadDataSource][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method ReadDataSource not implemented")
}

// Stop function
func (s *RawProviderServer) Stop(ctx context.Context, req *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	Dlog.Printf("[Stop][Request] >>>>>>\n%s\n<<<<<<", spew.Sdump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
