package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(id string, userID int, hub *Hub) *Client {
	return &Client{ID: id, UserID: userID, hub: hub, send: make(chan []byte, 10)}
}

func receive(t *testing.T, c *Client) Message {
	t.Helper()
	select {
	case data := <-c.send:
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		return msg
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected message to be received within 200ms")
		return Message{}
	}
}

func expectSilence(t *testing.T, c *Client) {
	t.Helper()
	select {
	case data := <-c.send:
		t.Fatalf("expected no message, got %s", data)
	case <-time.After(50 * time.Millisecond):
	}
}

// ===== Hub Creation =====

func TestNewHub_CreatesEmptyHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients on new hub, got %d", hub.ClientCount())
	}
}

// ===== Client Registration =====

func TestRegisterAndUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := newTestClient("test-client", 1, hub)

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client after register, got %d", hub.ClientCount())
	}

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)
	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", hub.ClientCount())
	}
}

// ===== Targeted delivery =====

func TestSendToUsers_DeliversOnlyToRecipients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	alice := newTestClient("alice", 1, hub)
	bob := newTestClient("bob", 2, hub)
	hub.Register(alice)
	hub.Register(bob)
	time.Sleep(10 * time.Millisecond)

	hub.SendToUsers([]int{1}, Message{Type: "test", Data: "hello"})

	if got := receive(t, alice); got.Type != "test" {
		t.Errorf("expected type 'test', got %q", got.Type)
	}
	expectSilence(t, bob)

	hub.Unregister(alice)
	hub.Unregister(bob)
}

func TestSendToUsers_EmptyRecipients_NoDelivery(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := newTestClient("c", 1, hub)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.SendToUsers(nil, Message{Type: "test"})
	expectSilence(t, client)

	hub.Unregister(client)
}

func TestSendWorkflowRunUpdate_SendsCorrectType(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := newTestClient("wf-receiver", 7, hub)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.SendWorkflowRunUpdate([]int{7}, map[string]string{"id": "123"})

	if got := receive(t, client); got.Type != "workflow_run" {
		t.Errorf("expected type 'workflow_run', got %q", got.Type)
	}

	hub.Unregister(client)
}

func TestSendSyncEvents_OnlyToInitiator(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	initiator := newTestClient("initiator", 1, hub)
	other := newTestClient("other", 2, hub)
	hub.Register(initiator)
	hub.Register(other)
	time.Sleep(10 * time.Millisecond)

	hub.SendSyncStart(1, 10)
	if got := receive(t, initiator); got.Type != "sync:start" {
		t.Errorf("expected sync:start, got %q", got.Type)
	}
	expectSilence(t, other)

	hub.SendSyncComplete(1, 5, 20, 100)
	if got := receive(t, initiator); got.Type != "sync:complete" {
		t.Errorf("expected sync:complete, got %q", got.Type)
	}

	hub.SendSyncError(1, "something went wrong")
	if got := receive(t, initiator); got.Type != "sync:error" {
		t.Errorf("expected sync:error, got %q", got.Type)
	}

	hub.Unregister(initiator)
	hub.Unregister(other)
}

func TestSendSyncProgress_ComputesPercentage(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	client := newTestClient("progress-receiver", 1, hub)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.SendSyncProgress(1, 5, 10, "my-repo")

	select {
	case data := <-client.send:
		var msg struct {
			Type string `json:"type"`
			Data struct {
				Progress float64 `json:"progress"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if msg.Type != "sync:progress" || msg.Data.Progress != 50 {
			t.Errorf("unexpected message %s", data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("expected sync:progress message to be delivered")
	}

	// Zero total must not divide by zero
	hub.SendSyncProgress(1, 0, 0, "")
	select {
	case data := <-client.send:
		var msg struct {
			Data struct {
				Progress float64 `json:"progress"`
			} `json:"data"`
		}
		_ = json.Unmarshal(data, &msg)
		if msg.Data.Progress != 0 {
			t.Errorf("expected 0%% progress for zero total, got %v", msg.Data.Progress)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("expected message to be delivered")
	}

	hub.Unregister(client)
}

func TestDeliver_SlowClientIsDisconnected(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	slow := &Client{ID: "slow", UserID: 1, hub: hub, send: make(chan []byte)} // unbuffered, nobody reads
	fast := newTestClient("fast", 1, hub)
	hub.Register(slow)
	hub.Register(fast)
	time.Sleep(10 * time.Millisecond)

	hub.SendToUser(1, Message{Type: "test"})
	receive(t, fast)
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("expected slow client to be removed, got %d clients", hub.ClientCount())
	}
	if _, open := <-slow.send; open {
		t.Error("expected slow client channel to be closed")
	}

	hub.Unregister(fast)
}

// ===== GetUpgraderWithOrigin =====

func TestGetUpgraderWithOrigin_AllowsSpecificOrigin(t *testing.T) {
	upgrader := GetUpgraderWithOrigin("https://myapp.example.com")

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://myapp.example.com")

	if !upgrader.CheckOrigin(req) {
		t.Error("expected specific origin to be allowed")
	}
}

func TestGetUpgraderWithOrigin_BlocksDifferentOrigin(t *testing.T) {
	upgrader := GetUpgraderWithOrigin("https://myapp.example.com")

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	if upgrader.CheckOrigin(req) {
		t.Error("expected different origin to be blocked")
	}
}

func TestGetUpgraderWithOrigin_EmptyDeniesAll(t *testing.T) {
	upgrader := GetUpgraderWithOrigin("")

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://any-origin.com")

	if upgrader.CheckOrigin(req) {
		t.Error("expected every origin to be denied when no origin is configured")
	}
}

// ===== ClientCount =====

func TestClientCount_MultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	clients := make([]*Client, 3)
	for i := range clients {
		clients[i] = newTestClient("client-"+string(rune('A'+i)), i+1, hub)
		hub.Register(clients[i])
	}
	time.Sleep(20 * time.Millisecond)

	if hub.ClientCount() != 3 {
		t.Errorf("expected 3 clients, got %d", hub.ClientCount())
	}

	for _, c := range clients {
		hub.Unregister(c)
	}
	time.Sleep(20 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after all unregistered, got %d", hub.ClientCount())
	}
}
