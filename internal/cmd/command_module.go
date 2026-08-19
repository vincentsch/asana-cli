// Package cmd builds Asana's domain command tree through rungrad modules.
package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vincentsch/rungrad"
)

type asanaCommandModule struct {
	commands []*rungrad.Command
	specs    []rungrad.CommandSpec
}

func normalizeExamples(value string) []string {
	var examples []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		examples = append(examples, line)
	}
	return examples
}

func newAsanaCommandModule(adapter *asanaRuntimeAdapter) asanaCommandModule {
	templates := asanaCommandTemplates()
	commands := make([]*rungrad.Command, 0, len(templates)+1)
	for _, template := range templates {
		commands = append(commands, buildRungradCommand(adapter, template))
	}
	updateCommand := newUpdateCommand()
	commands = append(commands, updateCommand)
	return asanaCommandModule{commands: commands, specs: buildCommandCatalog(templates, updateCommand)}
}

func (asanaCommandModule) Groups() []rungrad.Group { return nil }

func (module asanaCommandModule) Commands() []*rungrad.Command {
	return append([]*rungrad.Command(nil), module.commands...)
}

func (module asanaCommandModule) Catalog() []rungrad.CommandSpec {
	specs := make([]rungrad.CommandSpec, len(module.specs))
	for i, spec := range module.specs {
		specs[i] = spec
		specs[i].OutputModes = append([]string(nil), spec.OutputModes...)
		specs[i].Examples = append([]string(nil), spec.Examples...)
		specs[i].Related = append([]string(nil), spec.Related...)
	}
	return specs
}

func asanaCommandTemplates() []*cobra.Command {
	return []*cobra.Command{
		attachmentCmd,
		authCmd,
		completionCmd,
		configCmd,
		customFieldCmd,
		goalCmd,
		loginCmd,
		portfolioCmd,
		projectCmd,
		sectionCmd,
		tagCmd,
		taskCmd,
		teamCmd,
		userCmd,
		versionCmd,
		workspaceCmd,
	}
}

func buildRungradCommand(adapter *asanaRuntimeAdapter, template *cobra.Command) *rungrad.Command {
	contract := commandContractFor(template)
	command := &rungrad.Command{
		Use:          template.Use,
		Short:        template.Short,
		Long:         commandLong(template.Long),
		Examples:     normalizeExamples(template.Example),
		Related:      relatedCommands(template),
		OutputModes:  append([]string(nil), contract.outputModes...),
		Mutates:      contract.mutates,
		Destructive:  contract.destructive,
		RequiresAuth: contract.requiresAuth,
		SupportsMeta: contract.supportsMeta,
		GroupID:      template.GroupID,
		Args:         template.Args,
		Configure: func(cmd *cobra.Command) {
			cmd.Aliases = append([]string(nil), template.Aliases...)
			cmd.ValidArgs = append([]string(nil), template.ValidArgs...)
			cmd.ValidArgsFunction = template.ValidArgsFunction
			cmd.DisableFlagsInUseLine = template.DisableFlagsInUseLine
			cloneLocalFlags(cmd, template)
			if contract.destructive {
				cmd.Flags().Bool("confirm", false, "Confirm the destructive action without a prompt")
			}
		},
	}

	if template.RunE != nil || template.Run != nil {
		command.Run = func(factory *rungrad.Factory, cmd *cobra.Command, args []string) error {
			if err := bindCommandRuntime(cmd, factory, adapter); err != nil {
				return adaptCommandError(err)
			}
			run := template.RunE
			if run == nil {
				run = func(cmd *cobra.Command, args []string) error {
					template.Run(cmd, args)
					return nil
				}
			}
			return adaptCommandError(runCommandWithOutput(factory, cmd, args, run))
		}
	}
	for _, child := range template.Commands() {
		command.AddCommand(buildRungradCommand(adapter, child))
	}
	return command
}

// cloneLocalFlags gives every rungrad-built tree independent flag values.
func cloneLocalFlags(target, template *cobra.Command) {
	template.LocalNonPersistentFlags().VisitAll(func(flag *pflag.Flag) {
		flags := target.Flags()
		switch flag.Value.Type() {
		case "bool":
			value, _ := strconv.ParseBool(flag.DefValue)
			flags.BoolP(flag.Name, flag.Shorthand, value, flag.Usage)
		case "float64":
			value, _ := strconv.ParseFloat(flag.DefValue, 64)
			flags.Float64P(flag.Name, flag.Shorthand, value, flag.Usage)
		case "int":
			value, _ := strconv.Atoi(flag.DefValue)
			flags.IntP(flag.Name, flag.Shorthand, value, flag.Usage)
		case "string":
			flags.StringP(flag.Name, flag.Shorthand, flag.DefValue, flag.Usage)
		case "stringSlice":
			flags.StringSliceP(flag.Name, flag.Shorthand, nil, flag.Usage)
			if flag.DefValue != "[]" {
				_ = flags.Set(flag.Name, flag.DefValue)
			}
		default:
			panic("asana: unsupported command flag type " + flag.Value.Type())
		}
		cloned := flags.Lookup(flag.Name)
		cloned.DefValue = flag.DefValue
		cloned.NoOptDefVal = flag.NoOptDefVal
		cloned.Hidden = flag.Hidden
		cloned.Deprecated = flag.Deprecated
		if flag.Annotations != nil {
			cloned.Annotations = make(map[string][]string, len(flag.Annotations))
			for key, values := range flag.Annotations {
				cloned.Annotations[key] = append([]string(nil), values...)
			}
		}
	})
}
