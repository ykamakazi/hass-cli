package cmd

import (
	"fmt"
	"os"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// CalendarsCmd groups calendar subcommands.
type CalendarsCmd struct {
	List   CalendarsListCmd   `cmd:"" help:"List calendar entities."`
	Events CalendarsEventsCmd `cmd:"" help:"Get events from a calendar."`
}

// CalendarsListCmd lists all calendar entities.
type CalendarsListCmd struct{}

func (c *CalendarsListCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	calendars, err := client.GetCalendars()
	if err != nil {
		return fmt.Errorf("list calendars: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(calendars, os.Stdout)
	case outfmt.Plain:
		fields := make([][2]string, 0, len(calendars))
		for _, cal := range calendars {
			fields = append(fields, [2]string{cal.EntityID, cal.Name})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "%-50s  %s\n", "Entity ID", "Name")
		fmt.Fprintf(os.Stdout, "%-50s  %s\n", "---------", "----")
		for _, cal := range calendars {
			fmt.Fprintf(os.Stdout, "%-50s  %s\n", cal.EntityID, cal.Name)
		}
	}
	return nil
}

// CalendarsEventsCmd lists events for a specific calendar.
type CalendarsEventsCmd struct {
	CalendarID string `arg:"" help:"Calendar entity ID."`
	Start      string `name:"start" required:"" help:"Start date/time (RFC3339 or date)."`
	End        string `name:"end" required:"" help:"End date/time (RFC3339 or date)."`
}

func (c *CalendarsEventsCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	events, err := client.GetCalendarEvents(c.CalendarID, c.Start, c.End)
	if err != nil {
		return fmt.Errorf("get calendar events for %s: %w", c.CalendarID, err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(events, os.Stdout)
	case outfmt.Plain:
		fields := make([][2]string, 0, len(events))
		for _, e := range events {
			start := calendarEventTime(e)
			fields = append(fields, [2]string{start, e.Summary})
		}
		outfmt.OutputPlain(fields, os.Stdout)
	default:
		for _, e := range events {
			start := calendarEventTime(e)
			fmt.Fprintf(os.Stdout, "%s  %s\n", start, e.Summary)
			if e.Description != "" {
				fmt.Fprintf(os.Stdout, "  Description: %s\n", e.Description)
			}
			if e.Location != "" {
				fmt.Fprintf(os.Stdout, "  Location: %s\n", e.Location)
			}
		}
	}
	return nil
}

func calendarEventTime(e hassapi.CalendarEvent) string {
	if dt, ok := e.Start["dateTime"]; ok {
		return dt
	}
	if d, ok := e.Start["date"]; ok {
		return d
	}
	return ""
}
