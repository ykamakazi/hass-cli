package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ankur/hass-cli/internal/outfmt"
)

// testServer creates a mock HA server and returns Globals pointing at it.
func testServer(t *testing.T, handler http.HandlerFunc) (*Globals, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Globals{URL: srv.URL, Token: "test-token", Mode: outfmt.Human}, srv
}

// captureStdout captures os.Stdout during fn and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestLsCmd_FiltersByDomain(t *testing.T) {
	states := []map[string]any{
		{"entity_id": "light.living_room", "state": "on", "attributes": map[string]any{}},
		{"entity_id": "switch.coffee", "state": "off", "attributes": map[string]any{}},
		{"entity_id": "light.bedroom", "state": "off", "attributes": map[string]any{}},
	}

	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(states)
	})

	cmd := &LsCmd{Domain: "light"}
	out := captureStdout(t, func() {
		if err := cmd.Run(globals); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	})

	if !contains(out, "light.living_room") {
		t.Errorf("output missing light.living_room:\n%s", out)
	}
	if !contains(out, "light.bedroom") {
		t.Errorf("output missing light.bedroom:\n%s", out)
	}
	if contains(out, "switch.coffee") {
		t.Errorf("output should not contain switch.coffee:\n%s", out)
	}
}

func TestGetCmd_JSON(t *testing.T) {
	state := map[string]any{
		"entity_id":    "sensor.temperature",
		"state":        "21.5",
		"attributes":   map[string]any{"unit_of_measurement": "°C"},
		"last_changed": "2026-01-01T00:00:00Z",
		"last_updated": "2026-01-01T00:00:00Z",
		"context":      map[string]any{"id": "abc", "parent_id": "", "user_id": ""},
	}

	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
	})
	globals.Mode = outfmt.JSON

	cmd := &GetCmd{EntityID: "sensor.temperature"}
	out := captureStdout(t, func() {
		if err := cmd.Run(globals); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	})

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if got["state"] != "21.5" {
		t.Errorf("state = %v, want 21.5", got["state"])
	}
}

func TestStatesSetCmd_ParsesAttributes(t *testing.T) {
	var gotBody map[string]any

	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"entity_id":    "sensor.test",
			"state":        "42",
			"attributes":   gotBody["attributes"],
			"last_changed": "",
			"last_updated": "",
			"context":      map[string]any{},
		})
	})

	cmd := &StatesSetCmd{
		EntityID:   "sensor.test",
		State:      "42",
		Attributes: []string{"unit_of_measurement=°C", "friendly_name=Test Sensor"},
	}
	if err := cmd.Run(globals); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	attrs, ok := gotBody["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("attributes not sent in body: %+v", gotBody)
	}
	if attrs["unit_of_measurement"] != "°C" {
		t.Errorf("unit_of_measurement = %v, want °C", attrs["unit_of_measurement"])
	}
	if attrs["friendly_name"] != "Test Sensor" {
		t.Errorf("friendly_name = %v, want 'Test Sensor'", attrs["friendly_name"])
	}
}

func TestStatesSetCmd_InvalidAttribute(t *testing.T) {
	globals := &Globals{URL: "http://localhost", Token: "tok", Mode: outfmt.Human}
	cmd := &StatesSetCmd{
		EntityID:   "sensor.test",
		State:      "42",
		Attributes: []string{"badformat"},
	}
	if err := cmd.Run(globals); err == nil {
		t.Error("Run() expected error for bad attribute format, got nil")
	}
}

func TestStatesDeleteCmd(t *testing.T) {
	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	globals.Mode = outfmt.Plain

	cmd := &StatesDeleteCmd{EntityID: "sensor.stale"}
	out := captureStdout(t, func() {
		if err := cmd.Run(globals); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	})

	if !contains(out, "sensor.stale") {
		t.Errorf("output missing deleted entity:\n%s", out)
	}
}

func TestOnCmd(t *testing.T) {
	var gotPath string
	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	cmd := &OnCmd{EntityID: "light.living_room"}
	if err := cmd.Run(globals); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotPath != "/api/services/homeassistant/turn_on" {
		t.Errorf("path = %q, want /api/services/homeassistant/turn_on", gotPath)
	}
}

func TestOffCmd(t *testing.T) {
	var gotPath string
	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	cmd := &OffCmd{EntityID: "light.living_room"}
	if err := cmd.Run(globals); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotPath != "/api/services/homeassistant/turn_off" {
		t.Errorf("path = %q, want /api/services/homeassistant/turn_off", gotPath)
	}
}

func TestToggleCmd(t *testing.T) {
	var gotPath string
	globals, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})

	cmd := &ToggleCmd{EntityID: "switch.fan"}
	if err := cmd.Run(globals); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotPath != "/api/services/homeassistant/toggle" {
		t.Errorf("path = %q, want /api/services/homeassistant/toggle", gotPath)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
