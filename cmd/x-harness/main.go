package main

import (
	"os"

	"github.com/BrianNguyen29/x-harness/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
