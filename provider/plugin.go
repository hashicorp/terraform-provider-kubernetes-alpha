package provider

import (
	"context"
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

// Serve is the provider entry point
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
