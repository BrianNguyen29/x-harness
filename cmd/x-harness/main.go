package main

import (
	"os"

	"github.com/BrianNguyen29/x-harness/internal/cli"
)

var version string

func main() {
	cli.SetVersion(version)
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
