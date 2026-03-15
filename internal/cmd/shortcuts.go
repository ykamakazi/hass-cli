package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// OnCmd turns on an entity.
type OnCmd struct {
	EntityID string `arg:"" help:"Entity ID to turn on."`
}

func (c *OnCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	data := map[string]any{"entity_id": c.EntityID}
	states, err := client.CallService("homeassistant", "turn_on", data)
	if err != nil {
		return fmt.Errorf("turn on %s: %w", c.EntityID, err)
	}
	printStates(globals, states)
	return nil
}

// OffCmd turns off an entity.
type OffCmd struct {
	EntityID string `arg:"" help:"Entity ID to turn off."`
}

func (c *OffCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	data := map[string]any{"entity_id": c.EntityID}
	states, err := client.CallService("homeassistant", "turn_off", data)
	if err != nil {
		return fmt.Errorf("turn off %s: %w", c.EntityID, err)
	}
	printStates(globals, states)
	return nil
}

// ToggleCmd toggles an entity.
type ToggleCmd struct {
	EntityID string `arg:"" help:"Entity ID to toggle."`
}

func (c *ToggleCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	data := map[string]any{"entity_id": c.EntityID}
	states, err := client.CallService("homeassistant", "toggle", data)
	if err != nil {
		return fmt.Errorf("toggle %s: %w", c.EntityID, err)
	}
	printStates(globals, states)
	return nil
}

// GetCmd gets the state of an entity.
type GetCmd struct {
	EntityID string `arg:"" help:"Entity ID to get state for."`
}

func (c *GetCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	state, err := client.GetState(c.EntityID)
	if err != nil {
		return fmt.Errorf("get state for %s: %w", c.EntityID, err)
	}
	printState(globals, state)
	return nil
}

// LsCmd lists all entity states, optionally filtered by domain.
type LsCmd struct {
	Domain string `name:"domain" short:"d" help:"Filter by domain (e.g. light, switch)."`
}

func (c *LsCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	states, err := client.GetStates()
	if err != nil {
		return fmt.Errorf("list states: %w", err)
	}

	if c.Domain != "" {
		filtered := states[:0]
		for _, s := range states {
			if strings.HasPrefix(s.EntityID, c.Domain+".") {
				filtered = append(filtered, s)
			}
		}
		states = filtered
	}

	printStates(globals, states)
	return nil
}

// printState outputs a single state in the configured format.
func printState(globals *Globals, state *hassapi.State) {
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(state, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{
			{"entity_id", state.EntityID},
			{"state", state.State},
			{"last_changed", state.LastChanged},
			{"last_updated", state.LastUpdated},
		}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "%-50s  %s\n", state.EntityID, state.State)
		if len(state.Attributes) > 0 {
			for k, v := range state.Attributes {
				fmt.Fprintf(os.Stdout, "  %-48s  %v\n", k, v)
			}
		}
	}
}

// printStates outputs multiple states in the configured format.
func printStates(globals *Globals, states []hassapi.State) {
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(states, os.Stdout)
	case outfmt.Plain:
		for _, s := range states {
			outfmt.OutputPlain([][2]string{
				{s.EntityID, s.State},
			}, os.Stdout)
		}
	default:
		for _, s := range states {
			fmt.Fprintf(os.Stdout, "%-50s  %s\n", s.EntityID, s.State)
		}
	}
}
