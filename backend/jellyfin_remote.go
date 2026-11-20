package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/supersonic-app/supersonic/backend/mediaprovider"
	jellyfinMP "github.com/supersonic-app/supersonic/backend/mediaprovider/jellyfin"
	"github.com/supersonic-app/supersonic/backend/player"
	"golang.org/x/net/websocket"
)

const jellyfinTicksPerSecond = 10_000_000

// JellyfinRemoteController exposes Supersonic as a remote-controllable Jellyfin
// playback session (Cast / "Play On" target) over Jellyfin's WebSocket API.
type JellyfinRemoteController struct {
	pm *PlaybackManager
	sm *ServerManager

	OnRemoteMessage func(header, text string)

	mu         sync.Mutex
	running    bool
	cancel     context.CancelFunc
	ws         *websocket.Conn
	session    jellyfinMP.SessionInfo
	hasSession bool
	lastItemID string
}

func NewJellyfinRemoteController(pm *PlaybackManager, sm *ServerManager) *JellyfinRemoteController {
	rc := &JellyfinRemoteController{
		pm: pm,
		sm: sm,
	}
	rc.registerPlaybackHooks()
	return rc
}

func (r *JellyfinRemoteController) registerPlaybackHooks() {
	r.pm.OnSongChange(func(item mediaprovider.MediaItem, _ *mediaprovider.Track) {
		if item == nil {
			go r.reportPlaybackStopped()
			return
		}
		if item.Metadata().Type == mediaprovider.MediaItemTypeTrack {
			r.mu.Lock()
			r.lastItemID = item.Metadata().ID
			r.mu.Unlock()
			go r.reportPlaybackStart(item.Metadata().ID)
		}
	})
	r.pm.OnPlaying(func() {
		go r.reportPlaybackProgress("unpause")
	})
	r.pm.OnPaused(func() {
		go r.reportPlaybackProgress("pause")
	})
	r.pm.OnStopped(func() {
		go r.reportPlaybackStopped()
	})
	r.pm.OnSeek(func() {
		go r.reportPlaybackProgress("timeupdate")
	})
	r.pm.OnVolumeChange(func(_ int) {
		go r.reportPlaybackProgress("volumechange")
	})
	r.pm.OnLoopModeChange(func(_ LoopMode) {
		go r.reportPlaybackProgress("repeatmodechange")
	})
}

// Start begins advertising Supersonic as a Jellyfin remote control target if the
// currently connected server is a Jellyfin server. Safe to call repeatedly.
func (r *JellyfinRemoteController) Start(parentCtx context.Context) {
	r.Stop()

	jfProvider, ok := r.sm.GetServer().(*jellyfinMP.JellyfinMediaProvider)
	if !ok || jfProvider == nil {
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	r.mu.Lock()
	r.running = true
	r.cancel = cancel
	r.mu.Unlock()

	go r.runLoop(ctx, jfProvider)
}

// Stop disconnects the Jellyfin remote control WebSocket and stops reporting.
func (r *JellyfinRemoteController) Stop() {
	r.mu.Lock()
	cancel := r.cancel
	ws := r.ws
	r.cancel = nil
	r.ws = nil
	r.running = false
	r.hasSession = false
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if ws != nil {
		_ = ws.Close()
	}
}

func (r *JellyfinRemoteController) runLoop(ctx context.Context, provider *jellyfinMP.JellyfinMediaProvider) {
	backoff := 2 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		session, err := provider.GetSessionInfo()
		if err != nil {
			log.Printf("Jellyfin remote control: failed to get session info: %v", err)
			if !sleepContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		r.mu.Lock()
		r.session = session
		r.hasSession = true
		r.mu.Unlock()

		if err := r.postCapabilities(ctx, session); err != nil {
			log.Printf("Jellyfin remote control: failed to post capabilities: %v", err)
		}

		err = r.connectAndServe(ctx, session, provider)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("Jellyfin remote control: WebSocket disconnected: %v", err)
		}

		if !sleepContext(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func buildJellyfinWebSocketURL(baseURLStr, token, deviceID string) (string, string, error) {
	u, err := url.Parse(baseURLStr)
	if err != nil {
		return "", "", err
	}
	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/socket"
	q := u.Query()
	q.Set("api_key", token)
	q.Set("deviceId", deviceID)
	u.RawQuery = q.Encode()
	return u.String(), origin, nil
}

func (r *JellyfinRemoteController) connectAndServe(
	ctx context.Context,
	session jellyfinMP.SessionInfo,
	provider *jellyfinMP.JellyfinMediaProvider,
) error {
	wsURL, origin, err := buildJellyfinWebSocketURL(session.BaseURL, session.Token, session.DeviceID)
	if err != nil {
		return err
	}

	wsCfg, err := websocket.NewConfig(wsURL, origin)
	if err != nil {
		return err
	}
	wsCfg.TlsConfig = session.TLSConfig
	if session.AuthHeader != "" {
		wsCfg.Header.Set("Authorization", session.AuthHeader)
	}
	if session.Token != "" {
		wsCfg.Header.Set("X-Emby-Token", session.Token)
	}

	ws, err := websocket.DialConfig(wsCfg)
	if err != nil {
		return err
	}

	r.mu.Lock()
	if !r.running || ctx.Err() != nil {
		r.mu.Unlock()
		_ = ws.Close()
		return ctx.Err()
	}
	r.ws = ws
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		if r.ws == ws {
			r.ws = nil
		}
		r.mu.Unlock()
		_ = ws.Close()
	}()

	log.Printf("Jellyfin remote control: connected to %s", session.BaseURL)

	// Re-post capabilities once the WebSocket is open so the server marks the
	// session as remote-controllable immediately.
	if err := r.postCapabilities(ctx, session); err != nil {
		log.Printf("Jellyfin remote control: failed to refresh capabilities: %v", err)
	}

	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	// Close WebSocket promptly when parent context is canceled.
	go func() {
		<-connCtx.Done()
		_ = ws.Close()
	}()

	keepAliveIntervalCh := make(chan time.Duration, 1)
	go r.keepAliveLoop(connCtx, ws, keepAliveIntervalCh)
	go r.progressReportLoop(connCtx)

	for {
		var raw string
		if err := websocket.Message.Receive(ws, &raw); err != nil {
			return err
		}
		r.handleWSMessage(connCtx, raw, session, provider, keepAliveIntervalCh)
	}
}

func (r *JellyfinRemoteController) keepAliveLoop(ctx context.Context, ws *websocket.Conn, intervalCh <-chan time.Duration) {
	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case newInterval := <-intervalCh:
			if newInterval < 5*time.Second {
				newInterval = 5 * time.Second
			} else if newInterval > 30*time.Second {
				newInterval = 30 * time.Second
			}
			interval = newInterval
			ticker.Reset(interval)
		case <-ticker.C:
			if err := websocket.Message.Send(ws, `{"MessageType":"KeepAlive"}`); err != nil {
				return
			}
		}
	}
}

func (r *JellyfinRemoteController) progressReportLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if r.pm.PlaybackStatus().State == player.Playing {
				r.reportPlaybackProgress("timeupdate")
			}
		}
	}
}

type jfWSMessage struct {
	MessageType string          `json:"MessageType"`
	MessageId   string          `json:"MessageId,omitempty"`
	Data        json.RawMessage `json:"Data,omitempty"`
}

type jfPlayRequest struct {
	ItemIds            []string `json:"ItemIds"`
	StartPositionTicks *int64   `json:"StartPositionTicks"`
	PlayCommand        string   `json:"PlayCommand"`
	ControllingUserId  string   `json:"ControllingUserId"`
	StartIndex         *int     `json:"StartIndex"`
}

type jfPlaystateRequest struct {
	Command           string `json:"Command"`
	SeekPositionTicks *int64 `json:"SeekPositionTicks"`
}

type jfGeneralCommand struct {
	Name      string            `json:"Name"`
	Arguments map[string]string `json:"Arguments"`
}

func (r *JellyfinRemoteController) handleWSMessage(
	ctx context.Context,
	raw string,
	session jellyfinMP.SessionInfo,
	provider mediaprovider.MediaProvider,
	keepAliveIntervalCh chan<- time.Duration,
) {
	var msg jfWSMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return
	}

	switch msg.MessageType {
	case "ForceKeepAlive":
		var secs float64
		if json.Unmarshal(msg.Data, &secs) == nil && secs > 0 {
			d := time.Duration(secs/2) * time.Second
			select {
			case keepAliveIntervalCh <- d:
			default:
			}
		}
	case "KeepAlive":
		// Server heartbeat ack
	case "Play":
		var req jfPlayRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Jellyfin remote control: invalid Play payload: %v", err)
			return
		}
		go r.handlePlayCommand(ctx, session, provider, req)
	case "Playstate":
		var req jfPlaystateRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Jellyfin remote control: invalid Playstate payload: %v", err)
			return
		}
		r.handlePlaystateCommand(req)
	case "GeneralCommand":
		var cmd jfGeneralCommand
		if err := json.Unmarshal(msg.Data, &cmd); err != nil {
			log.Printf("Jellyfin remote control: invalid GeneralCommand payload: %v", err)
			return
		}
		r.handleGeneralCommand(cmd)
	}
}

func (r *JellyfinRemoteController) handlePlayCommand(
	ctx context.Context,
	session jellyfinMP.SessionInfo,
	provider mediaprovider.MediaProvider,
	req jfPlayRequest,
) {
	if len(req.ItemIds) == 0 || provider == nil {
		return
	}

	tracks := r.resolveItemIDsToTracks(ctx, session, provider, req.ItemIds)
	if len(tracks) == 0 {
		log.Printf("Jellyfin remote control: no playable audio tracks resolved for ItemIds=%v", req.ItemIds)
		return
	}

	switch req.PlayCommand {
	case "PlayNext":
		r.pm.LoadTracks(tracks, InsertNext, false)
	case "PlayLast":
		r.pm.LoadTracks(tracks, Append, false)
	default: // "PlayNow" or unspecified
		r.pm.LoadTracks(tracks, Replace, false)
		startIdx := 0
		if req.StartIndex != nil && *req.StartIndex >= 0 && *req.StartIndex < len(tracks) {
			startIdx = *req.StartIndex
		}
		r.pm.PlayTrackAt(startIdx)
		if req.StartPositionTicks != nil && *req.StartPositionTicks > 0 {
			sec := float64(*req.StartPositionTicks) / float64(jellyfinTicksPerSecond)
			r.pm.SeekSeconds(sec)
		}
	}
}

func (r *JellyfinRemoteController) resolveItemIDsToTracks(
	ctx context.Context,
	session jellyfinMP.SessionInfo,
	provider mediaprovider.MediaProvider,
	itemIDs []string,
) []*mediaprovider.Track {
	itemTypes := r.fetchItemTypes(ctx, session, itemIDs)

	var tracks []*mediaprovider.Track
	for _, id := range itemIDs {
		switch itemTypes[id] {
		case "MusicAlbum":
			if album, err := provider.GetAlbum(id); err == nil && album != nil {
				tracks = append(tracks, album.Tracks...)
			}
		case "Playlist":
			if pl, err := provider.GetPlaylist(id); err == nil && pl != nil {
				tracks = append(tracks, pl.Tracks...)
			}
		case "MusicArtist":
			if artistTracks, err := provider.GetArtistTracks(id); err == nil {
				tracks = append(tracks, artistTracks...)
			}
		default:
			if tr, err := provider.GetTrack(id); err == nil && tr != nil {
				tracks = append(tracks, tr)
			}
		}
	}
	return tracks
}

func (r *JellyfinRemoteController) fetchItemTypes(
	ctx context.Context,
	session jellyfinMP.SessionInfo,
	itemIDs []string,
) map[string]string {
	result := make(map[string]string, len(itemIDs))
	if session.UserID == "" || session.HTTPClient == nil || len(itemIDs) == 0 {
		return result
	}

	u, err := url.JoinPath(session.BaseURL, "Users", session.UserID, "Items")
	if err != nil {
		return result
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return result
	}
	q := parsed.Query()
	q.Set("Ids", strings.Join(itemIDs, ","))
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return result
	}
	applyJellyfinHeaders(req, session)

	resp, err := session.HTTPClient.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result
	}

	var payload struct {
		Items []struct {
			ID   string `json:"Id"`
			Type string `json:"Type"`
		} `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return result
	}
	for _, item := range payload.Items {
		result[item.ID] = item.Type
	}
	return result
}

func (r *JellyfinRemoteController) handlePlaystateCommand(req jfPlaystateRequest) {
	switch req.Command {
	case "Stop":
		r.pm.Stop()
	case "Pause":
		r.pm.Pause()
	case "Unpause":
		r.pm.Continue()
	case "PlayPause":
		r.pm.PlayPause()
	case "NextTrack":
		r.pm.SeekNext()
	case "PreviousTrack":
		r.pm.SeekBackOrPrevious()
	case "Seek":
		if req.SeekPositionTicks != nil {
			sec := float64(*req.SeekPositionTicks) / float64(jellyfinTicksPerSecond)
			r.pm.SeekSeconds(sec)
		}
	case "FastForward":
		r.pm.SeekBySeconds(10)
	case "Rewind":
		r.pm.SeekBySeconds(-10)
	}
}

func (r *JellyfinRemoteController) handleGeneralCommand(cmd jfGeneralCommand) {
	switch cmd.Name {
	case "SetVolume":
		if volStr, ok := cmd.Arguments["Volume"]; ok {
			if vol, err := strconv.Atoi(volStr); err == nil {
				r.pm.SetVolume(vol)
			}
		}
	case "VolumeUp":
		r.pm.AdjustVolume(5)
	case "VolumeDown":
		r.pm.AdjustVolume(-5)
	case "Mute":
		r.pm.SetMuted(true)
	case "Unmute":
		r.pm.SetMuted(false)
	case "ToggleMute":
		r.pm.ToggleMute()
	case "SetRepeatMode":
		switch cmd.Arguments["RepeatMode"] {
		case "RepeatNone":
			r.pm.SetLoopMode(LoopNone)
		case "RepeatAll":
			r.pm.SetLoopMode(LoopAll)
		case "RepeatOne":
			r.pm.SetLoopMode(LoopOne)
		}
	case "SetShuffleQueue":
		switch cmd.Arguments["ShuffleMode"] {
		case "Shuffle":
			r.pm.SetShuffle(true)
		case "Sorted":
			r.pm.SetShuffle(false)
		}
	case "DisplayMessage":
		if r.OnRemoteMessage != nil {
			r.OnRemoteMessage(cmd.Arguments["Header"], cmd.Arguments["Text"])
		}
	}
}

func (r *JellyfinRemoteController) postCapabilities(ctx context.Context, session jellyfinMP.SessionInfo) error {
	body := map[string]any{
		"PlayableMediaTypes": []string{"Audio"},
		"SupportedCommands": []string{
			"MoveUp", "MoveDown", "MoveLeft", "MoveRight",
			"VolumeUp", "VolumeDown", "Mute", "Unmute", "ToggleMute",
			"SetVolume", "SetRepeatMode", "SetShuffleQueue",
			"Play", "PlayState", "PlayNext", "PlayMediaSource",
			"DisplayMessage",
		},
		"SupportsMediaControl":         true,
		"SupportsPersistentIdentifier": true,
	}
	return postJellyfinJSON(ctx, session, "/Sessions/Capabilities/Full", body)
}

type jfQueueItem struct {
	Id             string `json:"Id"`
	PlaylistItemId string `json:"PlaylistItemId"`
}

func (r *JellyfinRemoteController) buildNowPlayingQueue() ([]jfQueueItem, string) {
	queue := r.pm.GetActivePlayQueue()
	if len(queue) == 0 {
		return nil, ""
	}
	nowIdx := r.pm.NowPlayingIndex()
	start := 0
	end := len(queue)
	if len(queue) > 100 {
		start = max(0, nowIdx-20)
		end = min(len(queue), start+100)
	}
	items := make([]jfQueueItem, 0, end-start)
	var currentPlaylistItemID string
	for i := start; i < end; i++ {
		id := queue[i].Metadata().ID
		plID := fmt.Sprintf("playlistItem%d", i)
		items = append(items, jfQueueItem{Id: id, PlaylistItemId: plID})
		if i == nowIdx {
			currentPlaylistItemID = plID
		}
	}
	return items, currentPlaylistItemID
}

func (r *JellyfinRemoteController) jellyfinRepeatMode() string {
	switch r.pm.GetLoopMode() {
	case LoopAll:
		return "RepeatAll"
	case LoopOne:
		return "RepeatOne"
	default:
		return "RepeatNone"
	}
}

func (r *JellyfinRemoteController) getActiveSession() (jellyfinMP.SessionInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running || !r.hasSession {
		return jellyfinMP.SessionInfo{}, false
	}
	return r.session, true
}

func (r *JellyfinRemoteController) reportPlaybackStart(itemID string) {
	session, ok := r.getActiveSession()
	if !ok || itemID == "" {
		return
	}
	queue, plID := r.buildNowPlayingQueue()
	body := map[string]any{
		"ItemId":          itemID,
		"PositionTicks":   int64(0),
		"IsPaused":        false,
		"IsMuted":         r.pm.IsMuted(),
		"VolumeLevel":     r.pm.Volume(),
		"RepeatMode":      r.jellyfinRepeatMode(),
		"CanSeek":         true,
		"PlayMethod":      "DirectStream",
		"NowPlayingQueue": queue,
		"PlaylistItemId":  plID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = postJellyfinJSON(ctx, session, "/Sessions/Playing", body)
}

func (r *JellyfinRemoteController) reportPlaybackProgress(eventName string) {
	session, ok := r.getActiveSession()
	if !ok {
		return
	}
	np := r.pm.NowPlaying()
	if np == nil || np.Metadata().Type != mediaprovider.MediaItemTypeTrack {
		return
	}
	status := r.pm.PlaybackStatus()
	queue, plID := r.buildNowPlayingQueue()
	body := map[string]any{
		"ItemId":          np.Metadata().ID,
		"PositionTicks":   int64(status.TimePos * float64(jellyfinTicksPerSecond)),
		"IsPaused":        status.State == player.Paused,
		"IsMuted":         r.pm.IsMuted(),
		"VolumeLevel":     r.pm.Volume(),
		"RepeatMode":      r.jellyfinRepeatMode(),
		"CanSeek":         true,
		"PlayMethod":      "DirectStream",
		"EventName":       eventName,
		"NowPlayingQueue": queue,
		"PlaylistItemId":  plID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = postJellyfinJSON(ctx, session, "/Sessions/Playing/Progress", body)
}

func (r *JellyfinRemoteController) reportPlaybackStopped() {
	session, ok := r.getActiveSession()
	if !ok {
		return
	}
	r.mu.Lock()
	itemID := r.lastItemID
	r.lastItemID = ""
	r.mu.Unlock()
	if itemID == "" {
		return
	}
	status := r.pm.PlaybackStatus()
	body := map[string]any{
		"ItemId":        itemID,
		"PositionTicks": int64(status.TimePos * float64(jellyfinTicksPerSecond)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = postJellyfinJSON(ctx, session, "/Sessions/Playing/Stopped", body)
}

func applyJellyfinHeaders(req *http.Request, session jellyfinMP.SessionInfo) {
	if session.Token != "" {
		req.Header.Set("X-Emby-Token", session.Token)
	}
	if session.AuthHeader != "" {
		req.Header.Set("Authorization", session.AuthHeader)
	}
}

func postJellyfinJSON(ctx context.Context, session jellyfinMP.SessionInfo, endpoint string, payload any) error {
	if session.HTTPClient == nil {
		return fmt.Errorf("missing HTTP client")
	}
	u, err := url.JoinPath(session.BaseURL, endpoint)
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	applyJellyfinHeaders(req, session)

	resp, err := session.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s returned HTTP %d", endpoint, resp.StatusCode)
	}
	return nil
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
