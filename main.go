package main

import (
	"flag"
	"os"

	"github.com/alexsomesan/terraform-provider-raw/provider"
)

func main() {
	flag.Parse()

	defer provider.InitDevLog()()

	provider.Dlog.Printf("Starting up with command line: %#v\n", os.Args)

	provider.Serve()
}
