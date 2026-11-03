// Package handlers contains the HTTP handler functions exposed by the
// installer's REST API.
package handlers

import (
	"log"
	"net/http"
	"sync"
)

// InventoryWSHub manages WebSocket connections for inventory change notifications.
type InventoryWSHub struct {
	mu          sync.RWMutex
	connections []chan []byte
}

// NewInventoryWSHub creates a new hub for inventory WebSocket connections.
func NewInventoryWSHub() *InventoryWSHub {
	return &InventoryWSHub{}
}

// Broadcast sends a message to all connected clients.
func (h *InventoryWSHub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.connections {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

// InventoryWebSocket handles GET /ws/inventory.
// Sends a text message when the inventory file changes externally.
func (h *InventoryWSHub) InventoryWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrade := r.Header.Get("Upgrade")
		connection := r.Header.Get("Connection")
		wsKey := r.Header.Get("Sec-WebSocket-Key")

		log.Printf("WebSocket upgrade check: Upgrade=%q, Connection=%q, Sec-WebSocket-Key=%q", upgrade, connection, wsKey)

		if upgrade != "websocket" || connection != "Upgrade" || wsKey == "" {
			http.Error(w, "WebSocket upgrade required", http.StatusBadRequest)
			return
		}

		conn, bufrw, err := upgradeWebSocket(w, r)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()

		msgCh := make(chan []byte, 16)
		h.mu.Lock()
		h.connections = append(h.connections, msgCh)
		h.mu.Unlock()
		defer func() {
			h.mu.Lock()
			for i, c := range h.connections {
				if c == msgCh {
					h.connections = append(h.connections[:i], h.connections[i+1:]...)
					break
				}
			}
			h.mu.Unlock()
		}()

		_ = writeWSTextFrame(bufrw, `{"event":"connected"}`)

		for {
			select {
			case msg := <-msgCh:
				if err := writeWSTextFrame(bufrw, string(msg)); err != nil {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	}
}
