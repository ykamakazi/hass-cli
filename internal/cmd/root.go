package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/ankur/hass-cli/internal/config"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// CLI is the root command struct parsed by Kong.
type CLI struct {
	URL   string `name:"url" env:"HASS_URL" help:"Home Assistant URL." short:"u"`
	Token string `name:"token" env:"HASS_TOKEN" help:"Long-lived access token." short:"t"`
	JSON  bool   `name:"json" env:"HASS_JSON" help:"Output as JSON."`
	Plain bool   `name:"plain" env:"HASS_PLAIN" help:"Output as plain text (TSV)."`

	// Top-level shortcuts
	On     OnCmd     `cmd:"" help:"Turn on an entity."`
	Off    OffCmd    `cmd:"" help:"Turn off an entity."`
	Toggle ToggleCmd `cmd:"" help:"Toggle an entity."`
	Get    GetCmd    `cmd:"" help:"Get state of an entity." aliases:"show"`
	Ls     LsCmd     `cmd:"" help:"List entity states." aliases:"list"`

	// Subcommand groups
	States    StatesCmd    `cmd:"" help:"Manage entity states."`
	Services  ServicesCmd  `cmd:"" help:"List and call services."`
	Events    EventsCmd    `cmd:"" help:"List and fire events."`
	History   HistoryCmd   `cmd:"" help:"View state history."`
	Logbook   LogbookCmd   `cmd:"" help:"View logbook entries."`
	Config    ConfigCmd    `cmd:"" help:"Home Assistant configuration."`
	Calendars CalendarsCmd `cmd:"" help:"Calendar entities."`
	Template  TemplateCmd  `cmd:"" help:"Render templates."`
	Agent     AgentCmd     `cmd:"" help:"Agent utilities."`
	Setup     SetupCmd     `cmd:"" help:"Configure hass-cli interactively."`
	Version   VersionCmd   `cmd:"" help:"Show version information."`
}

// Globals holds resolved global options passed to each command's Run method.
type Globals struct {
	URL   string
	Token string
	Mode  outfmt.Mode
}

// noAuthCommands are commands that don't require URL/Token.
var noAuthCommands = []string{"version", "agent", "setup"}

func requiresAuth(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, c := range noAuthCommands {
		if lower == c || strings.HasPrefix(lower, c+" ") {
			return false
		}
	}
	return true
}

// Execute parses the CLI and runs the selected command.
func Execute() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("hass"),
		kong.Description("Home Assistant CLI — control your HA instance from the terminal."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
	)

	// Determine output mode.
	mode := outfmt.Human
	if cli.JSON {
		mode = outfmt.JSON
	} else if cli.Plain {
		mode = outfmt.Plain
	}

	// Load saved config, then let flags/env override.
	saved, _ := config.Load()

	url := cli.URL
	if url == "" {
		url = saved.URL
	}
	token := cli.Token
	if token == "" {
		token = saved.Token
	}

	globals := &Globals{
		URL:   strings.TrimRight(url, "/"),
		Token: token,
		Mode:  mode,
	}

	// Validate auth for commands that need it.
	if requiresAuth(ctx.Command()) {
		if globals.URL == "" || globals.Token == "" {
			fmt.Fprintln(os.Stderr, "hass-cli is not configured yet.")
			fmt.Fprintln(os.Stderr, "Run `hass setup` to connect to your Home Assistant instance.")
			os.Exit(1)
		}
	}

	if err := ctx.Run(globals); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
