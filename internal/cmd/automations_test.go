package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankur/hass-cli/internal/outfmt"
)

func TestAutomationsCreate(t *testing.T) {
	for _, source := range []string{"data", "file", "stdin"} {
		t.Run(source, func(t *testing.T) {
			calls := 0
			globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/api/config/automation/config/new_alert" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing auth")
				}
				if calls == 1 {
					if r.Method != "GET" {
						t.Errorf("preflight method: %s", r.Method)
					}
					http.Error(w, "not found", 404)
					return
				}
				if r.Method != "POST" {
					t.Errorf("save method: %s", r.Method)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["id"] != "new_alert" || body["alias"] != "New alert" || body["actions"] == nil {
					t.Errorf("unexpected config: %v", body)
				}
				w.Write([]byte(`{"result":"ok"}`))
			})
			globals.Mode = outfmt.JSON
			data := `{"alias":"New alert","triggers":[],"actions":[]}`
			cmd := &AutomationsCreateCmd{ID: "new_alert"}
			if source == "data" {
				cmd.Data = data
			} else {
				path := filepath.Join(t.TempDir(), "config.json")
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
				cmd.File = path
				if source == "stdin" {
					f, err := os.Open(path)
					if err != nil {
						t.Fatal(err)
					}
					defer f.Close()
					old := os.Stdin
					os.Stdin = f
					defer func() { os.Stdin = old }()
					cmd.File = "-"
				}
			}
			var runErr error
			output := captureStdout(t, func() { runErr = cmd.Run(globals) })
			if runErr != nil {
				t.Fatal(runErr)
			}
			if calls != 2 {
				t.Errorf("requests: %d", calls)
			}
			var result map[string]string
			if err := json.Unmarshal([]byte(output), &result); err != nil || result["created"] != "new_alert" {
				t.Errorf("output: %s", output)
			}
		})
	}
}

func TestAutomationsCreatePreflight(t *testing.T) {
	for _, status := range []int{200, 401, 403, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Error("must not write when preflight fails")
				}
				w.WriteHeader(status)
				w.Write([]byte(`{}`))
			})
			cmd := &AutomationsCreateCmd{ID: "existing", Data: `{"alias":"test"}`}
			if err := cmd.Run(globals); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestAutomationsCreateInvalidInput(t *testing.T) {
	for _, cmd := range []AutomationsCreateCmd{
		{ID: "new"}, {ID: "new", Data: `{}`, File: "x"}, {ID: "new", Data: `null`},
		{ID: "new", Data: `[]`}, {ID: "new", Data: `{`}, {ID: "new", Data: `{"id":"other"}`},
		{ID: "automation.new", Data: `{}`}, {ID: "../new", Data: `{}`}, {ID: "new", File: "/nonexistent/config.json"},
	} {
		globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid input must not access server") })
		if err := cmd.Run(globals); err == nil {
			t.Errorf("expected error for %+v", cmd)
		}
	}
}

func TestAutomationsCreateSaveFailure(t *testing.T) {
	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			http.Error(w, "missing", 404)
		} else {
			http.Error(w, "invalid automation", 400)
		}
	})
	cmd := &AutomationsCreateCmd{ID: "new", Data: `{"alias":"test"}`}
	var err error
	output := captureStdout(t, func() { err = cmd.Run(globals) })
	if err == nil || !strings.Contains(err.Error(), "create automation") || output != "" {
		t.Errorf("err=%v output=%q", err, output)
	}
}
