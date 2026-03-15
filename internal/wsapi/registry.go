package wsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Area represents a Home Assistant area.
type Area struct {
	AreaID  string   `json:"area_id"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Icon    string   `json:"icon"`
}

// EntityRegistryEntry is an entity's registry record.
type EntityRegistryEntry struct {
	EntityID string `json:"entity_id"`
	Name     string `json:"name"`
	AreaID   string `json:"area_id"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
	Icon     string `json:"icon"`
}

// Device is a device registry record.
type Device struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AreaID       string `json:"area_id"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
}

func (c *Client) command(ctx context.Context, msgType string, extra map[string]any) (json.RawMessage, error) {
	id := c.nextMessageID()
	msg := map[string]any{"id": id, "type": msgType}
	for k, v := range extra {
		msg[k] = v
	}
	result, err := c.call(ctx, msg, id)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

// ── Area Registry ─────────────────────────────────────────────────────────────

func (c *Client) ListAreas(ctx context.Context) ([]Area, error) {
	raw, err := c.command(ctx, "config/area_registry/list", nil)
	if err != nil {
		return nil, err
	}
	var areas []Area
	return areas, json.Unmarshal(raw, &areas)
}

func (c *Client) CreateArea(ctx context.Context, name string) (*Area, error) {
	raw, err := c.command(ctx, "config/area_registry/create", map[string]any{"name": name})
	if err != nil {
		return nil, err
	}
	var area Area
	return &area, json.Unmarshal(raw, &area)
}

func (c *Client) UpdateArea(ctx context.Context, areaID, name string) (*Area, error) {
	raw, err := c.command(ctx, "config/area_registry/update", map[string]any{"area_id": areaID, "name": name})
	if err != nil {
		return nil, err
	}
	var area Area
	return &area, json.Unmarshal(raw, &area)
}

func (c *Client) DeleteArea(ctx context.Context, areaID string) error {
	_, err := c.command(ctx, "config/area_registry/delete", map[string]any{"area_id": areaID})
	return err
}

// ResolveArea maps a human-readable name or area_id to an area_id.
func (c *Client) ResolveArea(ctx context.Context, nameOrID string) (string, error) {
	areas, err := c.ListAreas(ctx)
	if err != nil {
		return "", err
	}
	lower := strings.ToLower(nameOrID)
	for _, a := range areas {
		if a.AreaID == nameOrID || strings.ToLower(a.Name) == lower {
			return a.AreaID, nil
		}
	}
	return "", fmt.Errorf("area %q not found", nameOrID)
}

// ── Entity Registry ───────────────────────────────────────────────────────────

func (c *Client) GetEntity(ctx context.Context, entityID string) (*EntityRegistryEntry, error) {
	raw, err := c.command(ctx, "config/entity_registry/get", map[string]any{"entity_id": entityID})
	if err != nil {
		return nil, err
	}
	var entry EntityRegistryEntry
	return &entry, json.Unmarshal(raw, &entry)
}

func (c *Client) SetEntityArea(ctx context.Context, entityID, areaID string) (*EntityRegistryEntry, error) {
	raw, err := c.command(ctx, "config/entity_registry/update", map[string]any{
		"entity_id": entityID,
		"area_id":   areaID,
	})
	if err != nil {
		return nil, err
	}
	return unmarshalEntityResult(raw)
}

func (c *Client) RenameEntity(ctx context.Context, entityID, name string) (*EntityRegistryEntry, error) {
	raw, err := c.command(ctx, "config/entity_registry/update", map[string]any{
		"entity_id": entityID,
		"name":      name,
	})
	if err != nil {
		return nil, err
	}
	return unmarshalEntityResult(raw)
}

func unmarshalEntityResult(raw json.RawMessage) (*EntityRegistryEntry, error) {
	var wrapper struct {
		Entry EntityRegistryEntry `json:"entity_entry"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Entry.EntityID != "" {
		return &wrapper.Entry, nil
	}
	var entry EntityRegistryEntry
	return &entry, json.Unmarshal(raw, &entry)
}

// ── Device Registry ───────────────────────────────────────────────────────────

func (c *Client) ListDevices(ctx context.Context) ([]Device, error) {
	raw, err := c.command(ctx, "config/device_registry/list", nil)
	if err != nil {
		return nil, err
	}
	var devices []Device
	return devices, json.Unmarshal(raw, &devices)
}

func (c *Client) SetDeviceArea(ctx context.Context, deviceID, areaID string) (*Device, error) {
	raw, err := c.command(ctx, "config/device_registry/update", map[string]any{
		"device_id": deviceID,
		"area_id":   areaID,
	})
	if err != nil {
		return nil, err
	}
	var device Device
	return &device, json.Unmarshal(raw, &device)
}
