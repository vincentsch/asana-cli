// Command generate-command-docs updates or checks the generated command manual.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vincentsch/asana-cli/internal/cmd"
	"github.com/vincentsch/asana-cli/internal/manualdocs"
)

func main() {
	check := flag.Bool("check", false, "check generated command docs without changing files")
	flag.Parse()

	const dir = "manual/commands"
	if *check {
		result, err := manualdocs.Check(cmd.NewApp(), dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !result.OK() {
			fmt.Fprintln(os.Stderr, result.String())
			os.Exit(1)
		}
		return
	}
	if err := manualdocs.Write(cmd.NewApp(), dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
