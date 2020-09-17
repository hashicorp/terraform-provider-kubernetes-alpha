package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/alexsomesan/openapi-cty/foundry"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	// this is how client-go expects auth plugins to be loaded
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// providerState is a very simplistic global state storage.
//
// Since the provider is essentially a gRPC server, the execution flow is dictated by the order of the client (Terraform) request calls.
// Thus it needs a way to persist state between the gRPC calls. This structure stores values that need to be persisted between gRPC calls,
// such as instances of the Kubernetes clients, configuration options needed at runtime.
var providerState map[string]interface{}

// keys into the global state storage
const (
	ClientConfig    string = "CLIENTCONFIG"
	DynamicClient   string = "DYNAMICCLIENT"
	DiscoveryClient string = "DISCOVERYCLIENT"
	RestClient      string = "RESTCLIENT"
	RestMapper      string = "RESTMAPPER"
	SSPlanning      string = "SERVERSIDEPLANNING"
	OAPIFoundry     string = "OPENAPIFOUNDRY"
)

// GetGlobalState returns the global state storage structure.
func GetGlobalState() map[string]interface{} {
	if providerState == nil {
		providerState = make(map[string]interface{})
	}
	return providerState
}

// GetClientConfig returns the client-go rest.Config produced from the provider block attributes.
func GetClientConfig() (*rest.Config, error) {
	s := GetGlobalState()
	c, ok := s[ClientConfig]
	if !ok {
		return nil, fmt.Errorf("no client configuration")
	}
	return c.(*rest.Config), nil
}

// GetDynamicClient returns a configured unstructured (dynamic) client instance
func GetDynamicClient() (dynamic.Interface, error) {
	s := GetGlobalState()
	c, ok := s[DynamicClient]
	if ok {
		return c.(dynamic.Interface), nil
	}
	clientConfig, err := GetClientConfig()
	if err != nil {
		return nil, err
	}
	dynClient, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	s[DynamicClient] = dynClient
	return dynClient, nil
}

// GetDiscoveryClient returns a configured discovery client instance.
func GetDiscoveryClient() (discovery.DiscoveryInterface, error) {
	s := GetGlobalState()
	c, ok := s[DiscoveryClient]
	if ok {
		return c.(*discovery.DiscoveryClient), nil
	}
	clientConfig, err := GetClientConfig()
	if err != nil {
		return nil, err
	}
	discoClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	s[DiscoveryClient] = discoClient
	return discoClient, nil
}

// GetRestMapper returns a RESTMapper client instance
func GetRestMapper() (meta.RESTMapper, error) {
	s := GetGlobalState()
	c, ok := s[RestMapper]
	if ok {
		return c.(meta.RESTMapper), nil
	}
	dc, err := GetDiscoveryClient()
	if err != nil {
		return nil, err
	}
	cacher := memory.NewMemCacheClient(dc)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cacher)
	s[RestMapper] = mapper
	return mapper, nil
}

// GetRestClient returns a raw REST client instance
func GetRestClient() (rest.Interface, error) {
	s := GetGlobalState()
	c, ok := s[RestClient]
	if ok {
		return c.(rest.Interface), nil
	}
	clientConfig, err := GetClientConfig()
	if err != nil {
		return nil, err
	}
	restClient, err := rest.UnversionedRESTClientFor(clientConfig)
	if err != nil {
		return nil, err
	}
	s[RestClient] = restClient
	return restClient, nil
}

// GetOAPIFoundry returns an interface to request cty types from an OpenAPI spec
func GetOAPIFoundry() (foundry.Foundry, error) {
	s := GetGlobalState()

	f, ok := s[OAPIFoundry]

	if ok {
		return f.(foundry.Foundry), nil
	}

	rc, err := GetRestClient()
	if err != nil {
		return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
	}

	rq := rc.Verb("GET").Timeout(10*time.Second).AbsPath("openapi", "v2")
	rs, err := rq.DoRaw(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed get OpenAPI spec: %s", err)
	}

	oapif, err := foundry.NewFoundryFromSpecV2(rs)
	if err != nil {
		return nil, fmt.Errorf("failed construct OpenAPI foundry: %s", err)
	}

	s[OAPIFoundry] = oapif

	return oapif, nil
}
