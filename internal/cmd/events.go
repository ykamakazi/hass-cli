package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// EventsCmd groups event subcommands.
type EventsCmd struct {
	List EventsListCmd `cmd:"" help:"List all events."`
	Fire EventsFireCmd `cmd:"" help:"Fire an event."`
}

// EventsListCmd lists all registered events.
type EventsListCmd struct{}

func (c *EventsListCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	events, err := client.GetEvents()
	if err != nil {
		return fmt.Errorf("list events: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(events, os.Stdout)
	case outfmt.Plain:
		fields := make([][2]string, 0, len(events))
		for _, e := range events {
			fields = append(fields, [2]string{e.Event, fmt.Sprintf("%d", e.ListenerCount)})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "%-50s  %s\n", "Event", "Listeners")
		fmt.Fprintf(os.Stdout, "%-50s  %s\n", "-----", "---------")
		for _, e := range events {
			fmt.Fprintf(os.Stdout, "%-50s  %d\n", e.Event, e.ListenerCount)
		}
	}
	return nil
}

// EventsFireCmd fires a Home Assistant event.
type EventsFireCmd struct {
	EventType string `arg:"" help:"Event type to fire."`
	Data      string `name:"data" short:"d" help:"JSON event data."`
}

func (c *EventsFireCmd) Run(globals *Globals) error {
	data := map[string]any{}
	if c.Data != "" {
		if err := json.Unmarshal([]byte(c.Data), &data); err != nil {
			return fmt.Errorf("parse --data JSON: %w", err)
		}
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	if err := client.FireEvent(c.EventType, data); err != nil {
		return fmt.Errorf("fire event %s: %w", c.EventType, err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"fired": c.EventType}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"fired", c.EventType}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Event fired: %s\n", c.EventType)
	}
	return nil
}
