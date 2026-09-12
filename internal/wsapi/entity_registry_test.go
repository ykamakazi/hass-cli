package wsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// recordingConn captures every message written so tests can assert on the
// exact payload sent to Home Assistant.
type recordingConn struct {
	fakeConn
	mu   sync.Mutex
	sent []map[string]any
}

func (r *recordingConn) WriteJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	r.mu.Lock()
	r.sent = append(r.sent, m)
	r.mu.Unlock()
	return nil
}

func (r *recordingConn) lastSent(t *testing.T) map[string]any {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sent) == 0 {
		t.Fatal("no messages were sent")
	}
	return r.sent[len(r.sent)-1]
}

func newRecordingClient(t *testing.T, responses []any) (*Client, *recordingConn) {
	t.Helper()
	conn := &recordingConn{fakeConn: fakeConn{receive: responses}}
	c := &Client{
		conn:    conn,
		nextID:  1,
		done:    make(chan struct{}),
		pending: make(map[int]chan *ResultMessage),
	}
	c.wg.Add(1)
	go c.readLoop()
	t.Cleanup(func() { c.Close() })
	return c, conn
}

func TestDeleteEntity(t *testing.T) {
	c, conn := newRecordingClient(t, []any{successResult(1, nil)})

	if err := c.DeleteEntity(context.Background(), "media_player.living_room"); err != nil {
		t.Fatalf("DeleteEntity() error = %v", err)
	}

	sent := conn.lastSent(t)
	if sent["type"] != "config/entity_registry/remove" {
		t.Errorf("type = %v, want config/entity_registry/remove", sent["type"])
	}
	if sent["entity_id"] != "media_player.living_room" {
		t.Errorf("entity_id = %v, want media_player.living_room", sent["entity_id"])
	}
}

func TestDeleteEntity_NotFound(t *testing.T) {
	c, _ := newRecordingClient(t, []any{errorResult(1, "not_found", "Entity not found")})

	err := c.DeleteEntity(context.Background(), "light.nope")
	if err == nil {
		t.Fatal("DeleteEntity() error = nil, want an error")
	}
}

func TestSetEntityDisabled_Disable(t *testing.T) {
	entry := EntityRegistryEntry{EntityID: "media_player.living_room", DisabledBy: "user"}
	c, conn := newRecordingClient(t, []any{
		successResult(1, map[string]any{"entity_entry": entry}),
	})

	res, err := c.SetEntityDisabled(context.Background(), "media_player.living_room", true)
	if err != nil {
		t.Fatalf("SetEntityDisabled() error = %v", err)
	}
	if res.Entry.DisabledBy != "user" {
		t.Errorf("DisabledBy = %q, want user", res.Entry.DisabledBy)
	}

	sent := conn.lastSent(t)
	if sent["type"] != "config/entity_registry/update" {
		t.Errorf("type = %v, want config/entity_registry/update", sent["type"])
	}
	if sent["disabled_by"] != "user" {
		t.Errorf("disabled_by = %v, want user", sent["disabled_by"])
	}
}

// Re-enabling requires disabled_by to be JSON null. An empty string would be
// stored verbatim and leave the entity disabled.
func TestSetEntityDisabled_EnableSendsNull(t *testing.T) {
	entry := EntityRegistryEntry{EntityID: "media_player.living_room"}
	c, conn := newRecordingClient(t, []any{
		successResult(1, map[string]any{"entity_entry": entry, "reload_delay": 30}),
	})

	res, err := c.SetEntityDisabled(context.Background(), "media_player.living_room", false)
	if err != nil {
		t.Fatalf("SetEntityDisabled() error = %v", err)
	}
	if res.ReloadDelay != 30 {
		t.Errorf("ReloadDelay = %d, want 30", res.ReloadDelay)
	}
	if res.Entry.DisabledBy != "" {
		t.Errorf("DisabledBy = %q, want empty", res.Entry.DisabledBy)
	}

	sent := conn.lastSent(t)
	v, ok := sent["disabled_by"]
	if !ok {
		t.Fatal("disabled_by key missing from update message")
	}
	if v != nil {
		t.Errorf("disabled_by = %#v, want nil (JSON null)", v)
	}
}

func TestSetEntityDisabled_RequireRestart(t *testing.T) {
	entry := EntityRegistryEntry{EntityID: "sensor.x"}
	c, _ := newRecordingClient(t, []any{
		successResult(1, map[string]any{"entity_entry": entry, "require_restart": true}),
	})

	res, err := c.SetEntityDisabled(context.Background(), "sensor.x", false)
	if err != nil {
		t.Fatalf("SetEntityDisabled() error = %v", err)
	}
	if !res.RequireRestart {
		t.Error("RequireRestart = false, want true")
	}
}

// blockingConn models a real socket: ReadJSON parks until the connection is
// closed, which is what made Close() deadlock.
type blockingConn struct {
	closed chan struct{}
	once   sync.Once
}

func (b *blockingConn) ReadJSON(any) error {
	<-b.closed
	return fmt.Errorf("use of closed connection")
}
func (b *blockingConn) WriteJSON(any) error { return nil }
func (b *blockingConn) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

// Regression: Close() used to wait for readLoop before closing the connection,
// but readLoop only returns once the connection is closed. Every WebSocket
// command printed its result and then hung forever on exit.
func TestClose_DoesNotDeadlockOnBlockingRead(t *testing.T) {
	c := &Client{
		conn:    &blockingConn{closed: make(chan struct{})},
		nextID:  1,
		done:    make(chan struct{}),
		pending: make(map[int]chan *ResultMessage),
	}
	c.wg.Add(1)
	go c.readLoop()

	// Let readLoop reach the blocking read.
	time.Sleep(20 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		c.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() deadlocked against a blocking ReadJSON")
	}
}
