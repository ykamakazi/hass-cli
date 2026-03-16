package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// AutomationsCmd groups automation subcommands.
type AutomationsCmd struct {
	List    AutomationsListCmd    `cmd:"" help:"List all automations."`
	Get     AutomationsGetCmd     `cmd:"" help:"Get automation state and config."`
	Config  AutomationsConfigCmd  `cmd:"" help:"Show raw automation config JSON."`
	Update  AutomationsUpdateCmd  `cmd:"" help:"Update an automation config."`
	Trigger AutomationsTriggerCmd `cmd:"" help:"Trigger an automation."`
	Enable  AutomationsEnableCmd  `cmd:"" help:"Enable an automation."`
	Disable AutomationsDisableCmd `cmd:"" help:"Disable an automation."`
	Delete  AutomationsDeleteCmd  `cmd:"" help:"Delete an automation."`
}

// automationIDFromState extracts the numeric config ID from a state's attributes.
func automationIDFromState(entityID string, state *hassapi.State) (string, error) {
	id, ok := state.Attributes["id"]
	if !ok {
		return "", fmt.Errorf("automation %s has no id attribute (may be defined in YAML, not UI)", entityID)
	}
	switch v := id.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// automationID resolves an automation entity_id to its numeric config ID
// by reading the "id" attribute from the entity state.
func automationID(client *hassapi.Client, entityID string) (string, error) {
	if !strings.HasPrefix(entityID, "automation.") {
		entityID = "automation." + entityID
	}
	state, err := client.GetState(context.Background(), entityID)
	if err != nil {
		return "", fmt.Errorf("get state for %s: %w", entityID, err)
	}
	return automationIDFromState(entityID, state)
}

// AutomationsListCmd lists all automations.
type AutomationsListCmd struct{}

func (c *AutomationsListCmd) Run(globals *Globals) error {
	client := hassapi.NewClient(globals.URL, globals.Token)
	states, err := client.GetStates(context.Background())
	if err != nil {
		return err
	}

	var automations []hassapi.State
	for _, s := range states {
		if strings.HasPrefix(s.EntityID, "automation.") {
			automations = append(automations, s)
		}
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(automations, os.Stdout)
	case outfmt.Plain:
		for _, a := range automations {
			name := a.Attributes["friendly_name"]
			lastTriggered := a.Attributes["last_triggered"]
			fmt.Fprintf(os.Stdout, "%s\t%s\t%v\t%v\n", a.EntityID, a.State, name, lastTriggered)
		}
	default:
		for _, a := range automations {
			name, _ := a.Attributes["friendly_name"].(string)
			lastTriggered, _ := a.Attributes["last_triggered"].(string)
			fmt.Fprintf(os.Stdout, "%-55s  %-8s  %s\n", a.EntityID, a.State, name)
			if lastTriggered != "" {
				fmt.Fprintf(os.Stdout, "  last triggered: %s\n", lastTriggered)
			}
		}
	}
	return nil
}

// AutomationsGetCmd shows state + full config for an automation.
type AutomationsGetCmd struct {
	EntityID string `arg:"" help:"Automation entity ID (e.g. automation.mirror_toggle)."`
}

func (c *AutomationsGetCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	client := hassapi.NewClient(globals.URL, globals.Token)

	// Fetch state once and extract the ID from it directly — no second round-trip.
	state, err := client.GetState(context.Background(), c.EntityID)
	if err != nil {
		return err
	}

	id, err := automationIDFromState(c.EntityID, state)
	if err != nil {
		// Still show state even if config not available
		printState(globals, state)
		return nil
	}

	cfg, err := client.GetAutomationConfig(context.Background(), id)
	if err != nil {
		printState(globals, state)
		return nil
	}

	result := map[string]any{
		"state":  state,
		"config": cfg,
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(result, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{
			{"entity_id", state.EntityID},
			{"state", state.State},
			{"friendly_name", fmt.Sprintf("%v", state.Attributes["friendly_name"])},
			{"last_triggered", fmt.Sprintf("%v", state.Attributes["last_triggered"])},
		}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Entity:        %s\n", state.EntityID)
		fmt.Fprintf(os.Stdout, "State:         %s\n", state.State)
		if name, ok := state.Attributes["friendly_name"].(string); ok {
			fmt.Fprintf(os.Stdout, "Name:          %s\n", name)
		}
		if lt, ok := state.Attributes["last_triggered"].(string); ok && lt != "" {
			fmt.Fprintf(os.Stdout, "Last triggered: %s\n", lt)
		}
		fmt.Fprintf(os.Stdout, "\nConfig:\n")
		b, _ := json.MarshalIndent(cfg, "  ", "  ")
		fmt.Fprintf(os.Stdout, "  %s\n", b)
	}
	return nil
}

// AutomationsConfigCmd prints the raw automation config JSON, ready to edit and pipe back.
type AutomationsConfigCmd struct {
	EntityID string `arg:"" help:"Automation entity ID."`
}

func (c *AutomationsConfigCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	client := hassapi.NewClient(globals.URL, globals.Token)

	id, err := automationID(client, c.EntityID)
	if err != nil {
		return err
	}

	cfg, err := client.GetAutomationConfig(context.Background(), id)
	if err != nil {
		return err
	}

	// Always output JSON regardless of mode — this is meant for editing/piping.
	outfmt.OutputJSON(cfg, os.Stdout)
	return nil
}

// AutomationsUpdateCmd updates an automation from a JSON string or file.
type AutomationsUpdateCmd struct {
	EntityID string `arg:"" help:"Automation entity ID."`
	Data     string `name:"data" short:"d" help:"JSON config string."`
	File     string `name:"file" short:"f" help:"Path to JSON config file (use - for stdin)."`
}

func (c *AutomationsUpdateCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	if c.Data == "" && c.File == "" {
		return fmt.Errorf("provide --data <json> or --file <path>")
	}

	var raw []byte
	var err error
	if c.File != "" {
		if c.File == "-" {
			raw, err = io.ReadAll(os.Stdin)
		} else {
			raw, err = os.ReadFile(c.File)
		}
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	} else {
		raw = []byte(c.Data)
	}

	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	id, err := automationID(client, c.EntityID)
	if err != nil {
		return err
	}

	// Ensure the id field matches.
	cfg["id"] = id

	result, err := client.UpdateAutomation(context.Background(), id, cfg)
	if err != nil {
		return fmt.Errorf("update automation: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(result, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"updated", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Updated automation: %s\n", c.EntityID)
	}
	return nil
}

// AutomationsTriggerCmd triggers an automation.
type AutomationsTriggerCmd struct {
	EntityID     string `arg:"" help:"Automation entity ID."`
	SkipCondition bool  `name:"skip-condition" help:"Skip conditions when triggering." default:"true"`
}

func (c *AutomationsTriggerCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	client := hassapi.NewClient(globals.URL, globals.Token)
	data := map[string]any{
		"entity_id":      c.EntityID,
		"skip_condition": c.SkipCondition,
	}
	_, err := client.CallService(context.Background(), "automation", "trigger", data)
	if err != nil {
		return fmt.Errorf("trigger %s: %w", c.EntityID, err)
	}
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"triggered": c.EntityID}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"triggered", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Triggered: %s\n", c.EntityID)
	}
	return nil
}

// AutomationsEnableCmd enables an automation.
type AutomationsEnableCmd struct {
	EntityID string `arg:"" help:"Automation entity ID."`
}

func (c *AutomationsEnableCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	client := hassapi.NewClient(globals.URL, globals.Token)
	_, err := client.CallService(context.Background(), "automation", "turn_on", map[string]any{"entity_id": c.EntityID})
	if err != nil {
		return fmt.Errorf("enable %s: %w", c.EntityID, err)
	}
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"enabled": c.EntityID}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"enabled", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Enabled: %s\n", c.EntityID)
	}
	return nil
}

// AutomationsDisableCmd disables an automation.
type AutomationsDisableCmd struct {
	EntityID string `arg:"" help:"Automation entity ID."`
}

func (c *AutomationsDisableCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	client := hassapi.NewClient(globals.URL, globals.Token)
	_, err := client.CallService(context.Background(), "automation", "turn_off", map[string]any{"entity_id": c.EntityID})
	if err != nil {
		return fmt.Errorf("disable %s: %w", c.EntityID, err)
	}
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"disabled": c.EntityID}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"disabled", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Disabled: %s\n", c.EntityID)
	}
	return nil
}

// AutomationsDeleteCmd deletes an automation.
type AutomationsDeleteCmd struct {
	EntityID string `arg:"" help:"Automation entity ID."`
}

func (c *AutomationsDeleteCmd) Run(globals *Globals) error {
	if !strings.HasPrefix(c.EntityID, "automation.") {
		c.EntityID = "automation." + c.EntityID
	}
	client := hassapi.NewClient(globals.URL, globals.Token)

	id, err := automationID(client, c.EntityID)
	if err != nil {
		return err
	}

	if err := client.DeleteAutomation(context.Background(), id); err != nil {
		return fmt.Errorf("delete automation: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"deleted": c.EntityID}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"deleted", c.EntityID}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "Deleted: %s\n", c.EntityID)
	}
	return nil
}
