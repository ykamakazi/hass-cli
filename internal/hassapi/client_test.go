package hassapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient returns a Client pointed at a mock HTTP server.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "test-token"), srv
}

func TestGetStates(t *testing.T) {
	want := []State{
		{EntityID: "light.living_room", State: "on"},
		{EntityID: "switch.coffee_maker", State: "off"},
	}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	})

	states, err := client.GetStates()
	if err != nil {
		t.Fatalf("GetStates() error = %v", err)
	}
	if len(states) != len(want) {
		t.Fatalf("GetStates() returned %d states, want %d", len(states), len(want))
	}
	for i, s := range states {
		if s.EntityID != want[i].EntityID {
			t.Errorf("states[%d].EntityID = %q, want %q", i, s.EntityID, want[i].EntityID)
		}
		if s.State != want[i].State {
			t.Errorf("states[%d].State = %q, want %q", i, s.State, want[i].State)
		}
	}
}

func TestGetState(t *testing.T) {
	want := State{
		EntityID:   "light.bedroom",
		State:      "on",
		Attributes: map[string]any{"brightness": float64(255)},
	}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states/light.bedroom" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	})

	state, err := client.GetState("light.bedroom")
	if err != nil {
		t.Fatalf("GetState() error = %v", err)
	}
	if state.EntityID != want.EntityID {
		t.Errorf("EntityID = %q, want %q", state.EntityID, want.EntityID)
	}
	if state.State != want.State {
		t.Errorf("State = %q, want %q", state.State, want.State)
	}
}

func TestGetState_NotFound(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Entity not found"}`, http.StatusNotFound)
	})

	_, err := client.GetState("light.nonexistent")
	if err == nil {
		t.Fatal("GetState() expected error for 404, got nil")
	}
}

func TestCallService(t *testing.T) {
	called := false
	returned := []State{{EntityID: "light.bedroom", State: "on"}}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/services/light/turn_on" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(returned)
	})

	states, err := client.CallService("light", "turn_on", map[string]any{"entity_id": "light.bedroom"})
	if err != nil {
		t.Fatalf("CallService() error = %v", err)
	}
	if !called {
		t.Error("CallService() did not hit the server")
	}
	if len(states) != 1 || states[0].EntityID != "light.bedroom" {
		t.Errorf("CallService() returned unexpected states: %+v", states)
	}
}

func TestSetState(t *testing.T) {
	want := State{EntityID: "sensor.test", State: "42"}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/api/states/sensor.test" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["state"] != "42" {
			t.Errorf("body state = %v, want 42", body["state"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	})

	state, err := client.SetState("sensor.test", "42", nil)
	if err != nil {
		t.Fatalf("SetState() error = %v", err)
	}
	if state.State != "42" {
		t.Errorf("State = %q, want %q", state.State, "42")
	}
}

func TestDeleteState(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %q", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteState("sensor.stale"); err != nil {
		t.Fatalf("DeleteState() error = %v", err)
	}
}

func TestRenderTemplate(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/template" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["template"] == "" {
			t.Error("template field missing from request body")
		}
		w.Write([]byte("on"))
	})

	result, err := client.RenderTemplate(`{{ states("light.living_room") }}`)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}
	if result != "on" {
		t.Errorf("RenderTemplate() = %q, want %q", result, "on")
	}
}

func TestGetEvents(t *testing.T) {
	want := []Event{
		{Event: "state_changed", ListenerCount: 5},
		{Event: "call_service", ListenerCount: 2},
	}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/events" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	})

	events, err := client.GetEvents()
	if err != nil {
		t.Fatalf("GetEvents() error = %v", err)
	}
	if len(events) != len(want) {
		t.Fatalf("GetEvents() returned %d events, want %d", len(events), len(want))
	}
}

func TestGetAPIStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"ok", http.StatusOK, false},
		{"unauthorized", http.StatusUnauthorized, true},
		{"not found", http.StatusNotFound, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				if tc.statusCode == http.StatusOK {
					w.Write([]byte(`{"message":"API running."}`))
				}
			})

			err := client.GetAPIStatus()
			if (err != nil) != tc.wantErr {
				t.Errorf("GetAPIStatus() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
