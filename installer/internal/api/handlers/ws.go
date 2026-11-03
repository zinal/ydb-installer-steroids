package handlers

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/logs"
	"ydb-installer/internal/services/session"
)

// upgrader is the shared WebSocket upgrader used by LogsWebSocket.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // allow all origins in dev
}

const (
	// pingInterval is how often the server sends a WebSocket ping to
	// keep the connection alive through proxies and browser timeouts.
	pingInterval = 30 * time.Second

	// pongWait is how long the server waits for a pong response before
	// considering the client disconnected.
	pongWait = 60 * time.Second

	// writeTimeout is the deadline for each write operation.
	writeTimeout = 10 * time.Second
)

// LogsWebSocket handles GET /ws/logs.
//
// It upgrades the HTTP connection to a WebSocket (RFC 6455) using
// gorilla/websocket and streams log lines from the LogBroadcaster.
// Each line is sent as a WebSocket text message.
//
// Heartbeat:
//   - Server sends a Ping frame every 30 seconds.
//   - Client must respond with a Pong within 60 seconds, otherwise
//     the connection is considered dead and closed.
//   - A read goroutine monitors the connection for client-side close.
//
// Fallback: if the client does not send WebSocket upgrade headers
// (e.g. curl), the response is streamed as Server-Sent Events.
//
// The handler is flow-aware: it looks up the current session flow and
// subscribes to the corresponding Broadcaster so that install and delete
// log streams are completely isolated.
func LogsWebSocket(states *models.OperationStates, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Optional ?op=<flow> overrides session flow (ydb-flow EM phase).
		state := resolveInstallState(states, store, r)
		if state == nil || state.Broadcaster == nil {
			http.Error(w, "no active flow or broadcaster not available", http.StatusServiceUnavailable)
			return
		}
		broadcaster := state.Broadcaster

		if r.Header.Get("Upgrade") != "websocket" {
			streamSSE(w, r, broadcaster)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws] Upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		sub := broadcaster.Subscribe()
		defer sub.Close()

		done := make(chan struct{})
		go func() {
			defer close(done)
			conn.SetReadDeadline(time.Now().Add(pongWait))
			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(pongWait))
				return nil
			})
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
						log.Printf("[ws] Read error (client disconnected): %v", err)
					}
					return
				}
			}
		}()

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case line, ok := <-sub.Ch:
				if !ok {
					_ = conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, "operation complete"))
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
					log.Printf("[ws] Write error: %v", err)
					return
				}

			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("[ws] Ping error: %v", err)
					return
				}

			case <-done:
				return

			case <-r.Context().Done():
				log.Printf("[ws] Request context done, exiting")
				return
			}
		}
	}
}

// streamSSE streams log lines as Server-Sent Events using the broadcaster.
func streamSSE(w http.ResponseWriter, r *http.Request, broadcaster *logs.LogBroadcaster) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := broadcaster.Subscribe()
	defer sub.Close()

	for {
		select {
		case line, ok := <-sub.Ch:
			if !ok {
				_, _ = fmt.Fprintf(w, "event: done\ndata: \n\n")
				flusher.Flush()
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// GetLogsFromBroadcaster handles GET /api/install/logs.
// It returns the current ring buffer contents as a JSON array.
func GetLogsFromBroadcaster(broadcaster *logs.LogBroadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		lines := broadcaster.Snapshot()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs":   lines,
			"source": "ringbuffer",
			"count":  len(lines),
		})
	}
}

// upgradeWebSocket performs the RFC 6455 handshake via HTTP hijacking.
// Returns the raw net.Conn and a buffered writer for sending frames.
func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (interface{ Close() error }, *bufio.ReadWriter, error) {
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijacking not supported by this server")
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}

	accept := wsAccept(key)
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"

	if _, err := bufrw.WriteString(resp); err != nil {
		conn.Close()
		return nil, nil, err
	}
	if err := bufrw.Flush(); err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, bufrw, nil
}

// wsAccept computes the Sec-WebSocket-Accept value per RFC 6455 §1.3.
func wsAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// writeWSTextFrame writes a single RFC 6455 text frame to the buffered writer.
// FIN=1, opcode=0x1 (text), no masking (server→client direction).
func writeWSTextFrame(bufrw *bufio.ReadWriter, text string) error {
	payload := []byte(text)
	n := len(payload)

	if err := bufrw.WriteByte(0x81); err != nil {
		return err
	}

	switch {
	case n <= 125:
		if err := bufrw.WriteByte(byte(n)); err != nil {
			return err
		}
	case n <= 65535:
		if err := bufrw.WriteByte(126); err != nil {
			return err
		}
		if err := bufrw.WriteByte(byte(n >> 8)); err != nil {
			return err
		}
		if err := bufrw.WriteByte(byte(n)); err != nil {
			return err
		}
	default:
		if err := bufrw.WriteByte(127); err != nil {
			return err
		}
		for i := 7; i >= 0; i-- {
			if err := bufrw.WriteByte(byte(n >> (uint(i) * 8))); err != nil {
				return err
			}
		}
	}

	if _, err := bufrw.Write(payload); err != nil {
		return err
	}
	return bufrw.Flush()
}
