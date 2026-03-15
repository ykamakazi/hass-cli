package cmd

import (
	"fmt"
	"os"

	"github.com/ankur/hass-cli/internal/outfmt"
)

// Version is set at build time.
var Version = "dev"

// VersionCmd shows version information.
type VersionCmd struct{}

func (v *VersionCmd) Run(globals *Globals) error {
	switch globals.Mode {
	case outfmt.JSON:
		outfmt.OutputJSON(map[string]string{"version": Version}, os.Stdout)
	case outfmt.Plain:
		outfmt.OutputPlain([][2]string{{"version", Version}}, os.Stdout)
	default:
		fmt.Fprintf(os.Stdout, "hass-cli version %s\n", Version)
	}
	return nil
}
