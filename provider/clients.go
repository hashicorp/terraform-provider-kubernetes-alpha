package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/alexsomesan/openapi-cty/foundry"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

var providerState map[string]interface{}

// keys into the provider state storage
const (
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

// GetDynamicClient returns a configured unstructured (dynamic) client instance
func GetDynamicClient() (dynamic.Interface, error) {
	s := GetProviderState()
	c, ok := s[DynamicClient]
	if !ok {
		return nil, fmt.Errorf("no dynamic client configured")
	}
	return c.(dynamic.Interface), nil
}

// GetDiscoveryClient returns a configured discyovery client instance
func GetDiscoveryClient() (discovery.DiscoveryInterface, error) {
	s := GetProviderState()
	c, ok := s[DiscoveryClient]
	if !ok {
		return nil, fmt.Errorf("no discovery client configured")
	}
	return c.(discovery.DiscoveryInterface), nil
}

// GetRestMapper returns a RESTMapper client instance
func GetRestMapper() (meta.RESTMapper, error) {
	s := GetProviderState()
	c, ok := s[RestMapper]
	if !ok {
		return nil, fmt.Errorf("no REST mapper client configured")
	}
	return c.(meta.RESTMapper), nil
}

// GetRestClient returns a raw REST client instance
func GetRestClient() (rest.Interface, error) {
	s := GetProviderState()
	c, ok := s[RestClient]
	if !ok {
		return nil, fmt.Errorf("no REST client configured")
	}
	return c.(rest.Interface), nil
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
