package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// LogbookCmd retrieves logbook entries.
type LogbookCmd struct {
	EntityID string `name:"entity" short:"e" help:"Filter by entity ID."`
	Start    string `name:"start" help:"Start time (RFC3339). Default: 24 hours ago."`
}

func (c *LogbookCmd) Run(globals *Globals) error {
	startTime := time.Now().Add(-24 * time.Hour)
	if c.Start != "" {
		t, err := time.Parse(time.RFC3339, c.Start)
		if err != nil {
			return fmt.Errorf("parse --start time: %w", err)
		}
		startTime = t
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	entries, err := client.GetLogbook(context.Background(), c.EntityID, &startTime)
	if err != nil {
		return fmt.Errorf("get logbook: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(entries, os.Stdout)
	case outfmt.Plain:
		fields := make([][2]string, 0, len(entries))
		for _, e := range entries {
			fields = append(fields, [2]string{e.When, e.Name})
			if e.EntityID != "" {
				fields = append(fields, [2]string{e.EntityID, e.Message})
			} else {
				fields = append(fields, [2]string{e.Name, e.Message})
			}
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		printLogEntries(entries)
	}
	return nil
}

func printLogEntries(entries []hassapi.LogEntry) {
	for _, e := range entries {
		entityPart := ""
		if e.EntityID != "" {
			entityPart = " [" + e.EntityID + "]"
		}
		fmt.Fprintf(os.Stdout, "%s  %s%s: %s\n", e.When, e.Name, entityPart, e.Message)
	}
}
