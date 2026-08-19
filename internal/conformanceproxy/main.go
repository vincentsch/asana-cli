// Command conformanceproxy gives the scorer a controlled environment for asana.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var apiBaseURL string

func main() {
	if apiBaseURL == "" {
		fmt.Fprintln(os.Stderr, "conformance proxy is missing its build configuration")
		os.Exit(2)
	}
	targetName := "asana-target"
	if runtime.GOOS == "windows" {
		targetName += ".exe"
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	targetPath := filepath.Join(filepath.Dir(executable), targetName)

	args := os.Args[1:]
	env := setEnvironment(os.Environ(), "ASANA_API_BASE_URL", apiBaseURL)
	if _, hasToken := os.LookupEnv("ASANA_TOKEN"); !hasToken && !isMissingCredentialProbe(args) {
		env = setEnvironment(env, "ASANA_TOKEN", "local-mock-token")
	}

	command := exec.Command(targetPath, args...)
	command.Env = env
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func isMissingCredentialProbe(args []string) bool {
	return len(args) >= 2 && args[0] == "user" && args[1] == "me"
}

func setEnvironment(env []string, key, value string) []string {
	prefix := key + "="
	for index, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			out := append([]string(nil), env...)
			out[index] = prefix + value
			return out
		}
	}
	return append(env, prefix+value)
}
