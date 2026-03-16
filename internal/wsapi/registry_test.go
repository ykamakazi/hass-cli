package wsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// fakeConn implements Conn for testing without a real WebSocket connection.
type fakeConn struct {
	receive []any
	pos     int
}

func (f *fakeConn) ReadJSON(v any) error {
	if f.pos >= len(f.receive) {
		// Return an error so readLoop exits cleanly, allowing Close() to return.
		return fmt.Errorf("no more messages")
	}
	b, err := json.Marshal(f.receive[f.pos])
	if err != nil {
		return err
	}
	f.pos++
	return json.Unmarshal(b, v)
}

func (f *fakeConn) WriteJSON(_ any) error { return nil }
func (f *fakeConn) Close() error          { return nil }

// newTestClient builds a Client with a fakeConn that returns the given responses.
func newTestClient(t *testing.T, responses []any) *Client {
	t.Helper()
	conn := &fakeConn{receive: responses}
	c := &Client{
		conn:    conn,
		nextID:  1,
		done:    make(chan struct{}),
		pending: make(map[int]chan *ResultMessage),
	}
	c.wg.Add(1)
	go c.readLoop()
	t.Cleanup(func() { c.Close() })
	return c
}

func successResult(id int, data any) map[string]any {
	b, _ := json.Marshal(data)
	return map[string]any{
		"id":      id,
		"type":    "result",
		"success": true,
		"result":  json.RawMessage(b),
	}
}

func errorResult(id int, code, msg string) map[string]any {
	return map[string]any{
		"id":      id,
		"type":    "result",
		"success": false,
		"error":   map[string]string{"code": code, "message": msg},
	}
}

// ── Area tests ────────────────────────────────────────────────────────────────

func TestListAreas(t *testing.T) {
	areas := []Area{
		{AreaID: "living_room", Name: "Living Room"},
		{AreaID: "kitchen", Name: "Kitchen"},
	}
	c := newTestClient(t, []any{successResult(1, areas)})

	got, err := c.ListAreas(context.Background())
	if err != nil {
		t.Fatalf("ListAreas() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListAreas() returned %d areas, want 2", len(got))
	}
	if got[0].AreaID != "living_room" {
		t.Errorf("areas[0].AreaID = %q, want living_room", got[0].AreaID)
	}
}

func TestCreateArea(t *testing.T) {
	want := Area{AreaID: "office", Name: "Office"}
	c := newTestClient(t, []any{successResult(1, want)})

	got, err := c.CreateArea(context.Background(), "Office")
	if err != nil {
		t.Fatalf("CreateArea() error = %v", err)
	}
	if got.AreaID != "office" {
		t.Errorf("AreaID = %q, want office", got.AreaID)
	}
}

func TestDeleteArea(t *testing.T) {
	c := newTestClient(t, []any{successResult(1, nil)})

	if err := c.DeleteArea(context.Background(), "office"); err != nil {
		t.Fatalf("DeleteArea() error = %v", err)
	}
}

func TestResolveArea_ByName(t *testing.T) {
	areas := []Area{
		{AreaID: "living_room", Name: "Living Room"},
		{AreaID: "kitchen", Name: "Kitchen"},
	}
	c := newTestClient(t, []any{successResult(1, areas)})

	id, err := c.ResolveArea(context.Background(), "living room")
	if err != nil {
		t.Fatalf("ResolveArea() error = %v", err)
	}
	if id != "living_room" {
		t.Errorf("ResolveArea() = %q, want living_room", id)
	}
}

func TestResolveArea_ByID(t *testing.T) {
	areas := []Area{{AreaID: "kitchen", Name: "Kitchen"}}
	c := newTestClient(t, []any{successResult(1, areas)})

	id, err := c.ResolveArea(context.Background(), "kitchen")
	if err != nil {
		t.Fatalf("ResolveArea() by ID error = %v", err)
	}
	if id != "kitchen" {
		t.Errorf("ResolveArea() = %q, want kitchen", id)
	}
}

func TestResolveArea_NotFound(t *testing.T) {
	c := newTestClient(t, []any{successResult(1, []Area{})})

	_, err := c.ResolveArea(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("ResolveArea() expected error for unknown area, got nil")
	}
}

func TestResolveArea_CaseInsensitive(t *testing.T) {
	areas := []Area{{AreaID: "living_room", Name: "Living Room"}}
	c := newTestClient(t, []any{successResult(1, areas)})

	id, err := c.ResolveArea(context.Background(), "LIVING ROOM")
	if err != nil {
		t.Fatalf("ResolveArea() error = %v", err)
	}
	if id != "living_room" {
		t.Errorf("ResolveArea() = %q, want living_room", id)
	}
}

// ── Entity registry tests ─────────────────────────────────────────────────────

func TestGetEntity(t *testing.T) {
	want := EntityRegistryEntry{
		EntityID: "switch.coffee_maker",
		Platform: "zigbee2mqtt",
		AreaID:   "kitchen",
	}
	c := newTestClient(t, []any{successResult(1, want)})

	got, err := c.GetEntity(context.Background(), "switch.coffee_maker")
	if err != nil {
		t.Fatalf("GetEntity() error = %v", err)
	}
	if got.EntityID != "switch.coffee_maker" {
		t.Errorf("EntityID = %q, want switch.coffee_maker", got.EntityID)
	}
	if got.AreaID != "kitchen" {
		t.Errorf("AreaID = %q, want kitchen", got.AreaID)
	}
}

func TestSetEntityArea(t *testing.T) {
	// HA wraps entity results in entity_entry.
	want := map[string]any{
		"entity_entry": map[string]any{
			"entity_id": "light.bedroom",
			"area_id":   "bedroom",
		},
	}
	c := newTestClient(t, []any{successResult(1, want)})

	entry, err := c.SetEntityArea(context.Background(), "light.bedroom", "bedroom")
	if err != nil {
		t.Fatalf("SetEntityArea() error = %v", err)
	}
	if entry.EntityID != "light.bedroom" {
		t.Errorf("EntityID = %q, want light.bedroom", entry.EntityID)
	}
	if entry.AreaID != "bedroom" {
		t.Errorf("AreaID = %q, want bedroom", entry.AreaID)
	}
}

func TestRenameEntity(t *testing.T) {
	want := map[string]any{
		"entity_entry": map[string]any{
			"entity_id": "switch.coffee_maker",
			"name":      "Coffee Machine",
		},
	}
	c := newTestClient(t, []any{successResult(1, want)})

	entry, err := c.RenameEntity(context.Background(), "switch.coffee_maker", "Coffee Machine")
	if err != nil {
		t.Fatalf("RenameEntity() error = %v", err)
	}
	if entry.Name != "Coffee Machine" {
		t.Errorf("Name = %q, want Coffee Machine", entry.Name)
	}
}

// ── Error handling ────────────────────────────────────────────────────────────

func TestCommand_APIError(t *testing.T) {
	c := newTestClient(t, []any{errorResult(1, "not_found", "entity not found")})

	_, err := c.GetEntity(context.Background(), "sensor.nonexistent")
	if err == nil {
		t.Fatal("GetEntity() expected error for API error result, got nil")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

// ── Device registry tests ─────────────────────────────────────────────────────

func TestListDevices(t *testing.T) {
	devices := []Device{
		{ID: "abc123", Name: "Zigbee Button", AreaID: "living_room"},
		{ID: "def456", Name: "Motion Sensor", AreaID: "kitchen"},
	}
	c := newTestClient(t, []any{successResult(1, devices)})

	got, err := c.ListDevices(context.Background())
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListDevices() returned %d devices, want 2", len(got))
	}
	if got[0].ID != "abc123" {
		t.Errorf("devices[0].ID = %q, want abc123", got[0].ID)
	}
}

func TestSetDeviceArea(t *testing.T) {
	want := Device{ID: "abc123", Name: "Zigbee Button", AreaID: "bedroom"}
	c := newTestClient(t, []any{successResult(1, want)})

	got, err := c.SetDeviceArea(context.Background(), "abc123", "bedroom")
	if err != nil {
		t.Fatalf("SetDeviceArea() error = %v", err)
	}
	if got.AreaID != "bedroom" {
		t.Errorf("AreaID = %q, want bedroom", got.AreaID)
	}
}
