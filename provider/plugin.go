package provider

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/tfplugin5"
	"google.golang.org/grpc"
)

// Serve is the provider entry point
func Serve() {
	handshake := plugin.HandshakeConfig{
		ProtocolVersion: 5,
		// The magic cookie values should NEVER be changed.
		MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
		MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshake,
		GRPCServer:      plugin.DefaultGRPCServer,
		Plugins: plugin.PluginSet{
			"provider": &grpcPlugin{
				providerServer: &RawProviderServer{},
			},
		},
	})
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
