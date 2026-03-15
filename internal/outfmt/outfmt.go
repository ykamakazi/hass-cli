package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
)

// Mode controls the output format.
type Mode int

const (
	Human Mode = iota
	JSON
	Plain
)

// Output writes data in the specified format.
func Output(mode Mode, data any, w io.Writer) {
	switch mode {
	case JSON:
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Fprintf(w, `{"error": %q}`+"\n", err.Error())
			return
		}
		fmt.Fprintf(w, "%s\n", b)
	case Plain:
		// For plain mode, data should be [][2]string or similar; fall through to human
		fmt.Fprintf(w, "%v\n", data)
	case Human:
		fmt.Fprintf(w, "%v\n", data)
	}
}

// OutputPlain writes key-value pairs as tab-separated lines.
func OutputPlain(fields [][2]string, w io.Writer) {
	for _, f := range fields {
		fmt.Fprintf(w, "%s\t%s\n", f[0], f[1])
	}
}

// OutputJSON writes data as indented JSON.
func OutputJSON(data any, w io.Writer) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(w, `{"error": %q}`+"\n", err.Error())
		return
	}
	fmt.Fprintf(w, "%s\n", b)
}
