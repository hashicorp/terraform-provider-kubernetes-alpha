package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
	"google.golang.org/grpc"
)

var providerName = "hashicorp/kubernetes-alpha"

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion: 5,
	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}

// Serve is the default entrypoint for the provider.
func Serve() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		GRPCServer:      plugin.DefaultGRPCServer,
		Plugins: plugin.PluginSet{
			"provider": &grpcPlugin{
				providerServer: &RawProviderServer{},
			},
		},
	})
}

// ServeReattach is the entrypoint for manually starting the provider
// as a process in reattach mode for debugging.
func ServeReattach() {
	reattachConfigCh := make(chan *plugin.ReattachConfig)
	go func() {
		reattachConfig, err := waitForReattachConfig(reattachConfigCh)
		if err != nil {
			fmt.Printf("Error getting reattach config: %s\n", err)
			return
		}
		printReattachConfig(reattachConfig)
	}()

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		GRPCServer:      plugin.DefaultGRPCServer,
		Plugins: plugin.PluginSet{
			"provider": &grpcPlugin{
				providerServer: &RawProviderServer{},
			},
		},
		Test: &plugin.ServeTestConfig{
			ReattachConfigCh: reattachConfigCh,
		},
		Logger: hclog.FromStandardLogger(Dlog, &hclog.LoggerOptions{
			JSONFormat: false,
			Level:      hclog.Debug,
		}),
	})
}

// ServeTest is for serving the provider in-process when testing.
// Returns a ReattachInfo or an error.
func ServeTest(ctx context.Context) (tfexec.ReattachInfo, error) {
	reattachConfigCh := make(chan *plugin.ReattachConfig)

	go plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		GRPCServer:      plugin.DefaultGRPCServer,
		Plugins: plugin.PluginSet{
			"provider": &grpcPlugin{
				providerServer: &RawProviderServer{},
			},
		},
		Test: &plugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: reattachConfigCh,
		},
		Logger: hclog.FromStandardLogger(Dlog, &hclog.LoggerOptions{
			JSONFormat: false,
			Level:      hclog.Info,
		}),
	})

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

type grpcPlugin struct {
	plugin.Plugin
	providerServer *RawProviderServer
}

func (p *grpcPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	tfplugin5.RegisterProviderServer(s, p.providerServer)
	return nil
}

func (p *grpcPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	//lintignore:R009
	panic("This is a plugin - it cannot implement GRPCClient")
}
