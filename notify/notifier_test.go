package notify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slack-go/slack"
)

type mockNotifier struct {
	called bool
	err    error
}

func (m *mockNotifier) Notify(ctx context.Context, summary string) error {
	m.called = true
	return m.err
}

func TestMulti_Notify(t *testing.T) {
	n1 := &mockNotifier{}
	n2 := &mockNotifier{}

	multi := NewMulti(n1, n2)
	err := multi.Notify(context.Background(), "test message")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !n1.called || !n2.called {
		t.Error("expected both notifiers to be called")
	}
}

func TestMulti_Notify_WithError(t *testing.T) {
	n1 := &mockNotifier{err: errors.New("fail")}
	n2 := &mockNotifier{}

	multi := NewMulti(n1, n2)
	err := multi.Notify(context.Background(), "test message")
	if err == nil {
		t.Error("expected error, got nil")
	}

	if !n1.called || !n2.called {
		t.Error("expected both notifiers to be called even if one fails")
	}
}

func TestSlackNotifier_Notify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Errorf("unexpected URL path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			OK bool `json:"ok"`
		}{OK: true})
	}))
	defer server.Close()

	notifier := NewSlackNotifier("fake-token", "fake-channel")
	notifier.client = slack.New("fake-token", slack.OptionAPIURL(server.URL+"/"))

	err := notifier.Notify(context.Background(), "hello from test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
