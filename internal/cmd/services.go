package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// ServicesCmd groups service subcommands.
type ServicesCmd struct {
	List ServicesListCmd `cmd:"" help:"List available services."`
	Call ServicesCallCmd `cmd:"" help:"Call a service."`
}

// ServicesListCmd lists all services, optionally filtered by domain.
type ServicesListCmd struct {
	Domain string `name:"domain" short:"d" help:"Filter by domain."`
}

func (c *ServicesListCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	domains, err := client.GetServices(context.Background())
	if err != nil {
		return fmt.Errorf("list services: %w", err)
	}

	if c.Domain != "" {
		filtered := domains[:0]
		for _, d := range domains {
			if d.Domain == c.Domain {
				filtered = append(filtered, d)
			}
		}
		domains = filtered
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(domains, os.Stdout)
	case outfmt.Plain:
		fields := [][2]string{}
		for _, d := range domains {
			keys := make([]string, 0, len(d.Services))
			for k := range d.Services {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				svc := d.Services[k]
				name := fmt.Sprintf("%s.%s", d.Domain, k)
				fields = append(fields, [2]string{name, svc.Description})
			}
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		for _, d := range domains {
			fmt.Fprintf(os.Stdout, "Domain: %s\n", d.Domain)
			keys := make([]string, 0, len(d.Services))
			for k := range d.Services {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				svc := d.Services[k]
				fmt.Fprintf(os.Stdout, "  %-30s  %s\n", k, svc.Description)
			}
		}
	}
	return nil
}

// ServicesCallCmd calls a service.
type ServicesCallCmd struct {
	Domain   string `arg:"" help:"Service domain."`
	Service  string `arg:"" help:"Service name."`
	EntityID string `name:"entity" short:"e" help:"Entity ID to target."`
	Data     string `name:"data" short:"d" help:"JSON data to pass to the service."`
}

func (c *ServicesCallCmd) Run(globals *Globals) error {
	data := map[string]any{}

	if c.Data != "" {
		if err := json.Unmarshal([]byte(c.Data), &data); err != nil {
			return fmt.Errorf("parse --data JSON: %w", err)
		}
	}

	if c.EntityID != "" {
		data["entity_id"] = c.EntityID
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	states, err := client.CallService(context.Background(), c.Domain, c.Service, data)
	if err != nil {
		return fmt.Errorf("call service %s.%s: %w", c.Domain, c.Service, err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(states, os.Stdout)
	case outfmt.Plain:
		for _, s := range states {
			outfmt.OutputPlain([][2]string{{s.EntityID, s.State}}, os.Stdout)
		}
	default:
		if len(states) == 0 {
			fmt.Fprintf(os.Stdout, "Service %s.%s called successfully.\n", c.Domain, c.Service)
		} else {
			for _, s := range states {
				fmt.Fprintf(os.Stdout, "%-50s  %s\n", s.EntityID, s.State)
			}
		}
	}
	return nil
}
