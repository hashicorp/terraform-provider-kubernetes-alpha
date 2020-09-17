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

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion: 5,
	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}

// Serve is the default provider entry point
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

// ServeTest is for serving the provider in-process when testing
// returns the reattach configuration or an error
func ServeTest() (tfexec.ReattachInfo, error) {
	reattachConfigCh := make(chan *plugin.ReattachConfig)

	go func() {
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
				Level:      hclog.Info,
			}),
		})
	}()

	select {
	case reattachConfig := <-reattachConfigCh:
		return map[string]tfexec.ReattachConfig{
			"hashicorp/kubernetes-alpha": {
				Protocol: string(reattachConfig.Protocol),
				Pid:      reattachConfig.Pid,
				Test:     true,
				Addr: tfexec.ReattachConfigAddr{
					Network: reattachConfig.Addr.Network(),
					String:  reattachConfig.Addr.String(),
				},
			},
		}, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout when starting the provider")
	}
}

// DebugServe is the provider entry point when running in stand-alone process mode
func DebugServe() {
	reattachCh := make(chan *plugin.ReattachConfig)

	go waitForReattachConfig(reattachCh)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		GRPCServer:      plugin.DefaultGRPCServer,
		Plugins: plugin.PluginSet{
			"provider": &grpcPlugin{
				providerServer: &RawProviderServer{},
			},
		},
		Test: &plugin.ServeTestConfig{
			ReattachConfigCh: reattachCh,
		},
		Logger: hclog.FromStandardLogger(Dlog, &hclog.LoggerOptions{
			JSONFormat: false,
			Level:      hclog.Debug,
		}),
	})
}

func waitForReattachConfig(ch chan *plugin.ReattachConfig) {
	var config *plugin.ReattachConfig
	select {
	case config = <-ch:
		reattachStr, err := json.Marshal(map[string]tfexec.ReattachConfig{
			"hashicorp/kubernetes-alpha": {
				Protocol: string(config.Protocol),
				Pid:      config.Pid,
				Test:     config.Test,
				Addr: tfexec.ReattachConfigAddr{
					String:  config.Addr.String(),
					Network: config.Addr.Network(),
				},
			},
		})
		if err != nil {
			fmt.Printf("Error building reattach string: %s", err)
			return
		}
		fmt.Printf("# Provider server started\nexport TF_REATTACH_PROVIDERS='%s'\n", string(reattachStr))
		return
	case <-time.After(2 * time.Second):
		fmt.Printf("Timeout while waiting for reattach configuration\n")
		return
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
