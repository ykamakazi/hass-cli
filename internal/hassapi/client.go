package hassapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// State represents a Home Assistant entity state.
type State struct {
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes"`
	LastChanged string        `json:"last_changed"`
	LastUpdated string        `json:"last_updated"`
	Context    struct {
		ID       string `json:"id"`
		ParentID string `json:"parent_id"`
		UserID   string `json:"user_id"`
	} `json:"context"`
}

// ServiceDomain represents a domain and its services.
type ServiceDomain struct {
	Domain   string             `json:"domain"`
	Services map[string]Service `json:"services"`
}

// Service represents a single HA service.
type Service struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Event represents a Home Assistant event.
type Event struct {
	Event         string `json:"event"`
	ListenerCount int    `json:"listener_count"`
}

// LogEntry represents a logbook entry.
type LogEntry struct {
	When          string `json:"when"`
	Name          string `json:"name"`
	Message       string `json:"message"`
	Domain        string `json:"domain,omitempty"`
	EntityID      string `json:"entity_id,omitempty"`
	ContextUserID string `json:"context_user_id,omitempty"`
}

// Calendar represents a calendar entity.
type Calendar struct {
	EntityID string `json:"entity_id"`
	Name     string `json:"name"`
}

// CalendarEvent represents an event on a calendar.
type CalendarEvent struct {
	Summary     string            `json:"summary"`
	Start       map[string]string `json:"start"`
	End         map[string]string `json:"end"`
	Description string            `json:"description,omitempty"`
	Location    string            `json:"location,omitempty"`
}

// ConfigResponse holds the HA configuration.
type ConfigResponse map[string]any

// TemplateResponse holds the rendered template result.
type TemplateResponse struct {
	Template string `json:"template"`
}

// ConfigCheck holds the result of a config check.
type ConfigCheck struct {
	Result string `json:"result"`
	Errors string `json:"errors,omitempty"`
}

// ComponentList is a list of loaded components.
type ComponentList []string

// Client is an HTTP client for the Home Assistant REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new HA API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) buildURL(path string) string {
	return c.baseURL + "/api" + path
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	return resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
		}
	}

	return nil
}

func (c *Client) doText(ctx context.Context, method, path string, body any) (string, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

func (c *Client) doDelete(ctx context.Context, path string) error {
	req, err := c.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetAPIStatus checks if the API is running.
func (c *Client) GetAPIStatus(ctx context.Context) error {
	var result map[string]any
	return c.doJSON(ctx, "GET", "/", nil, &result)
}

// GetStates returns all entity states.
func (c *Client) GetStates(ctx context.Context) ([]State, error) {
	var states []State
	if err := c.doJSON(ctx, "GET", "/states", nil, &states); err != nil {
		return nil, err
	}
	return states, nil
}

// GetState returns the state of a specific entity.
func (c *Client) GetState(ctx context.Context, entityID string) (*State, error) {
	var state State
	if err := c.doJSON(ctx, "GET", "/states/"+url.PathEscape(entityID), nil, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SetState creates or updates the state of an entity.
func (c *Client) SetState(ctx context.Context, entityID, state string, attributes map[string]any) (*State, error) {
	payload := map[string]any{
		"state":      state,
		"attributes": attributes,
	}
	var result State
	if err := c.doJSON(ctx, "POST", "/states/"+url.PathEscape(entityID), payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteState deletes an entity's state.
func (c *Client) DeleteState(ctx context.Context, entityID string) error {
	return c.doDelete(ctx, "/states/"+url.PathEscape(entityID))
}

// GetServices returns all service domains.
func (c *Client) GetServices(ctx context.Context) ([]ServiceDomain, error) {
	var domains []ServiceDomain
	if err := c.doJSON(ctx, "GET", "/services", nil, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

// CallService calls a Home Assistant service.
func (c *Client) CallService(ctx context.Context, domain, service string, data map[string]any) ([]State, error) {
	var states []State
	if err := c.doJSON(ctx, "POST", "/services/"+url.PathEscape(domain)+"/"+url.PathEscape(service), data, &states); err != nil {
		return nil, err
	}
	return states, nil
}

// GetEvents returns all registered events.
func (c *Client) GetEvents(ctx context.Context) ([]Event, error) {
	var events []Event
	if err := c.doJSON(ctx, "GET", "/events", nil, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// FireEvent fires a Home Assistant event.
func (c *Client) FireEvent(ctx context.Context, eventType string, data map[string]any) error {
	var result map[string]any
	return c.doJSON(ctx, "POST", "/events/"+url.PathEscape(eventType), data, &result)
}

// GetHistory returns the history of entity states.
func (c *Client) GetHistory(ctx context.Context, entityID string, startTime time.Time, endTime *time.Time, significantChangesOnly bool) ([][]State, error) {
	path := "/history/period/" + url.PathEscape(startTime.Format(time.RFC3339))

	params := url.Values{}
	if entityID != "" {
		params.Set("filter_entity_id", entityID)
	}
	if endTime != nil {
		params.Set("end_time", endTime.Format(time.RFC3339))
	}
	if significantChangesOnly {
		params.Set("significant_changes_only", "1")
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var history [][]State
	if err := c.doJSON(ctx, "GET", path, nil, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// GetLogbook returns logbook entries.
func (c *Client) GetLogbook(ctx context.Context, entityID string, startTime *time.Time) ([]LogEntry, error) {
	path := "/logbook"
	if startTime != nil {
		path += "/" + url.PathEscape(startTime.Format(time.RFC3339))
	}

	params := url.Values{}
	if entityID != "" {
		params.Set("entity", entityID)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var entries []LogEntry
	if err := c.doJSON(ctx, "GET", path, nil, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetConfig returns the HA configuration.
func (c *Client) GetConfig(ctx context.Context) (map[string]any, error) {
	var cfg map[string]any
	if err := c.doJSON(ctx, "GET", "/config", nil, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CheckConfig validates the HA configuration.
func (c *Client) CheckConfig(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	if err := c.doJSON(ctx, "POST", "/config/core/check_config", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetComponents returns a list of loaded components.
func (c *Client) GetComponents(ctx context.Context) ([]string, error) {
	var components []string
	if err := c.doJSON(ctx, "GET", "/components", nil, &components); err != nil {
		return nil, err
	}
	return components, nil
}

// GetCalendars returns all calendar entities.
func (c *Client) GetCalendars(ctx context.Context) ([]Calendar, error) {
	var calendars []Calendar
	if err := c.doJSON(ctx, "GET", "/calendars", nil, &calendars); err != nil {
		return nil, err
	}
	return calendars, nil
}

// GetCalendarEvents returns events for a specific calendar.
func (c *Client) GetCalendarEvents(ctx context.Context, calendarID, start, end string) ([]CalendarEvent, error) {
	params := url.Values{}
	params.Set("start", start)
	params.Set("end", end)

	path := "/calendars/" + url.PathEscape(calendarID) + "?" + params.Encode()

	var events []CalendarEvent
	if err := c.doJSON(ctx, "GET", path, nil, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// RenderTemplate renders a Home Assistant template.
func (c *Client) RenderTemplate(ctx context.Context, tmpl string) (string, error) {
	payload := map[string]string{"template": tmpl}
	return c.doText(ctx, "POST", "/template", payload)
}

// GetErrorLog returns the HA error log.
func (c *Client) GetErrorLog(ctx context.Context) (string, error) {
	return c.doText(ctx, "GET", "/error_log", nil)
}

// GetAutomationConfig returns the full stored config for an automation by its numeric ID.
func (c *Client) GetAutomationConfig(ctx context.Context, id string) (map[string]any, error) {
	var result map[string]any
	if err := c.doJSON(ctx, "GET", "/config/automation/config/"+url.PathEscape(id), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateAutomation creates or replaces an automation config by its numeric ID.
func (c *Client) UpdateAutomation(ctx context.Context, id string, cfg map[string]any) (map[string]any, error) {
	var result map[string]any
	if err := c.doJSON(ctx, "POST", "/config/automation/config/"+url.PathEscape(id), cfg, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteAutomation deletes an automation by its numeric ID.
func (c *Client) DeleteAutomation(ctx context.Context, id string) error {
	return c.doDelete(ctx, "/config/automation/config/"+url.PathEscape(id))
}
