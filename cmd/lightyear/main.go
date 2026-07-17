// Package main is the entrypoint for the lightyear CLI.
//
// Instalação via Go (binário nomeado lightyear):
//
//	go install github.com/nvizble/Lightyear42/cmd/lightyear@latest
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
