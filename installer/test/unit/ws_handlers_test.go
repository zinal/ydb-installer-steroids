package unit_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ydb-installer/internal/api"
	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
	"ydb-installer/internal/services/logs"
	"ydb-installer/internal/services/session"
)

// setSessionFlow switches the session flow via PUT /api/session.
func setSessionFlow(t *testing.T, base, flow string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"flow": flow})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, base+"/api/session", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("flow switch status = %d, want 200", resp.StatusCode)
	}
}

// waitForSubscribers polls the broadcaster until it has at least n
// subscribers (or fails the test after 5 seconds).
func waitForSubscribers(t *testing.T, b *logs.LogBroadcaster, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b.SubscriberCount() >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d broadcaster subscriber(s)", n)
}

// wsURL converts an httptest http:// URL into a ws:// URL.
func wsURL(httpURL, path string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http") + path
}

// newWSServer builds a router with session store + operation states + the
// inventory WS hub wired, returning the server and its backing objects.
func newWSServer(t *testing.T) (*session.Store, *models.OperationStates, *handlers.InventoryWSHub, string) {
	t.Helper()
	store := session.New("en")
	states := models.NewOperationStates()
	hub := handlers.NewInventoryWSHub()
	h := api.New(api.Options{
		SessionStore:    store,
		OperationStates: states,
		InventoryWSHub:  hub,
	})
	srv := newServerFromHandler(h)
	t.Cleanup(srv.Close)
	return store, states, hub, srv.URL
}

func TestLogsWebSocket_NoFlowReturns503(t *testing.T) {
	_, _, _, base := newWSServer(t)

	resp, err := http.Get(base + "/ws/logs")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (no active flow)", resp.StatusCode)
	}
}

func TestLogsWebSocket_SSEFallbackStreamsLines(t *testing.T) {
	_, states, _, base := newWSServer(t)
	setSessionFlow(t, base, "ydb")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/ws/logs", nil)
	if err != nil {
		t.Fatal(err)
	}

	type sseResult struct {
		contentType string
		line        string
		err         error
	}
	resCh := make(chan sseResult, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resCh <- sseResult{err: err}
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "data: ") {
				resCh <- sseResult{
					contentType: resp.Header.Get("Content-Type"),
					line:        strings.TrimPrefix(scanner.Text(), "data: "),
				}
				return
			}
		}
		resCh <- sseResult{err: scanner.Err()}
	}()

	waitForSubscribers(t, states.Install.Broadcaster, 1)
	states.Install.WriteLog("hello-sse")

	select {
	case r := <-resCh:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.contentType != "text/event-stream" {
			t.Errorf("Content-Type = %q, want text/event-stream", r.contentType)
		}
		if r.line != "hello-sse" {
			t.Errorf("SSE line = %q, want hello-sse", r.line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for SSE line")
	}
}

func TestLogsWebSocket_OpOverrideSelectsFlow(t *testing.T) {
	_, states, _, base := newWSServer(t)
	// No session flow set — the ?op= override must still resolve a state.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/ws/logs?op=ydb", nil)
	if err != nil {
		t.Fatal(err)
	}

	type sseResult struct {
		status      int
		contentType string
		line        string
		err         error
	}
	resCh := make(chan sseResult, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resCh <- sseResult{err: err}
			return
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "data: ") {
				resCh <- sseResult{
					status:      resp.StatusCode,
					contentType: resp.Header.Get("Content-Type"),
					line:        strings.TrimPrefix(scanner.Text(), "data: "),
				}
				return
			}
		}
		resCh <- sseResult{status: resp.StatusCode, err: scanner.Err()}
	}()

	// The override must resolve the Install state without any session flow;
	// headers are flushed on the first published line.
	waitForSubscribers(t, states.Install.Broadcaster, 1)
	states.Install.WriteLog("hello-op")

	select {
	case r := <-resCh:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.status != http.StatusOK {
			t.Fatalf("status = %d, want 200 (op override resolved)", r.status)
		}
		if r.contentType != "text/event-stream" {
			t.Errorf("Content-Type = %q, want text/event-stream", r.contentType)
		}
		if r.line != "hello-op" {
			t.Errorf("SSE line = %q, want hello-op", r.line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for SSE line via ?op= override")
	}
}

func TestLogsWebSocket_WebSocketStreamsLines(t *testing.T) {
	_, states, _, base := newWSServer(t)
	setSessionFlow(t, base, "ydb")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(base, "/ws/logs"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	waitForSubscribers(t, states.Install.Broadcaster, 1)
	states.Install.WriteLog("hello-ws")

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(msg) != "hello-ws" {
		t.Errorf("WS message = %q, want hello-ws", msg)
	}
}

func TestInventoryWS_NotUpgradeReturns400(t *testing.T) {
	_, _, _, base := newWSServer(t)

	resp, err := http.Get(base + "/ws/inventory")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (upgrade required)", resp.StatusCode)
	}
}

func TestInventoryWS_BroadcastReachesClient(t *testing.T) {
	_, _, hub, base := newWSServer(t)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(base, "/ws/inventory"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Initial handshake message.
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read connected message failed: %v", err)
	}
	if string(msg) != `{"event":"connected"}` {
		t.Errorf("connected message = %q", msg)
	}

	// Hub broadcast must reach the connected client.
	hub.Broadcast([]byte(`{"event":"changed"}`))
	_, msg, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read broadcast message failed: %v", err)
	}
	if string(msg) != `{"event":"changed"}` {
		t.Errorf("broadcast message = %q", msg)
	}
}
