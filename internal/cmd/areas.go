package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ankur/hass-cli/internal/outfmt"
	"github.com/ankur/hass-cli/internal/wsapi"
)

// AreasCmd groups area registry subcommands.
type AreasCmd struct {
	List   AreasListCmd   `cmd:"" help:"List all areas."`
	Create AreasCreateCmd `cmd:"" help:"Create a new area."`
	Rename AreasRenameCmd `cmd:"" help:"Rename an area."`
	Delete AreasDeleteCmd `cmd:"" help:"Delete an area."`
}

func wsConnect(globals *Globals) (*wsapi.Client, error) {
	return wsapi.Connect(context.Background(), globals.URL, globals.Token, wsapi.DefaultDialer)
}

// AreasListCmd lists all areas.
type AreasListCmd struct{}

func (c *AreasListCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	areas, err := ws.ListAreas(context.Background())
	if err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(areas, os.Stdout)
	case outfmt.Plain:
		for _, a := range areas {
			outfmt.OutputPlain([][2]string{{"area_id", a.AreaID}, {"name", a.Name}}, os.Stdout)
		}
	default:
		for _, a := range areas {
			fmt.Fprintf(os.Stdout, "%-30s  %s\n", a.AreaID, a.Name)
		}
	}
	return nil
}

// AreasCreateCmd creates a new area.
type AreasCreateCmd struct {
	Name string `arg:"" help:"Name of the new area."`
}

func (c *AreasCreateCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	area, err := ws.CreateArea(context.Background(), c.Name)
	if err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(area, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Created area: %s (id: %s)\n", area.Name, area.AreaID)
	}
	return nil
}

// AreasRenameCmd renames an area.
type AreasRenameCmd struct {
	Area string `arg:"" help:"Area name or ID to rename."`
	Name string `arg:"" help:"New name."`
}

func (c *AreasRenameCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	areaID, err := ws.ResolveArea(context.Background(), c.Area)
	if err != nil {
		return err
	}

	area, err := ws.UpdateArea(context.Background(), areaID, c.Name)
	if err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(area, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Renamed area %s → %s\n", areaID, area.Name)
	}
	return nil
}

// AreasDeleteCmd deletes an area.
type AreasDeleteCmd struct {
	Area string `arg:"" help:"Area name or ID to delete."`
}

func (c *AreasDeleteCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	areaID, err := ws.ResolveArea(context.Background(), c.Area)
	if err != nil {
		return err
	}

	if err := ws.DeleteArea(context.Background(), areaID); err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"deleted": areaID}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Deleted area: %s\n", areaID)
	}
	return nil
}
