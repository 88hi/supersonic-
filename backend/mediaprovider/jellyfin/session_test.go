package jellyfin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	jf "github.com/dweymouth/go-jellyfin"
)

func TestSessionCaptureTransportAndGetSessionInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/System/Info/Public":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ServerName": "TestJellyfin",
				"Version":    "10.10.0",
			})
		case "/Users/authenticatebyname":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"AccessToken": "tok-12345",
				"ServerId":    "srv-1",
				"User": map[string]any{
					"Name":     "alice",
					"ServerId": "srv-1",
					"Id":       "user-999",
				},
			})
		case "/Users/user-999/Views":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Items": []any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cli, err := jf.NewClient(srv.URL, "Supersonic", "0.21.0")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	js := &JellyfinServer{Client: *cli}
	js.AttachSessionCapture()

	resp := js.Login("alice", "secret")
	if resp.Error != nil {
		t.Fatalf("Login failed: %v", resp.Error)
	}

	mp := js.MediaProvider().(*JellyfinMediaProvider)
	info, err := mp.GetSessionInfo()
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}
	if info.Token != "tok-12345" {
		t.Errorf("Token = %q, want %q", info.Token, "tok-12345")
	}
	if info.UserID != "user-999" {
		t.Errorf("UserID = %q, want %q", info.UserID, "user-999")
	}
	if info.DeviceID == "" {
		t.Errorf("DeviceID should not be empty")
	}
	if info.AuthHeader == "" {
		t.Errorf("AuthHeader should not be empty")
	}
}
