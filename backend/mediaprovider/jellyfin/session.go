package jellyfin

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// SessionInfo holds the active Jellyfin session credentials and HTTP client
// needed for establishing a remote-control WebSocket and reporting capabilities.
type SessionInfo struct {
	BaseURL    string
	Token      string
	UserID     string
	DeviceID   string
	AuthHeader string
	HTTPClient *http.Client
	TLSConfig  *tls.Config
}

var deviceIDRegex = regexp.MustCompile(`DeviceId="([^"]+)"`)

type sessionCaptureTransport struct {
	base http.RoundTripper

	mu         sync.RWMutex
	token      string
	userID     string
	deviceID   string
	authHeader string
}

func newSessionCaptureTransport(base http.RoundTripper) *sessionCaptureTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &sessionCaptureTransport{base: base}
}

func (t *sessionCaptureTransport) tlsConfig() *tls.Config {
	if tr, ok := t.base.(*http.Transport); ok && tr.TLSClientConfig != nil {
		return tr.TLSClientConfig.Clone()
	}
	return nil
}

func (t *sessionCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if tok := req.Header.Get("X-Emby-Token"); tok != "" {
		t.mu.Lock()
		t.token = tok
		t.mu.Unlock()
	}
	if auth := req.Header.Get("Authorization"); auth != "" {
		t.mu.Lock()
		// Prefer the Authorization header that includes Token="..." once logged in.
		if t.authHeader == "" || strings.Contains(auth, "Token=") {
			t.authHeader = auth
		}
		if m := deviceIDRegex.FindStringSubmatch(auth); len(m) == 2 && m[1] != "" {
			t.deviceID = m[1]
		}
		t.mu.Unlock()
	}
	if q := req.URL.Query(); len(q) > 0 {
		t.mu.Lock()
		if uid := q.Get("UserId"); uid != "" {
			t.userID = uid
		}
		if did := q.Get("DeviceId"); did != "" {
			t.deviceID = did
		}
		t.mu.Unlock()
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK && strings.HasSuffix(strings.ToLower(req.URL.Path), "/users/authenticatebyname") {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		if readErr == nil {
			var authResp struct {
				AccessToken string `json:"AccessToken"`
				User        struct {
					ID string `json:"Id"`
				} `json:"User"`
			}
			if json.Unmarshal(bodyBytes, &authResp) == nil {
				t.mu.Lock()
				if authResp.AccessToken != "" {
					t.token = authResp.AccessToken
				}
				if authResp.User.ID != "" {
					t.userID = authResp.User.ID
				}
				// go-jellyfin resets deviceID on Login and recalculates it on the next request.
				t.deviceID = ""
				t.authHeader = ""
				t.mu.Unlock()
			}
		}
	}

	return resp, nil
}

func (t *sessionCaptureTransport) snapshot() (token, userID, deviceID, authHeader string) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.token, t.userID, t.deviceID, t.authHeader
}

// AttachSessionCapture wraps the underlying Jellyfin HTTP client transport so
// session credentials (token, user ID, device ID, authorization header) can be
// inspected for remote control without modifying go-jellyfin.
func (j *JellyfinServer) AttachSessionCapture() {
	if j.Client.HTTPClient == nil {
		j.Client.HTTPClient = &http.Client{}
	}
	if sc, ok := j.Client.HTTPClient.Transport.(*sessionCaptureTransport); ok {
		j.sessionCapture = sc
		return
	}
	sc := newSessionCaptureTransport(j.Client.HTTPClient.Transport)
	j.Client.HTTPClient.Transport = sc
	j.sessionCapture = sc
}

// GetSessionInfo returns the active Jellyfin session information.
func (j *JellyfinMediaProvider) GetSessionInfo() (SessionInfo, error) {
	if j.sessionCapture == nil {
		return SessionInfo{}, errors.New("jellyfin session capture not attached")
	}
	token, userID, deviceID, authHeader := j.sessionCapture.snapshot()
	if deviceID == "" || authHeader == "" || token == "" {
		// Trigger a lightweight authenticated call so go-jellyfin computes and
		// sends its post-login DeviceId and Authorization header.
		_, _ = j.client.GetUserViews()
		token, userID, deviceID, authHeader = j.sessionCapture.snapshot()
	}
	if token == "" || deviceID == "" {
		return SessionInfo{}, errors.New("jellyfin session not authenticated")
	}
	return SessionInfo{
		BaseURL:    j.client.BaseURL().String(),
		Token:      token,
		UserID:     userID,
		DeviceID:   deviceID,
		AuthHeader: authHeader,
		HTTPClient: j.client.HTTPClient,
		TLSConfig:  j.sessionCapture.tlsConfig(),
	}, nil
}
