package provider

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
	"github.com/mitchellh/go-homedir"

	"github.com/hashicorp/go-cty/cty/msgpack"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func init() {
	install.Install(scheme.Scheme)
}

// RawProviderServer implements the ProviderServer interface as exported from ProtoBuf.
type RawProviderServer struct{}

// GetSchema function
func (s *RawProviderServer) GetSchema(ctx context.Context, req *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {

	cfgSchema, err := GetProviderConfigSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to construct provider schema: %s", err)
	}
	resSchema, err := GetProviderResourceSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to construct resource schema: %s", err)
	}
	resp := &tfplugin5.GetProviderSchema_Response{
		Provider:        cfgSchema,
		ResourceSchemas: resSchema,
	}
	return resp, nil
}

// PrepareProviderConfig function
func (s *RawProviderServer) PrepareProviderConfig(ctx context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	resp := &tfplugin5.PrepareProviderConfig_Response{}

	diags := []*tfplugin5.Diagnostic{}
	var err error

	providerConfig, err := msgpack.Unmarshal(req.Config.Msgpack, getConfigObjectType())
	if err != nil {
		return resp, err
	}

	configPath := providerConfig.GetAttr("config_path")
	if !configPath.IsNull() {
		configPathAbs, err := homedir.Expand(configPath.AsString())
		if err == nil {
			_, err = os.Stat(configPathAbs)
		}
		if err != nil {
			diags = append(diags, &tfplugin5.Diagnostic{
				Severity: tfplugin5.Diagnostic_INVALID,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'config_path' refers to an invalid file path",
				Attribute: &tfplugin5.AttributePath{
					Steps: []*tfplugin5.AttributePath_Step{
						{
							Selector: &tfplugin5.AttributePath_Step_AttributeName{
								AttributeName: "config_path",
							},
						},
					},
				},
			})
		}
	}

	host := providerConfig.GetAttr("host")
	if !host.IsNull() {
		_, err = url.ParseRequestURI(host.AsString())
		if err != nil {
			diags = append(diags, &tfplugin5.Diagnostic{
				Severity: tfplugin5.Diagnostic_INVALID,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'host' is not a valid URL",
				Attribute: &tfplugin5.AttributePath{
					Steps: []*tfplugin5.AttributePath_Step{
						{
							Selector: &tfplugin5.AttributePath_Step_AttributeName{
								AttributeName: "host",
							},
						},
					},
				},
			})
		}
	}

	pemCA := providerConfig.GetAttr("cluster_ca_certificate")
	if !pemCA.IsNull() {
		pem, _ := pem.Decode([]byte(pemCA.AsString()))
		if pem == nil || pem.Type != "CERTIFICATE" {
			diags = append(diags, &tfplugin5.Diagnostic{
				Severity: tfplugin5.Diagnostic_INVALID,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'cluster_ca_certificate' is not a valid PEM encoded certificate",
				Attribute: &tfplugin5.AttributePath{
					Steps: []*tfplugin5.AttributePath_Step{
						{
							Selector: &tfplugin5.AttributePath_Step_AttributeName{
								AttributeName: "cluster_ca_certificate",
							},
						},
					},
				},
			})
		}
	}

	pemCC := providerConfig.GetAttr("client_certificate")
	if !pemCC.IsNull() {
		pem, _ := pem.Decode([]byte(pemCC.AsString()))
		if pem == nil || pem.Type != "CERTIFICATE" {
			diags = append(diags, &tfplugin5.Diagnostic{
				Severity: tfplugin5.Diagnostic_INVALID,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_certificate' is not a valid PEM encoded certificate",
				Attribute: &tfplugin5.AttributePath{
					Steps: []*tfplugin5.AttributePath_Step{
						{
							Selector: &tfplugin5.AttributePath_Step_AttributeName{
								AttributeName: "client_certificate",
							},
						},
					},
				},
			})
		}
	}

	pemCK := providerConfig.GetAttr("client_key")
	if !pemCK.IsNull() {
		pem, _ := pem.Decode([]byte(pemCK.AsString()))
		if pem == nil || !strings.Contains(pem.Type, "PRIVATE KEY") {
			diags = append(diags, &tfplugin5.Diagnostic{
				Severity: tfplugin5.Diagnostic_INVALID,
				Summary:  "Invalid attribute in provider configuration",
				Detail:   "'client_key' is not a valid PEM encoded private key",
				Attribute: &tfplugin5.AttributePath{
					Steps: []*tfplugin5.AttributePath_Step{
						{
							Selector: &tfplugin5.AttributePath_Step_AttributeName{
								AttributeName: "client_key",
							},
						},
					},
				},
			})
		}
	}

	resp.Diagnostics = diags

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

// UpgradeResourceState isn't really useful in this provider, but we have to loop the state back through to keep Terraform happy.
func (s *RawProviderServer) UpgradeResourceState(ctx context.Context, req *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	resp := &tfplugin5.UpgradeResourceState_Response{}
	resp.Diagnostics = []*tfplugin5.Diagnostic{}

	sch, err := GetProviderResourceSchema()
	if err != nil {
		return resp, err
	}
	rt, err := GetObjectTypeFromSchema(sch[req.TypeName])
	if err != nil {
		return resp, err
	}
	rv, err := json.Unmarshal(req.RawState.Json, rt)
	if err != nil {
		resp.Diagnostics = AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	newStateMP, err := msgpack.Marshal(rv, rt)
	if err != nil {
		resp.Diagnostics = AppendProtoDiag(resp.Diagnostics, err)
		return resp, nil
	}
	resp.UpgradedState = &tfplugin5.DynamicValue{Msgpack: newStateMP}
	return resp, nil
}

// Configure function
func (s *RawProviderServer) Configure(ctx context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	response := &tfplugin5.Configure_Response{}

	providerConfig, err := msgpack.Unmarshal(req.Config.Msgpack, getConfigObjectType())
	if err != nil {
		return response, err
	}

	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	var configPath cty.Value
	if configPathEnv, ok := os.LookupEnv("KUBE_CONFIG_PATH"); ok && configPathEnv != "" {
		configPath = cty.StringVal(configPathEnv)
	} else {
		configPath = providerConfig.GetAttr("config_path")
	}
	if !configPath.IsNull() {
		configPathAbs, err := homedir.Expand(configPath.AsString())
		if err != nil {
			return response, fmt.Errorf("cannot load specified config file: %s", err)
		}
		loader.ExplicitPath = configPathAbs
	}

	var cfgContext cty.Value
	if cfgContextEnv, ok := os.LookupEnv("KUBE_CTX"); ok && cfgContextEnv != "" {
		cfgContext = cty.StringVal(cfgContextEnv)
	} else {
		cfgContext = providerConfig.GetAttr("config_context")
	}
	if !cfgContext.IsNull() {
		overrides.CurrentContext = cfgContext.AsString()
	}

	overrides.Context = clientcmdapi.Context{}

	var cfgCtxCluster cty.Value
	if cfgCtxClusterEnv, ok := os.LookupEnv("KUBE_CTX_CLUSTER"); ok && cfgCtxClusterEnv != "" {
		cfgCtxCluster = cty.StringVal(cfgCtxClusterEnv)
	} else {
		cfgCtxCluster = providerConfig.GetAttr("config_context_cluster")
	}
	if !cfgCtxCluster.IsNull() {
		overrides.Context.Cluster = cfgCtxCluster.AsString()
	}

	var cfgContextAuthInfo cty.Value
	if cfgContextAuthInfoEnv, ok := os.LookupEnv("KUBE_CTX_USER"); ok && cfgContextAuthInfoEnv != "" {
		cfgContextAuthInfo = cty.StringVal(cfgContextAuthInfoEnv)
	} else {
		cfgContextAuthInfo = providerConfig.GetAttr("config_context_user")
	}
	if !cfgContextAuthInfo.IsNull() {
		overrides.Context.AuthInfo = cfgContextAuthInfo.AsString()
	}

	var insecure cty.Value
	if insecureEnv, ok := os.LookupEnv("KUBE_INSECURE"); ok && insecureEnv != "" {
		iv, err := strconv.ParseBool(insecureEnv)
		if err != nil {
			return response, fmt.Errorf("failed to parse config value of 'insecure': %s", err)
		}
		insecure = cty.BoolVal(iv)
	} else {
		insecure = providerConfig.GetAttr("insecure")
	}
	if !insecure.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = insecure.True()
	}

	var clusterCA cty.Value
	if clusterCAEnv, ok := os.LookupEnv("KUBE_CLUSTER_CA_CERT_DATA"); ok && clusterCAEnv != "" {
		clusterCA = cty.StringVal(clusterCAEnv)
	} else {
		clusterCA = providerConfig.GetAttr("cluster_ca_certificate")
	}
	if !clusterCA.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(clusterCA.AsString()).Bytes()
	}

	var clientCrt cty.Value
	if clientCrtEnv, ok := os.LookupEnv("KUBE_CLIENT_CERT_DATA"); ok && clientCrtEnv != "" {
		clientCrt = cty.StringVal(clientCrtEnv)
	} else {
		clientCrt = providerConfig.GetAttr("client_certificate")
	}
	if !clientCrt.IsNull() {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(clientCrt.AsString()).Bytes()
	}

	var clientCrtKey cty.Value
	if clientCrtKeyEnv, ok := os.LookupEnv("KUBE_CLIENT_KEY_DATA"); ok && clientCrtKeyEnv != "" {
		clientCrtKey = cty.StringVal(clientCrtKeyEnv)
	} else {
		clientCrtKey = providerConfig.GetAttr("client_key")
	}
	if !clientCrtKey.IsNull() {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(clientCrtKey.AsString()).Bytes()
	}

	var host cty.Value
	if hostEnv, ok := os.LookupEnv("KUBE_HOST"); ok && hostEnv != "" {
		host = cty.StringVal(hostEnv)
	} else {
		host = providerConfig.GetAttr("host")
	}
	if !host.IsNull() {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		hostURL, _, err := rest.DefaultServerURL(host.AsString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}
		overrides.ClusterInfo.Server = hostURL.String()
	}

	var username cty.Value
	if usernameEnv, ok := os.LookupEnv("KUBE_USERNAME"); ok && usernameEnv != "" {
		username = cty.StringVal(usernameEnv)
	} else {
		username = providerConfig.GetAttr("username")
	}
	if !username.IsNull() {
		overrides.AuthInfo.Username = username.AsString()
	}

	var password cty.Value
	if passwordEnv, ok := os.LookupEnv("KUBE_PASSWORD"); ok && passwordEnv != "" {
		password = cty.StringVal(passwordEnv)
	} else {
		password = providerConfig.GetAttr("password")
	}
	if !password.IsNull() {
		overrides.AuthInfo.Username = password.AsString()
	}

	var token cty.Value
	if tokenEnv, ok := os.LookupEnv("KUBE_TOKEN"); ok && tokenEnv != "" {
		token = cty.StringVal(tokenEnv)
	} else {
		token = providerConfig.GetAttr("token")
	}
	if !token.IsNull() {
		overrides.AuthInfo.Token = token.AsString()
	}

	execObj := providerConfig.GetAttr("exec")
	if !execObj.IsNull() {
		execCfg := clientcmdapi.ExecConfig{}
		apiv := execObj.GetAttr("api_version")
		if !apiv.IsNull() {
			execCfg.APIVersion = apiv.AsString()
		}
		cmd := execObj.GetAttr("command")
		if !cmd.IsNull() {
			execCfg.Command = cmd.AsString()
		}
		xcmdArgs := execObj.GetAttr("args")
		if !xcmdArgs.IsNull() && xcmdArgs.LengthInt() > 0 {
			execCfg.Args = make([]string, 0, xcmdArgs.LengthInt())
			for ait := xcmdArgs.ElementIterator(); ait.Next(); {
				_, v := ait.Element()
				execCfg.Args = append(execCfg.Args, v.AsString())
			}
		}
		xcmdEnvs := execObj.GetAttr("env")
		if !xcmdEnvs.IsNull() && xcmdEnvs.LengthInt() > 0 {
			execCfg.Env = make([]clientcmdapi.ExecEnvVar, 0, xcmdEnvs.LengthInt())
			for eit := xcmdEnvs.ElementIterator(); eit.Next(); {
				k, v := eit.Element()
				execCfg.Env = append(execCfg.Env, clientcmdapi.ExecEnvVar{
					Name:  k.AsString(),
					Value: v.AsString(),
				})
			}
		}
		overrides.AuthInfo.Exec = &execCfg
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	clientConfig, err := cc.ClientConfig()
	if err != nil {
		Dlog.Printf("[Configure] Failed to load config:\n%s\n", spew.Sdump(cc))
		return response, fmt.Errorf("cannot load Kubernetes client config: %s", err)
	}

	Dlog.Printf("[Configure][ClientConfig] %s\n", spew.Sdump(*clientConfig))

	if logging.IsDebugOrHigher() {
		clientConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return logging.NewTransport("Kubernetes", rt)
		}
	}

	dynClient, errClient := dynamic.NewForConfig(clientConfig)
	if errClient != nil {
		return response, errClient
	}

	discoClient, errClient := discovery.NewDiscoveryClientForConfig(clientConfig)
	if errClient != nil {
		return response, errClient
	}

	cacher := memory.NewMemCacheClient(discoClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cacher)

	codec := runtime.NoopEncoder{Decoder: scheme.Codecs.UniversalDecoder()}
	clientConfig.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})
	restClient, errClient := rest.UnversionedRESTClientFor(clientConfig)
	if errClient != nil {
		Dlog.Printf("[Configure] Error creating REST client %v", errClient)
		return response, errClient
	}

	ps := GetProviderState()
	ps[DynamicClient] = dynClient
	ps[DiscoveryClient] = discoClient
	ps[RestMapper] = mapper
	ps[RestClient] = restClient

	ssp := providerConfig.GetAttr("server_side_planning")
	if !ssp.IsKnown() || ssp.IsNull() {
		ssp = cty.True // default to true
	}
	ps[SSPlanning] = ssp.True()
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

	co := currentState.GetAttr("object")
	mo, err := CtyObjectToUnstructured(&co)
	if err != nil {
		return resp, fmt.Errorf("failed to convert current state to unstructured: %s", err)
	}

	uo := unstructured.Unstructured{Object: mo}
	client, err := GetDynamicClient()
	if err != nil {
		return resp, err
	}
	cGVR, err := GVRFromCtyUnstructured(&uo)
	if err != nil {
		return resp, err
	}
	ns, err := IsResourceNamespaced(cGVR)
	if err != nil {
		return resp, err
	}
	rcl := client.Resource(cGVR)

	rnamespace := uo.GetNamespace()
	rname := uo.GetName()

	var fo *unstructured.Unstructured
	if ns {
		fo, err = rcl.Namespace(rnamespace).Get(ctx, rname, v1.GetOptions{})
	} else {
		fo, err = rcl.Get(ctx, rname, v1.GetOptions{})
	}
	if err != nil {
		if errors.IsNotFound(err) {
			return resp, nil
		}
		d := tfplugin5.Diagnostic{
			Severity: tfplugin5.Diagnostic_ERROR,
			Summary:  fmt.Sprintf("Cannot GET resource %s", spew.Sdump(co)),
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, &d)
		return resp, err
	}

	nobj, err := UnstructuredToCty(FilterEphemeralFields(fo.Object))
	if err != nil {
		return resp, err
	}
	newstate, err := cty.Transform(currentState, ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object"), &nobj))
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

	proposedState, err := UnmarshalResource(req.TypeName, req.GetProposedNewState().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from proposed plan: %#v", err)
	}
	priorState, err := UnmarshalResource(req.TypeName, req.GetPriorState().GetMsgpack())
	if err != nil {
		return resp, fmt.Errorf("Failed to extract resource from prior state: %#v", err)
	}

	if proposedState.IsNull() {
		// we plan to delete the resource
		if !priorState.Type().HasAttribute("object") {
			return resp, fmt.Errorf("cannot find existing object state before delete")
		}
		resp.PlannedState = req.ProposedNewState
		return resp, nil
	}

	var planned cty.Value

	switch req.TypeName {
	case "kubernetes_manifest":
		planned, err = PlanUpdateResourceHCL(ctx, &proposedState)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics,
				&tfplugin5.Diagnostic{
					Severity: tfplugin5.Diagnostic_ERROR,
					Summary:  err.Error(),
				})
			return resp, err
		}
	}

	plannedState, err := MarshalResource(req.TypeName, &planned)
	if err != nil {
		return resp, err
	}
	resp.PlannedState = &tfplugin5.DynamicValue{
		Msgpack: plannedState,
	}
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
	Dlog.Printf("[ApplyResourceChange][Request][PriorState]\n%s\n", spew.Sdump(applyPriorState))
	Dlog.Printf("[ApplyResourceChange][Request][PlannedPrivate]\n%s\n", spew.Sdump(req.PlannedPrivate))

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
	var rs dynamic.ResourceInterface

	switch {
	case applyPriorState.IsNull():
		{ // Create resource
			o := applyPlannedState.GetAttr("object")
			gvr, err := GVRFromCtyObject(&o)
			if err != nil {
				return resp, err
			}
			mi, err := CtyObjectToUnstructured(&o)
			if err != nil {
				return resp, err
			}
			uo := unstructured.Unstructured{Object: mi}
			ns, err := IsResourceNamespaced(gvr)
			if err != nil {
				return resp, err
			}
			rnamespace := uo.GetNamespace()
			rname := uo.GetName()
			if ns {
				rs = c.Resource(gvr).Namespace(rnamespace)
			} else {
				rs = c.Resource(gvr)
			}
			jd, err := uo.MarshalJSON()
			if err != nil {
				return resp, err
			}
			// Call the Kubernetes API to create the new resource
			result, err := rs.Patch(ctx, rname, types.ApplyPatchType, jd, v1.PatchOptions{FieldManager: "Terraform"})
			if err != nil {
				Dlog.Printf("[ApplyResourceChange][Create] Error: %s\n%s\n", spew.Sdump(err), spew.Sdump(result))
				n := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
				return resp, fmt.Errorf("CREATE resource %s failed: %s", n, err)
			}
			Dlog.Printf("[ApplyResourceChange][Create] API response:\n%s\n", spew.Sdump(result))

			newResObject, err := UnstructuredToCty(FilterEphemeralFields(result.Object))
			if err != nil {
				return resp, err
			}
			Dlog.Printf("[ApplyResourceChange][Create] transformed response:\n%s\n", spew.Sdump(newResObject))

			newResState, err := cty.Transform(applyPlannedState,
				ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object"), &newResObject),
			)
			if err != nil {
				return resp, err
			}
			Dlog.Printf("[ApplyResourceChange][Create] transformed new state:\n%s", spew.Sdump(newResState))
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
			pco := applyPriorState.GetAttr("object")
			pu, err := CtyObjectToUnstructured(&pco)
			if err != nil {
				return resp, err
			}
			uo := unstructured.Unstructured{Object: pu}
			gvr, err := GVRFromCtyUnstructured(&uo)
			if err != nil {
				return resp, err
			}
			ns, err := IsResourceNamespaced(gvr)
			if err != nil {
				return resp, err
			}

			rnamespace := uo.GetNamespace()
			rname := uo.GetName()

			if ns {
				rs = c.Resource(gvr).Namespace(rnamespace)
			} else {
				rs = c.Resource(gvr)
			}
			err = rs.Delete(ctx, rname, v1.DeleteOptions{})
			if err != nil {
				rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
				return resp, fmt.Errorf("DELETE resource %s failed: %s", rn, err)
			}
			resp.NewState = req.PlannedState
		}
	case !applyPlannedState.IsNull() && !applyPriorState.IsNull():
		{ // Update existing resource
			o := applyPlannedState.GetAttr("object")
			gvr, err := GVRFromCtyObject(&o)
			if err != nil {
				return resp, err
			}
			pu, err := CtyObjectToUnstructured(&o)
			if err != nil {
				return resp, err
			}
			ns, err := IsResourceNamespaced(gvr)
			if err != nil {
				return resp, err
			}
			uo := unstructured.Unstructured{Object: pu}
			rnamespace := uo.GetNamespace()
			rname := uo.GetName()

			c, err := GetDynamicClient()
			if err != nil {
				return resp, err
			}
			if ns {
				rs = c.Resource(gvr).Namespace(rnamespace)
			} else {
				rs = c.Resource(gvr)
			}
			jd, err := uo.MarshalJSON()
			if err != nil {
				return resp, err
			}
			// Call the Kubernetes API to apply the new resource state
			result, err := rs.Patch(ctx, rname, types.ApplyPatchType, jd, v1.PatchOptions{FieldManager: "Terraform"})
			if err != nil {
				rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
				resp.Diagnostics = append(resp.Diagnostics,
					&tfplugin5.Diagnostic{
						Severity: tfplugin5.Diagnostic_ERROR,
						Detail:   err.Error(),
						Summary:  fmt.Sprintf("PATCH resource %s failed: %s", rn, err),
					})
				return resp, fmt.Errorf("PATCH resource %s failed: %s", rn, err)
			}
			Dlog.Printf("[ApplyResourceChange][Update] API response:\n%s\n", spew.Sdump(result))
			newResObject, err := UnstructuredToCty(FilterEphemeralFields(result.Object))
			if err != nil {
				return resp, err
			}
			Dlog.Printf("[ApplyResourceChange][Update] transformed response:\n%s", spew.Sdump(newResObject))
			newResState, err := cty.Transform(applyPlannedState,
				ResourceDeepUpdateObjectAttr(cty.GetAttrPath("object"), &newResObject),
			)
			if err != nil {
				return resp, err
			}
			Dlog.Printf("[ApplyResourceChange][Update] transformed new state:\n%s", spew.Sdump(newResState))
			mp, err := MarshalResource(req.TypeName, &newResState)
			if err != nil {
				return resp, err
			}
			resp.NewState = &tfplugin5.DynamicValue{Msgpack: mp}
		}
	}

	return resp, nil
}

// ImportResourceState function
func (*RawProviderServer) ImportResourceState(ctx context.Context, req *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	// Terraform only gives us the schema name of the resource and an ID string, as passed by the user on the command line.
	// The ID should be a combination of a Kubernetes GRV and a namespace/name type of resource identifier.
	// Without the user supplying the GRV there is no way to fully identify the resource when making the Get API call to K8s.
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
