package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	tf5server "github.com/hashicorp/terraform-plugin-go/tfprotov5/server"
)

var providerName = "registry.terraform.io/hashicorp/kubernetes-alpha"

// Serve is the default entrypoint for the provider.
func Serve(ctx context.Context) error {
	return tf5server.Serve(providerName, func() tfprotov5.ProviderServer { return &(RawProviderServer{}) })
}

// ServeReattach is the entrypoint for manually starting the provider
// as a process in reattach mode for debugging.
func ServeReattach(ctx context.Context) error {
	reattachConfigCh := make(chan *plugin.ReattachConfig)
	go func() {
		reattachConfig, err := waitForReattachConfig(reattachConfigCh)
		if err != nil {
			fmt.Printf("Error getting reattach config: %s\n", err)
			return
		}
		printReattachConfig(reattachConfig)
	}()

	logger := hclog.FromStandardLogger(Dlog, &hclog.LoggerOptions{
		JSONFormat: false,
		Level:      hclog.Debug,
	})

	return tf5server.Serve(providerName,
		func() tfprotov5.ProviderServer { return &(RawProviderServer{}) },
		tf5server.WithDebug(ctx, reattachConfigCh, nil),
		tf5server.WithGoPluginLogger(logger),
	)
}

// ServeTest is for serving the provider in-process when testing.
// Returns a ReattachInfo or an error.
func ServeTest(ctx context.Context) (tfexec.ReattachInfo, error) {
	reattachConfigCh := make(chan *plugin.ReattachConfig)

	logger := hclog.FromStandardLogger(Dlog, &hclog.LoggerOptions{
		JSONFormat: false,
		Level:      hclog.Debug,
	})

	go tf5server.Serve(providerName,
		func() tfprotov5.ProviderServer { return &(RawProviderServer{}) },
		tf5server.WithDebug(ctx, reattachConfigCh, nil),
		tf5server.WithGoPluginLogger(logger),
	)

	reattachConfig, err := waitForReattachConfig(reattachConfigCh)
	if err != nil {
		return nil, fmt.Errorf("Error getting reattach config: %s", err)
	}

	return map[string]tfexec.ReattachConfig{
		providerName: convertReattachConfig(reattachConfig),
	}, nil
}

// convertReattachConfig converts plugin.ReattachConfig to tfexec.ReattachConfig
func convertReattachConfig(reattachConfig *plugin.ReattachConfig) tfexec.ReattachConfig {
	return tfexec.ReattachConfig{
		Protocol: string(reattachConfig.Protocol),
		Pid:      reattachConfig.Pid,
		Test:     true,
		Addr: tfexec.ReattachConfigAddr{
			Network: reattachConfig.Addr.Network(),
			String:  reattachConfig.Addr.String(),
		},
	}
}

// printReattachConfig prints the line the user needs to copy and paste
// to set the TF_REATTACH_PROVIDERS variable
func printReattachConfig(config *plugin.ReattachConfig) {
	reattachStr, err := json.Marshal(map[string]tfexec.ReattachConfig{
		providerName: convertReattachConfig(config),
	})
	if err != nil {
		fmt.Printf("Error building reattach string: %s", err)
		return
	}
	fmt.Printf("# Provider server started\nexport TF_REATTACH_PROVIDERS='%s'\n", string(reattachStr))
}

// waitForReattachConfig blocks until a ReattachConfig is recieved on the
// supplied channel or times out after 2 seconds.
func waitForReattachConfig(ch chan *plugin.ReattachConfig) (*plugin.ReattachConfig, error) {
	select {
	case config := <-ch:
		return config, nil
	case <-time.After(2 * time.Second):
		return nil, fmt.Errorf("timeout while waiting for reattach configuration")
	}
}
