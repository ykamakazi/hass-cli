package wsapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// DefaultDialer is a Dialer backed by gorilla/websocket.
func DefaultDialer(ctx context.Context, haURL string, header http.Header) (Conn, error) {
	// Convert http(s) → ws(s).
	wsURL := strings.Replace(haURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/api/websocket"

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	return conn, err
}
