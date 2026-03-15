package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// StatesCmd groups state subcommands.
type StatesCmd struct {
	List   StatesListCmd   `cmd:"" help:"List all entity states."`
	Get    StatesGetCmd    `cmd:"" help:"Get state of a single entity."`
	Set    StatesSetCmd    `cmd:"" help:"Create or update an entity state."`
	Delete StatesDeleteCmd `cmd:"" help:"Delete an entity state."`
}

// StatesListCmd lists all or filtered entity states.
type StatesListCmd struct {
	Domain string `name:"domain" short:"d" help:"Filter by domain."`
}

func (c *StatesListCmd) Run(globals *Globals) error {
	return listStates(globals, c.Domain)
}

// StatesGetCmd gets a single entity's state.
type StatesGetCmd struct {
	EntityID string `arg:"" help:"Entity ID."`
}

func (c *StatesGetCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	state, err := client.GetState(context.Background(), c.EntityID)
	if err != nil {
		return fmt.Errorf("get state %s: %w", c.EntityID, err)
	}
	printState(globals, state)
	return nil
}

// StatesSetCmd creates or updates an entity state.
type StatesSetCmd struct {
	EntityID   string   `arg:"" help:"Entity ID."`
	State      string   `arg:"" help:"New state value."`
	Attributes []string `name:"attr" help:"Attributes as key=value pairs." short:"a"`
}

func (c *StatesSetCmd) Run(globals *Globals) error {
	attrs := map[string]any{}
	for _, a := range c.Attributes {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid attribute format %q, expected key=value", a)
		}
		attrs[parts[0]] = parts[1]
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	state, err := client.SetState(context.Background(), c.EntityID, c.State, attrs)
	if err != nil {
		return fmt.Errorf("set state %s: %w", c.EntityID, err)
	}
	printState(globals, state)
	return nil
}

// StatesDeleteCmd deletes an entity's state.
type StatesDeleteCmd struct {
	EntityID string `arg:"" help:"Entity ID to delete."`
}

func (c *StatesDeleteCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	if err := client.DeleteState(context.Background(), c.EntityID); err != nil {
		return fmt.Errorf("delete state %s: %w", c.EntityID, err)
	}
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"deleted": c.EntityID}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"deleted", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Deleted entity: %s\n", c.EntityID)
	}
	return nil
}
