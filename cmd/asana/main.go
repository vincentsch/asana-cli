// Command asana is the CLI entry point.
package main

import (
	"os"

	"github.com/vincentsch/asana-cli/internal/cmd"
)

func main() {
	app := cmd.NewApp()
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
