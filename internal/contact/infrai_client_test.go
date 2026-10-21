package contact

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyCaptchaIncludesWidgetRecordID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/captcha/verify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["widget_record_id"] != "widget-1" || body["token"] != "token-1" {
			t.Errorf("unexpected captcha request: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewInfraiClient(server.URL, "test-key", server.Client())
	if err := client.VerifyCaptcha(context.Background(), "widget-1", "token-1", "192.0.2.10"); err != nil {
		t.Fatal(err)
	}
}
