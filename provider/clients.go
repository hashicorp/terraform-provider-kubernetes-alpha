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
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var providerState map[string]interface{}

// keys into the provider state storage
const (
	ClientConfig    string = "CLIENTCONFIG"
	DynamicClient   string = "DYNAMICCLIENT"
	DiscoveryClient string = "DISCOVERYCLIENT"
	RestClient      string = "RESTCLIENT"
	RestMapper      string = "RESTMAPPER"
	SSPlanning      string = "SERVERSIDEPLANNING"
	OAPIFoundry     string = "OPENAPIFOUNDRY"
)

// GetProviderState returns a global state storage structure.
func GetProviderState() map[string]interface{} {
	if providerState == nil {
		providerState = make(map[string]interface{})
	}
	return providerState
}

// GetClientConfig returns the client.Config produced from the
// provider block attribues
func GetClientConfig() (*rest.Config, error) {
	s := GetProviderState()
	c, ok := s[ClientConfig]
	if !ok {
		return nil, fmt.Errorf("no client configuration")
	}
	return c.(*rest.Config), nil
}

// GetDynamicClient returns a configured unstructured (dynamic) client instance
func GetDynamicClient() (dynamic.Interface, error) {
	s := GetProviderState()
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

// GetDiscoveryClient returns a configured discyovery client instance
func GetDiscoveryClient() (discovery.DiscoveryInterface, error) {
	s := GetProviderState()
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
	s := GetProviderState()
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
	s := GetProviderState()
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
	s := GetProviderState()

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
