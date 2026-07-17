// Package main is the entrypoint for the 42 CLI.
package main

import (
	"os"

	"github.com/joaodiniz/42cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
