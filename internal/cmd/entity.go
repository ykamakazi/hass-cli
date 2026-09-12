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
	Disable EntityDisableCmd `cmd:"" help:"Disable an entity so its integration stops providing it."`
	Enable  EntityEnableCmd  `cmd:"" help:"Re-enable a disabled entity."`
	Delete  EntityDeleteCmd  `cmd:"" help:"Remove an entity from the entity registry."`
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
			{"disabled_by", entry.DisabledBy},
			{"hidden_by", entry.HiddenBy},
		}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "entity_id:   %s\n", entry.EntityID)
		fmt.Fprintf(os.Stdout, "name:        %s\n", entry.Name)
		fmt.Fprintf(os.Stdout, "area_id:     %s\n", entry.AreaID)
		fmt.Fprintf(os.Stdout, "device_id:   %s\n", entry.DeviceID)
		fmt.Fprintf(os.Stdout, "platform:    %s\n", entry.Platform)
		fmt.Fprintf(os.Stdout, "disabled_by: %s\n", orDash(entry.DisabledBy))
		fmt.Fprintf(os.Stdout, "hidden_by:   %s\n", orDash(entry.HiddenBy))
	}
	return nil
}

// orDash renders an empty registry field as a dash for human output.
func orDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// setDisabled is the shared body of `entity disable` and `entity enable`.
func setDisabled(globals *Globals, entityID string, disabled bool) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	res, err := ws.SetEntityDisabled(context.Background(), entityID, disabled)
	if err != nil {
		return err
	}

	verb := "Enabled"
	if disabled {
		verb = "Disabled"
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]any{
			"entity_id":       entityID,
			"disabled":        disabled,
			"disabled_by":     res.Entry.DisabledBy,
			"require_restart": res.RequireRestart,
			"reload_delay":    res.ReloadDelay,
		}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{
			{"entity_id", entityID},
			{"disabled", fmt.Sprintf("%t", disabled)},
			{"disabled_by", res.Entry.DisabledBy},
			{"require_restart", fmt.Sprintf("%t", res.RequireRestart)},
		}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "%s %s\n", verb, entityID)
		switch {
		case res.RequireRestart:
			fmt.Fprintln(os.Stdout, "Restart Home Assistant for this to take effect.")
		case res.ReloadDelay > 0:
			fmt.Fprintf(os.Stdout, "Takes effect once the integration reloads (~%ds).\n", res.ReloadDelay)
		}
	}
	return nil
}

// EntityDisableCmd disables an entity.
type EntityDisableCmd struct {
	EntityID string `arg:"" help:"Entity ID."`
}

func (c *EntityDisableCmd) Run(globals *Globals) error {
	return setDisabled(globals, c.EntityID, true)
}

// EntityEnableCmd re-enables a disabled entity.
type EntityEnableCmd struct {
	EntityID string `arg:"" help:"Entity ID."`
}

func (c *EntityEnableCmd) Run(globals *Globals) error {
	return setDisabled(globals, c.EntityID, false)
}

// EntityDeleteCmd removes an entity from the entity registry.
type EntityDeleteCmd struct {
	EntityID string `arg:"" help:"Entity ID."`
}

func (c *EntityDeleteCmd) Run(globals *Globals) error {
	ws, err := wsConnect(globals)
	if err != nil {
		return err
	}
	defer ws.Close()

	if err := ws.DeleteEntity(context.Background(), c.EntityID); err != nil {
		return err
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"deleted": c.EntityID}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"deleted", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Deleted entity: %s\n", c.EntityID)
		fmt.Fprintln(os.Stdout, "If its integration still discovers it, it will come back — use `hass entity disable` instead.")
	}
	return nil
}
