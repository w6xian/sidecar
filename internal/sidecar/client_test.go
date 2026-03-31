package sidecar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/w6xian/sidecar/internal/config"
	"github.com/w6xian/sidecar/internal/protocol"
	"github.com/w6xian/sidecar/pkg/logger"
)

func initTestLogger(t *testing.T) {
	t.Helper()
	logger.Init(&config.Logger{Stdout: false}, &lumberjack.Logger{
		Filename: filepath.Join(os.TempDir(), "sidecar-test.log"),
	})
}

func TestClientConnectAuthFailureBackoffsAsConnectError(t *testing.T) {
	initTestLogger(t)

	expectedToken := "good-token"
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}

		if msg.Token != expectedToken {
			errMsg, _ := protocol.NewErrorMessage("", 401, "authentication failed")
			b, _ := json.Marshal(errMsg)
			conn.WriteMessage(websocket.TextMessage, b)
			return
		}

		ackPayload := &protocol.RegisterACK{
			Success:   true,
			Message:   "registered successfully",
			Accepted:  []string{},
			ConnAlias: "",
		}
		ackMsg, _ := protocol.NewMessage(protocol.MsgRegisterACK, "", ackPayload)
		b, _ := json.Marshal(ackMsg)
		conn.WriteMessage(websocket.TextMessage, b)
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	c := NewClient(&config.Profile{
		Sidecar: &config.SidecarConfig{
			ServerAddr:           wsURL,
			ReconnectInterval:    10 * time.Millisecond,
			MaxReconnectInterval: 50 * time.Millisecond,
		},
	})

	err := c.connect()
	if err == nil {
		t.Fatalf("expected connect() to return auth error")
	}
	if !strings.Contains(err.Error(), "register rejected: 401") {
		t.Fatalf("unexpected error: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		t.Fatalf("expected conn to be nil after failed connect")
	}

	c2 := NewClient(&config.Profile{
		Sidecar: &config.SidecarConfig{
			ServerAddr:           wsURL,
			ReconnectInterval:    10 * time.Millisecond,
			MaxReconnectInterval: 50 * time.Millisecond,
		},
	})

	if err := c2.connect(); err != nil {
		t.Fatalf("expected connect() to succeed, got %v", err)
	}

	c2.mu.Lock()
	if c2.conn != nil {
		c2.conn.Close()
		c2.conn = nil
	}
	c2.mu.Unlock()
}
