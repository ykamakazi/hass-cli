package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ankur/hass-cli/internal/outfmt"
)

// EntityCmd groups entity registry subcommands.
type EntityCmd struct {
	SetArea EntitySetAreaCmd `cmd:"" name:"set-area" help:"Assign an entity to an area."`
	Rename  EntityRenameCmd  `cmd:"" help:"Set the friendly name of an entity."`
	Info    EntityInfoCmd    `cmd:"" help:"Show entity registry entry."`
}

// EntitySetAreaCmd assigns an entity to an area.
type EntitySetAreaCmd struct {
	EntityID string `arg:"" help:"Entity ID (e.g. automation.sub_off)."`
	Area     string `arg:"" help:"Area name or ID (e.g. 'living room')."`
}

func (c *EntitySetAreaCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	areaID, err := ws.ResolveArea(context.Background(), c.Area)
	if err != nil {
		return err
	}

	entry, err := ws.SetEntityArea(context.Background(), c.EntityID, areaID)
	if err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(entry, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"entity_id", c.EntityID}, {"area_id", areaID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Set area for %s → %s\n", c.EntityID, areaID)
	}
	return nil
}

// EntityRenameCmd sets the friendly name of an entity.
type EntityRenameCmd struct {
	EntityID string `arg:"" help:"Entity ID."`
	Name     string `arg:"" help:"New friendly name."`
}

func (c *EntityRenameCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	entry, err := ws.RenameEntity(context.Background(), c.EntityID, c.Name)
	if err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(entry, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"entity_id", c.EntityID}, {"name", c.Name}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Renamed %s → %q\n", c.EntityID, c.Name)
	}
	return nil
}

// EntityInfoCmd shows the entity registry entry.
type EntityInfoCmd struct {
	EntityID string `arg:"" help:"Entity ID."`
}

func (c *EntityInfoCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	entry, err := ws.GetEntity(context.Background(), c.EntityID)
	if err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(entry, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{
			{"entity_id", entry.EntityID},
			{"name", entry.Name},
			{"area_id", entry.AreaID},
			{"device_id", entry.DeviceID},
			{"platform", entry.Platform},
		}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "entity_id:  %s\n", entry.EntityID)
		fmt.Fprintf(os.Stdout, "name:       %s\n", entry.Name)
		fmt.Fprintf(os.Stdout, "area_id:    %s\n", entry.AreaID)
		fmt.Fprintf(os.Stdout, "device_id:  %s\n", entry.DeviceID)
		fmt.Fprintf(os.Stdout, "platform:   %s\n", entry.Platform)
	}
	return nil
}
