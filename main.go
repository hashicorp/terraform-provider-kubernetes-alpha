package main

import (
	"flag"
	"os"

	"github.com/hashicorp/terraform-provider-kubernetes-alpha/provider"
)

func main() {
	var dFlag = flag.Bool("debug", false, "run the provider in re-attach mode")
	flag.Parse()

	defer provider.InitDevLog()()

	provider.Dlog.Printf("Starting up with command line: %#v\n", os.Args)

	if *dFlag {
		provider.DebugServe()
	} else {
		provider.Serve()
	}
}
