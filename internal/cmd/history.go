package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// HistoryCmd retrieves state history.
type HistoryCmd struct {
	EntityID        string `name:"entity" short:"e" help:"Filter by entity ID."`
	Start           string `name:"start" help:"Start time (RFC3339). Default: 1 hour ago."`
	End             string `name:"end" help:"End time (RFC3339)."`
	SignificantOnly bool   `name:"significant-only" help:"Only return significant state changes."`
}

func (c *HistoryCmd) Run(globals *Globals) error {
	startTime := time.Now().Add(-1 * time.Hour)
	if c.Start != "" {
		t, err := time.Parse(time.RFC3339, c.Start)
		if err != nil {
			return fmt.Errorf("parse --start time: %w", err)
		}
		startTime = t
	}

	var endTime *time.Time
	if c.End != "" {
		t, err := time.Parse(time.RFC3339, c.End)
		if err != nil {
			return fmt.Errorf("parse --end time: %w", err)
		}
		endTime = &t
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	history, err := client.GetHistory(c.EntityID, startTime, endTime, c.SignificantOnly)
	if err != nil {
		return fmt.Errorf("get history: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(history, os.Stdout)
	case outfmt.Plain:
		for _, series := range history {
			for _, s := range series {
				outfmt.OutputPlain([][2]string{
					{s.EntityID + "\t" + s.LastChanged, s.State},
				}, os.Stdout)
			}
		}
	default:
		for _, series := range history {
			if len(series) == 0 {
				continue
			}
			fmt.Fprintf(os.Stdout, "Entity: %s\n", series[0].EntityID)
			for _, s := range series {
				fmt.Fprintf(os.Stdout, "  %s  ->  %s\n", s.LastChanged, s.State)
			}
		}
	}
	return nil
}
