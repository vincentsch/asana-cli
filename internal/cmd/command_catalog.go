// Package cmd declares the complete user-visible Asana command catalog.
package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad"
)

var unauthenticatedCommandPaths = stringSet(
	"auth login",
	"completion",
	"config set",
	"login",
	"update",
	"version",
)

var mutatingCommandPaths = stringSet(
	"attachment delete", "attachment upload",
	"auth login",
	"config set",
	"custom-field create", "custom-field delete", "custom-field update",
	"goal create", "goal delete", "goal metric set", "goal update",
	"login",
	"portfolio create", "portfolio delete", "portfolio project add", "portfolio project remove",
	"project create", "project delete", "project duplicate", "project member add", "project member remove", "project update",
	"section create", "section delete", "section move", "section update",
	"tag create", "tag delete", "tag update",
	"task comment", "task comment delete", "task comment update", "task create", "task delete",
	"task dependency add", "task dependency remove", "task dependent add", "task dependent remove",
	"task done", "task duplicate", "task follower add", "task follower remove", "task move",
	"task parent set", "task project add", "task project remove", "task reopen", "task subtask create",
	"task tag add", "task tag remove", "task update",
	"team create", "team member add", "team member remove",
	"update",
	"workspace user add", "workspace user remove",
)

var destructiveCommandPaths = stringSet(
	"attachment delete",
	"custom-field delete",
	"goal delete",
	"portfolio delete", "portfolio project remove",
	"project delete", "project member remove",
	"section delete",
	"tag delete",
	"task comment delete", "task delete", "task dependency remove", "task dependent remove",
	"task follower remove", "task project remove", "task tag remove",
	"team member remove",
	"workspace user remove",
)

type commandContract struct {
	requiresAuth bool
	mutates      bool
	destructive  bool
	supportsMeta bool
	outputModes  []string
}

func commandContractFor(command *cobra.Command) commandContract {
	path := commandPath(command)
	executable := command.Run != nil || command.RunE != nil
	structured := executable && path != "completion"
	requiresAuth := executable && !unauthenticatedCommandPaths[path]

	contract := commandContract{
		requiresAuth: requiresAuth,
		mutates:      mutatingCommandPaths[path],
		destructive:  destructiveCommandPaths[path],
		supportsMeta: structured && (requiresAuth || path == "config set"),
	}
	if structured {
		contract.outputModes = []string{
			rungrad.OutputModeHuman,
			rungrad.OutputModeJSON,
			rungrad.OutputModePlain,
			rungrad.OutputModeJQ,
			rungrad.OutputModeTemplate,
		}
	}
	return contract
}

func relatedCommands(command *cobra.Command) []string {
	const marker = "\n\nSee also: "
	start := strings.LastIndex(command.Long, marker)
	if start < 0 {
		return nil
	}
	valueStart := start + len(marker)
	valueEnd := len(command.Long)
	if newline := strings.IndexByte(command.Long[valueStart:], '\n'); newline >= 0 {
		valueEnd = valueStart + newline
	}

	var related []string
	for _, value := range strings.Split(command.Long[valueStart:valueEnd], ",") {
		if value = strings.TrimSpace(value); value != "" {
			related = append(related, value)
		}
	}
	if len(related) == 0 {
		return nil
	}

	return related
}

func commandLong(value string) string {
	const marker = "\n\nSee also: "
	start := strings.LastIndex(value, marker)
	if start < 0 {
		return value
	}
	valueStart := start + len(marker)
	valueEnd := len(value)
	if newline := strings.IndexByte(value[valueStart:], '\n'); newline >= 0 {
		valueEnd = valueStart + newline
	}
	return value[:start] + value[valueEnd:]
}

func buildCommandCatalog(roots []*cobra.Command, updateCommand *rungrad.Command) []rungrad.CommandSpec {
	commands := map[string]*cobra.Command{}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.Hidden || command.Name() == "help" || command.Name() == "__rungrad_manifest" {
			return
		}
		path := commandPath(command)
		commands[path] = command
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	paths := make([]string, 0, len(commands)+1)
	for path := range commands {
		paths = append(paths, path)
	}
	paths = append(paths, "update")
	sort.Strings(paths)
	for path, command := range commands {
		for _, related := range relatedCommands(command) {
			if _, ok := commands[related]; !ok && related != "update" {
				panic(fmt.Sprintf("asana: command %q references unknown related command %q", path, related))
			}
		}
	}

	specs := make([]rungrad.CommandSpec, 0, len(paths))
	for _, path := range paths {
		if path == "update" {
			specs = append(specs, rungrad.CommandSpec{
				Path:         path,
				Summary:      updateCommand.Short,
				GroupID:      updateCommand.GroupID,
				OutputModes:  append([]string(nil), updateCommand.OutputModes...),
				Examples:     append([]string(nil), updateCommand.Examples...),
				Related:      append([]string(nil), updateCommand.Related...),
				RequiresAuth: updateCommand.RequiresAuth,
				Mutates:      updateCommand.Mutates,
				Destructive:  updateCommand.Destructive,
				SupportsMeta: updateCommand.SupportsMeta,
			})
			continue
		}
		command := commands[path]
		if command == nil {
			panic(fmt.Sprintf("asana: catalog command %q is not present in the built tree", path))
		}
		contract := commandContractFor(command)
		specs = append(specs, rungrad.CommandSpec{
			Path:         path,
			Summary:      command.Short,
			GroupID:      command.GroupID,
			OutputModes:  append([]string(nil), contract.outputModes...),
			Examples:     normalizeExamples(command.Example),
			Related:      relatedCommands(command),
			RequiresAuth: contract.requiresAuth,
			Mutates:      contract.mutates,
			Destructive:  contract.destructive,
			SupportsMeta: contract.supportsMeta,
		})
	}
	return specs
}

func stringSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
