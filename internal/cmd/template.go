package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ankur/hass-cli/internal/hassapi"
	"github.com/ankur/hass-cli/internal/outfmt"
)

// TemplateCmd groups template subcommands.
type TemplateCmd struct {
	Render TemplateRenderCmd `cmd:"" help:"Render a Home Assistant template."`
}

// TemplateRenderCmd renders a template string.
type TemplateRenderCmd struct {
	Template string `arg:"" help:"Template string to render. Use '-' to read from stdin."`
}

func (c *TemplateRenderCmd) Run(globals *Globals) error {
	tmpl := c.Template

	if tmpl == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read template from stdin: %w", err)
		}
		tmpl = string(data)
	}

	client := hassapi.NewClient(globals.URL, globals.Token)
	result, err := client.RenderTemplate(context.Background(), tmpl)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"result": result}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"result", result}}, os.Stdout)
	default:
		fmt.Fprint(os.Stdout, result)
		if len(result) > 0 && result[len(result)-1] != '\n' {
			fmt.Fprintln(os.Stdout)
		}
	}
	return nil
}
