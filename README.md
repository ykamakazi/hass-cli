# hass-cli

A command-line interface for [Home Assistant](https://www.home-assistant.io/) designed to be used by humans and AI agents alike. Query and control your smart home from your terminal, shell scripts, or an AI assistant's tool calls.

Inspired by [gogcli](https://github.com/steipete/gogcli) — the same philosophy applied to Home Assistant: clean JSON output, stable exit codes, and a structure that makes it easy for an LLM to discover and invoke capabilities without guesswork.

## Why

Home Assistant has a powerful REST API but no first-class CLI. This fills that gap.

The primary motivation is **agent use**: with `--json` output, stable exit codes, and a `hass agent schema` discovery command, any AI that can run shell commands can become a smart home controller. Ask Claude to turn off the lights when you leave, check if the garage door is still open, or summarize what happened in your home while you were away.

## Installation

### Homebrew (recommended)

```bash
brew tap ykamakazi/tap
brew install hass-cli
```

### Build from source

Requires Go 1.21+.

```bash
git clone https://github.com/ykamakazi/hass-cli
cd hass-cli
make
mv hass /usr/local/bin/hass
```

## Authentication

Run the setup wizard — it auto-discovers Home Assistant on your network, opens the token creation page in your browser, and saves everything for you:

```bash
hass setup
```

Config is saved to `~/Library/Application Support/hass-cli/.env` (macOS) or `~/.config/hass-cli/.env` (Linux) and loaded automatically on every command.

To update your settings later, just run `hass setup` again.

## Usage

### Quick commands

```bash
# List all entity states (optionally filter by domain)
hass ls
hass ls --domain light
hass ls --domain sensor

# Get a specific entity
hass get sensor.outdoor_temperature
hass show light.living_room   # alias for get

# Control devices
hass on light.living_room
hass off light.living_room
hass toggle switch.coffee_maker
```

### States

```bash
hass states list --domain climate
hass states get climate.living_room
hass states set sensor.my_sensor 42 --attr unit_of_measurement=°C
hass states delete sensor.stale_entity
```

### Services

```bash
# List all available services
hass services list
hass services list --domain light

# Call a service
hass services call light turn_on --entity light.bedroom --data '{"brightness": 128}'
hass services call climate set_temperature --entity climate.office --data '{"temperature": 21}'
hass services call cover open_cover --entity cover.garage_door
```

### History & Logbook

```bash
# Last hour of state changes for an entity
hass history --entity sensor.outdoor_temperature

# Custom time range
hass history --entity light.living_room --start 2024-01-01T00:00:00Z --end 2024-01-02T00:00:00Z

# Logbook (last 24h by default)
hass logbook
hass logbook --entity lock.front_door
```

### Config & Diagnostics

```bash
hass config get
hass config check
hass config components
hass config error-log
```

### Templates

```bash
# Render any Home Assistant template expression
hass template render '{{ states("sensor.outdoor_temperature") }}°C'
hass template render '{{ states | selectattr("domain", "eq", "light") | selectattr("state", "eq", "on") | list | count }} lights on'

# Pipe a template from a file
cat my_template.j2 | hass template render -
```

### Calendars

```bash
hass calendars list
hass calendars events calendar.family --start 2024-01-01 --end 2024-01-31
```

### Output formats

Every command supports three output modes:

```bash
hass ls --domain light           # human-readable (default)
hass ls --domain light --json    # JSON (for scripts and agents)
hass ls --domain light --plain   # TSV (for grep/awk pipelines)
```

---

## Using with AI Agents

### Claude Code

Add these lines to your project's `CLAUDE.md` or to `~/.claude/CLAUDE.md` so Claude always knows how to control your home:

```markdown
## Smart Home
The user has Home Assistant running locally. Use `hass` CLI to interact with it.
Config is stored in ~/Library/Application Support/hass-cli/.env and loaded automatically.

Key commands:
- `hass ls --domain light --json` — list all lights and their states
- `hass on <entity_id>` / `hass off <entity_id>` — control devices
- `hass services call <domain> <service> --entity <id> --data '<json>'` — call any HA service
- `hass history --entity <id> --json` — check recent state changes
- `hass template render '<template>'` — query anything expressible in Jinja2
- `hass agent schema --json` — discover available commands

Always use --json flag when you need to parse the output.
```

Now you can just ask Claude:

> "Is the garage door open?"
> "Turn off all the lights in the living room"
> "What's the current temperature in each room?"
> "Did anyone open the front door in the last hour?"
> "Set the thermostat to 21°C"
> "Show me a summary of what happened at home today"

### OpenClaw / Other agents

Any agent that can execute shell commands works the same way. Point it at the binary and give it the schema:

```bash
# Agent discovers capabilities
hass agent schema --json

# Agent checks exit codes
hass agent exit-codes --json
```

The CLI is designed to be unambiguous for LLMs:
- **`--json` always produces valid JSON on stdout** — no prose, no color codes
- **Errors always go to stderr** — stdout is clean for parsing
- **Exit code 0** = success, **1** = error, **2** = usage error
- **Entity IDs are explicit** — `light.living_room`, `switch.coffee_maker` — no ambiguity

### Example agent workflow

```bash
# 1. Discover what lights exist
hass ls --domain light --json | jq '[.[] | {id: .entity_id, state: .state, name: .attributes.friendly_name}]'

# 2. Check if a specific device is on
hass get switch.coffee_maker --json | jq '.state'

# 3. Turn something on based on a condition
if [ "$(hass get binary_sensor.someone_home --json | jq -r '.state')" = "on" ]; then
    hass on light.welcome_light
fi

# 4. Use a template for complex queries
hass template render '{{ states.sensor | selectattr("attributes.device_class", "eq", "temperature") | map(attribute="state") | list }}' --json
```

---

## All commands

```
hass <command> [flags]

Flags:
  -u, --url=STRING      Override Home Assistant URL (saved by `hass setup`)
  -t, --token=STRING    Override access token (saved by `hass setup`)
      --json            Output as JSON
      --plain           Output as plain text / TSV

Commands:
  on <entity>                        Turn on an entity
  off <entity>                       Turn off an entity
  toggle <entity>                    Toggle an entity
  get <entity>                       Get state of an entity (alias: show)
  ls [--domain <domain>]             List entity states (alias: list)

  states list [--domain]             List all entity states
  states get <entity>                Get a single entity state
  states set <entity> <state>        Create or update entity state
  states delete <entity>             Delete an entity

  services list [--domain]           List available services
  services call <domain> <service>   Call a service

  events list                        List all events
  events fire <type> [--data]        Fire an event

  history [--entity] [--start] [--end] [--significant-only]
  logbook [--entity] [--start]

  config get                         Show HA configuration
  config check                       Validate configuration
  config components                  List loaded components
  config error-log                   Show error log

  calendars list                     List calendar entities
  calendars events <id> --start --end

  template render <template>         Render a Jinja2 template

  agent exit-codes                   List exit codes (no auth required)
  agent schema                       Show CLI schema (no auth required)
  version                            Show version (no auth required)
```

## License

MIT
