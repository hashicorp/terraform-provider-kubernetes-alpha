package main

import (
	"context"
	"flag"
	"os"

	"github.com/hashicorp/terraform-provider-kubernetes-alpha/provider"
)

func main() {
	var debug = flag.Bool("debug", false, "run the provider in re-attach mode")
	flag.Parse()
	ctx := context.Background()

	defer provider.InitDevLog()()

	provider.Dlog.Printf("Starting up with command line: %#v\n", os.Args)
	if *debug {
		provider.ServeReattach(ctx)
	} else {
		provider.Serve(ctx)
	}
}
