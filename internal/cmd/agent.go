package cmd

import (
	"fmt"
	"os"

	"github.com/ankur/hass-cli/internal/outfmt"
)

// AgentCmd provides agent utilities.
type AgentCmd struct {
	ExitCodes ExitCodesCmd `cmd:"" name:"exit-codes" help:"List exit codes and their meanings."`
	Schema    SchemaCmd    `cmd:"" help:"Show CLI schema."`
}

// exitCode describes a CLI exit code.
type exitCode struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Meaning string `json:"meaning"`
}

var exitCodes = []exitCode{
	{0, "ok", "Command completed successfully."},
	{1, "error", "General error (API failure, network issue, etc.)."},
	{2, "usage", "Usage error (bad arguments, missing flags)."},
}

// ExitCodesCmd lists exit codes and their meanings.
type ExitCodesCmd struct{}

func (e *ExitCodesCmd) Run(globals *Globals) error {
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(exitCodes, os.Stdout)
	case outfmt.Plain:
		fields := make([][2]string, 0, len(exitCodes))
		for _, ec := range exitCodes {
			fields = append(fields, [2]string{ec.Name, ec.Meaning})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		for _, ec := range exitCodes {
			fmt.Fprintf(os.Stdout, "%-2d  %-8s  %s\n", ec.Code, ec.Name, ec.Meaning)
		}
	}
	return nil
}

// SchemaCmd shows a basic schema of the CLI.
type SchemaCmd struct{}

func (s *SchemaCmd) Run(globals *Globals) error {
	schema := map[string]any{
		"name":    "hass-cli",
		"version": Version,
		"commands": []string{
			"on", "off", "toggle", "get", "ls",
			"states", "services", "events", "automations",
			"areas", "entity", "history", "logbook", "config",
			"calendars", "template", "agent", "setup", "version",
		},
		"global_flags": []string{"--url", "--token", "--json", "--plain"},
	}
	outfmt.OutputJSON(schema, os.Stdout)
	return nil
}
