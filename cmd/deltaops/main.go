package main

import (
	"os"

	"deltaops/internal/cli"
)

var (
	version = "dev"
	commit  = ""
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Options{Version: version, Commit: commit}))
}
