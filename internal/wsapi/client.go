package wsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Message represents a generic WebSocket API message.
type Message struct {
	ID   int    `json:"id,omitempty"`
	Type string `json:"type"`
}

// AuthMessage is sent to authenticate with the WebSocket API.
type AuthMessage struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token"`
}

// CallMessage is sent to call a WebSocket API command.
type CallMessage struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

// ResultMessage is the response to a command.
type ResultMessage struct {
	ID      int             `json:"id"`
	Type    string          `json:"type"`
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Conn is a WebSocket connection abstraction used by Client.
// It is satisfied by gorilla/websocket *websocket.Conn but allows testing.
type Conn interface {
	ReadJSON(v any) error
	WriteJSON(v any) error
	Close() error
}

// Dialer dials a WebSocket connection.
type Dialer func(ctx context.Context, url string, header http.Header) (Conn, error)

// Client is a WebSocket API client for Home Assistant.
type Client struct {
	conn   Conn
	token  string
	nextID int
	mu     sync.Mutex

	done chan struct{}
	once sync.Once
	wg   sync.WaitGroup

	pending map[int]chan *ResultMessage
}

// Connect establishes a WebSocket connection and authenticates.
func Connect(ctx context.Context, wsURL, token string, dial Dialer) (*Client, error) {
	conn, err := dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial websocket: %w", err)
	}

	c := &Client{
		conn:    conn,
		token:   token,
		nextID:  1,
		done:    make(chan struct{}),
		pending: make(map[int]chan *ResultMessage),
	}

	// Read the auth_required message.
	var msg Message
	if err := conn.ReadJSON(&msg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("read auth_required: %w", err)
	}
	if msg.Type != "auth_required" {
		conn.Close()
		return nil, fmt.Errorf("expected auth_required, got %q", msg.Type)
	}

	// Send auth.
	if err := conn.WriteJSON(AuthMessage{Type: "auth", AccessToken: token}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send auth: %w", err)
	}

	// Read auth result.
	var authResult Message
	if err := conn.ReadJSON(&authResult); err != nil {
		conn.Close()
		return nil, fmt.Errorf("read auth result: %w", err)
	}
	if authResult.Type != "auth_ok" {
		conn.Close()
		return nil, fmt.Errorf("authentication failed: %q", authResult.Type)
	}

	c.wg.Add(1)
	go c.readLoop()

	return c, nil
}

// readLoop dispatches incoming messages to waiting callers.
func (c *Client) readLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		default:
		}

		var result ResultMessage
		if err := c.conn.ReadJSON(&result); err != nil {
			// Check if we're shutting down.
			select {
			case <-c.done:
				return
			default:
			}
			// Connection error — signal all pending callers.
			c.mu.Lock()
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[int]chan *ResultMessage)
			c.mu.Unlock()
			return
		}

		c.mu.Lock()
		ch, ok := c.pending[result.ID]
		if ok {
			delete(c.pending, result.ID)
		}
		c.mu.Unlock()

		if ok {
			select {
			case ch <- &result:
			case <-c.done:
				return
			}
		}
	}
}

// call sends a message and waits for the result, with a 10-second timeout.
func (c *Client) call(ctx context.Context, msg any, id int) (*ResultMessage, error) {
	ch := make(chan *ResultMessage, 1)

	c.mu.Lock()
	select {
	case <-c.done:
		c.mu.Unlock()
		return nil, fmt.Errorf("connection closed")
	default:
	}
	c.pending[id] = ch
	if err := c.conn.WriteJSON(msg); err != nil {
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("write message: %w", err)
	}
	c.mu.Unlock()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	select {
	case result, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("connection closed while waiting for response")
		}
		if !result.Success {
			if result.Error != nil {
				return nil, fmt.Errorf("API error %s: %s", result.Error.Code, result.Error.Message)
			}
			return nil, fmt.Errorf("command failed")
		}
		return result, nil
	case <-timer.C:
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response")
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		return nil, fmt.Errorf("connection closed")
	}
}

// nextMessageID returns a new unique message ID.
func (c *Client) nextMessageID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextID
	c.nextID++
	return id
}

// GetStates retrieves all entity states via the WebSocket API.
func (c *Client) GetStates(ctx context.Context) (json.RawMessage, error) {
	id := c.nextMessageID()
	msg := map[string]any{"id": id, "type": "get_states"}
	result, err := c.call(ctx, msg, id)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

// Close signals the readLoop to stop and waits for it to exit.
func (c *Client) Close() error {
	var closeErr error
	c.once.Do(func() {
		close(c.done)
		closeErr = c.conn.Close()
		c.wg.Wait()
	})
	return closeErr
}
