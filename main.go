// Package main is the entrypoint for the lightyear CLI.
package main

import (
	"os"

	"github.com/nvizble/Lightyear42/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
