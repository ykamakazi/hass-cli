package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// ConfigCmd groups config subcommands.
type ConfigCmd struct {
	Get        ConfigGetCmd        `cmd:"" help:"Show HA configuration."`
	Check      ConfigCheckCmd      `cmd:"" help:"Validate configuration."`
	Components ConfigComponentsCmd `cmd:"" help:"List loaded components."`
	ErrorLog   ConfigErrorLogCmd   `cmd:"" name:"error-log" help:"Show error log."`
}

// ConfigGetCmd shows the HA configuration.
type ConfigGetCmd struct{}

func (c *ConfigGetCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	cfg, err := client.GetConfig(context.Background())
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(cfg, os.Stdout)
	case outfmt.Plain:
		keys := make([]string, 0, len(cfg))
		for k := range cfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fields := make([][2]string, 0, len(keys))
		for _, k := range keys {
			fields = append(fields, [2]string{k, fmt.Sprintf("%v", cfg[k])})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		keys := make([]string, 0, len(cfg))
		for k := range cfg {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stdout, "%-30s  %v\n", k, cfg[k])
		}
	}
	return nil
}

// ConfigCheckCmd validates the HA configuration.
type ConfigCheckCmd struct{}

func (c *ConfigCheckCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	result, err := client.CheckConfig(context.Background())
	if err != nil {
		return fmt.Errorf("check config: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(result, os.Stdout)
	case outfmt.Plain:
		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fields := make([][2]string, 0, len(keys))
		for _, k := range keys {
			fields = append(fields, [2]string{k, fmt.Sprintf("%v", result[k])})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		res, _ := result["result"].(string)
		errors, _ := result["errors"].(string)
		fmt.Fprintf(os.Stdout, "Result: %s\n", res)
		if errors != "" {
			fmt.Fprintf(os.Stdout, "Errors:\n%s\n", errors)
		}
	}
	return nil
}

// ConfigComponentsCmd lists loaded HA components.
type ConfigComponentsCmd struct{}

func (c *ConfigComponentsCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	components, err := client.GetComponents(context.Background())
	if err != nil {
		return fmt.Errorf("get components: %w", err)
	}

	sort.Strings(components)

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(components, os.Stdout)
	case outfmt.Plain:
		fields := make([][2]string, 0, len(components))
		for _, comp := range components {
			fields = append(fields, [2]string{"component", comp})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		for _, comp := range components {
			fmt.Fprintf(os.Stdout, "%s\n", comp)
		}
	}
	return nil
}

// ConfigErrorLogCmd shows the HA error log.
type ConfigErrorLogCmd struct{}

func (c *ConfigErrorLogCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	log, err := client.GetErrorLog(context.Background())
	if err != nil {
		return fmt.Errorf("get error log: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"log": log}, os.Stdout)
	default:
		fmt.Fprint(os.Stdout, log)
	}
	return nil
}
