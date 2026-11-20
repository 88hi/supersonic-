package shortcuts

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/supersonic-app/supersonic/backend"
	"github.com/supersonic-app/supersonic/backend/windows"
)

type MusicActionID string

const (
	ActionPlayPause       MusicActionID = "play_pause"
	ActionStop            MusicActionID = "stop"
	ActionNextTrack       MusicActionID = "next_track"
	ActionPreviousTrack   MusicActionID = "previous_track"
	ActionSeekForward     MusicActionID = "seek_forward"
	ActionSeekBackward    MusicActionID = "seek_backward"
	ActionVolumeUp        MusicActionID = "volume_up"
	ActionVolumeDown      MusicActionID = "volume_down"
	ActionToggleMute      MusicActionID = "toggle_mute"
	ActionToggleShuffle   MusicActionID = "toggle_shuffle"
	ActionCycleRepeatMode MusicActionID = "cycle_repeat_mode"
	ActionToggleFavorite  MusicActionID = "toggle_favorite"
	ActionTogglePlayQueue MusicActionID = "toggle_play_queue"
)

type MusicShortcutAction struct {
	ID       MusicActionID
	LabelKey string
	Default  string
	Get      func(*backend.ShortcutsConfig) string
	Set      func(*backend.ShortcutsConfig, string)
}

var defaultShortcuts = backend.DefaultShortcutsConfig()

var MusicShortcutActions = []MusicShortcutAction{
	{
		ID:       ActionPlayPause,
		LabelKey: "Play / Pause",
		Default:  defaultShortcuts.PlayPause,
		Get:      func(c *backend.ShortcutsConfig) string { return c.PlayPause },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.PlayPause = v },
	},
	{
		ID:       ActionStop,
		LabelKey: "Stop playback",
		Default:  defaultShortcuts.Stop,
		Get:      func(c *backend.ShortcutsConfig) string { return c.Stop },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.Stop = v },
	},
	{
		ID:       ActionNextTrack,
		LabelKey: "Next track",
		Default:  defaultShortcuts.NextTrack,
		Get:      func(c *backend.ShortcutsConfig) string { return c.NextTrack },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.NextTrack = v },
	},
	{
		ID:       ActionPreviousTrack,
		LabelKey: "Previous track",
		Default:  defaultShortcuts.PreviousTrack,
		Get:      func(c *backend.ShortcutsConfig) string { return c.PreviousTrack },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.PreviousTrack = v },
	},
	{
		ID:       ActionSeekForward,
		LabelKey: "Seek forward 10s",
		Default:  defaultShortcuts.SeekForward,
		Get:      func(c *backend.ShortcutsConfig) string { return c.SeekForward },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.SeekForward = v },
	},
	{
		ID:       ActionSeekBackward,
		LabelKey: "Seek backward 10s",
		Default:  defaultShortcuts.SeekBackward,
		Get:      func(c *backend.ShortcutsConfig) string { return c.SeekBackward },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.SeekBackward = v },
	},
	{
		ID:       ActionVolumeUp,
		LabelKey: "Volume up",
		Default:  defaultShortcuts.VolumeUp,
		Get:      func(c *backend.ShortcutsConfig) string { return c.VolumeUp },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.VolumeUp = v },
	},
	{
		ID:       ActionVolumeDown,
		LabelKey: "Volume down",
		Default:  defaultShortcuts.VolumeDown,
		Get:      func(c *backend.ShortcutsConfig) string { return c.VolumeDown },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.VolumeDown = v },
	},
	{
		ID:       ActionToggleMute,
		LabelKey: "Mute / Unmute",
		Default:  defaultShortcuts.ToggleMute,
		Get:      func(c *backend.ShortcutsConfig) string { return c.ToggleMute },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.ToggleMute = v },
	},
	{
		ID:       ActionToggleShuffle,
		LabelKey: "Toggle shuffle",
		Default:  defaultShortcuts.ToggleShuffle,
		Get:      func(c *backend.ShortcutsConfig) string { return c.ToggleShuffle },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.ToggleShuffle = v },
	},
	{
		ID:       ActionCycleRepeatMode,
		LabelKey: "Cycle repeat mode",
		Default:  defaultShortcuts.CycleRepeatMode,
		Get:      func(c *backend.ShortcutsConfig) string { return c.CycleRepeatMode },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.CycleRepeatMode = v },
	},
	{
		ID:       ActionToggleFavorite,
		LabelKey: "Favorite current track",
		Default:  defaultShortcuts.ToggleFavorite,
		Get:      func(c *backend.ShortcutsConfig) string { return c.ToggleFavorite },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.ToggleFavorite = v },
	},
	{
		ID:       ActionTogglePlayQueue,
		LabelKey: "Show / Hide play queue",
		Default:  defaultShortcuts.TogglePlayQueue,
		Get:      func(c *backend.ShortcutsConfig) string { return c.TogglePlayQueue },
		Set:      func(c *backend.ShortcutsConfig, v string) { c.TogglePlayQueue = v },
	},
}

var namedKeys = map[string]fyne.KeyName{
	"space":     fyne.KeySpace,
	"return":    fyne.KeyReturn,
	"enter":     fyne.KeyReturn,
	"kp_enter":  fyne.KeyEnter,
	"tab":       fyne.KeyTab,
	"backspace": fyne.KeyBackspace,
	"insert":    fyne.KeyInsert,
	"delete":    fyne.KeyDelete,
	"del":       fyne.KeyDelete,
	"right":     fyne.KeyRight,
	"left":      fyne.KeyLeft,
	"down":      fyne.KeyDown,
	"up":        fyne.KeyUp,
	"pageup":    fyne.KeyPageUp,
	"prior":     fyne.KeyPageUp,
	"pagedown":  fyne.KeyPageDown,
	"next":      fyne.KeyPageDown,
	"home":      fyne.KeyHome,
	"end":       fyne.KeyEnd,
	"f1":        fyne.KeyF1,
	"f2":        fyne.KeyF2,
	"f3":        fyne.KeyF3,
	"f4":        fyne.KeyF4,
	"f5":        fyne.KeyF5,
	"f6":        fyne.KeyF6,
	"f7":        fyne.KeyF7,
	"f8":        fyne.KeyF8,
	"f9":        fyne.KeyF9,
	"f10":       fyne.KeyF10,
	"f11":       fyne.KeyF11,
	"f12":       fyne.KeyF12,
	"0":         fyne.Key0,
	"1":         fyne.Key1,
	"2":         fyne.Key2,
	"3":         fyne.Key3,
	"4":         fyne.Key4,
	"5":         fyne.Key5,
	"6":         fyne.Key6,
	"7":         fyne.Key7,
	"8":         fyne.Key8,
	"9":         fyne.Key9,
	"a":         fyne.KeyA,
	"b":         fyne.KeyB,
	"c":         fyne.KeyC,
	"d":         fyne.KeyD,
	"e":         fyne.KeyE,
	"f":         fyne.KeyF,
	"g":         fyne.KeyG,
	"h":         fyne.KeyH,
	"i":         fyne.KeyI,
	"j":         fyne.KeyJ,
	"k":         fyne.KeyK,
	"l":         fyne.KeyL,
	"m":         fyne.KeyM,
	"n":         fyne.KeyN,
	"o":         fyne.KeyO,
	"p":         fyne.KeyP,
	"q":         fyne.KeyQ,
	"r":         fyne.KeyR,
	"s":         fyne.KeyS,
	"t":         fyne.KeyT,
	"u":         fyne.KeyU,
	"v":         fyne.KeyV,
	"w":         fyne.KeyW,
	"x":         fyne.KeyX,
	"y":         fyne.KeyY,
	"z":         fyne.KeyZ,
	"'":         fyne.KeyApostrophe,
	",":         fyne.KeyComma,
	"-":         fyne.KeyMinus,
	".":         fyne.KeyPeriod,
	"/":         fyne.KeySlash,
	"\\":        fyne.KeyBackslash,
	"[":         fyne.KeyLeftBracket,
	"]":         fyne.KeyRightBracket,
	";":         fyne.KeySemicolon,
	"=":         fyne.KeyEqual,
	"*":         fyne.KeyAsterisk,
	"+":         fyne.KeyPlus,
	"`":         fyne.KeyBackTick,
}

// ModifierForKey returns the KeyModifier corresponding to a modifier KeyName, or 0 if not a modifier.
func ModifierForKey(key fyne.KeyName) fyne.KeyModifier {
	switch key {
	case desktop.KeyControlLeft, desktop.KeyControlRight:
		return fyne.KeyModifierControl
	case desktop.KeyShiftLeft, desktop.KeyShiftRight:
		return fyne.KeyModifierShift
	case desktop.KeyAltLeft, desktop.KeyAltRight:
		return fyne.KeyModifierAlt
	case desktop.KeySuperLeft, desktop.KeySuperRight:
		return fyne.KeyModifierSuper
	default:
		return 0
	}
}

// IsModifierKey reports whether key is a standalone modifier key.
func IsModifierKey(key fyne.KeyName) bool {
	return ModifierForKey(key) != 0
}

// ParseShortcut parses a human-readable shortcut string such as "Space", "Ctrl+Right",
// "Ctrl+Shift+S", or "Ctrl++". Returns ok=false if empty or invalid.
func ParseShortcut(spec string) (fyne.KeyName, fyne.KeyModifier, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", 0, false
	}

	var tokens []string
	// Handle literal "+" as the key (e.g. "+" or "Ctrl++")
	if spec == "+" {
		tokens = []string{"+"}
	} else if strings.HasSuffix(spec, "++") {
		prefix := strings.TrimSuffix(spec, "++")
		if prefix != "" {
			tokens = strings.Split(prefix, "+")
		}
		tokens = append(tokens, "+")
	} else {
		tokens = strings.Split(spec, "+")
	}

	var mod fyne.KeyModifier
	var key fyne.KeyName
	for i, raw := range tokens {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			return "", 0, false
		}
		lower := strings.ToLower(tok)
		isLast := i == len(tokens)-1
		if !isLast {
			switch lower {
			case "ctrl", "control":
				mod |= fyne.KeyModifierControl
				continue
			case "shift":
				mod |= fyne.KeyModifierShift
				continue
			case "alt":
				mod |= fyne.KeyModifierAlt
				continue
			case "super", "meta", "cmd", "win":
				mod |= fyne.KeyModifierSuper
				continue
			default:
				return "", 0, false
			}
		}
		if k, ok := namedKeys[lower]; ok {
			key = k
		} else {
			return "", 0, false
		}
	}
	if key == "" {
		return "", 0, false
	}
	return key, mod, true
}

// FormatShortcut formats a KeyName and KeyModifier into a canonical string.
func FormatShortcut(key fyne.KeyName, mod fyne.KeyModifier) string {
	if key == "" || key == fyne.KeyUnknown {
		return ""
	}
	var parts []string
	if mod&fyne.KeyModifierControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mod&fyne.KeyModifierAlt != 0 {
		parts = append(parts, "Alt")
	}
	if mod&fyne.KeyModifierShift != 0 {
		parts = append(parts, "Shift")
	}
	if mod&fyne.KeyModifierSuper != 0 {
		parts = append(parts, "Super")
	}
	keyDisp := string(key)
	switch key {
	case fyne.KeyPageUp:
		keyDisp = "PageUp"
	case fyne.KeyPageDown:
		keyDisp = "PageDown"
	case fyne.KeyEnter:
		keyDisp = "Enter"
	}
	parts = append(parts, keyDisp)
	return strings.Join(parts, "+")
}

func isFunctionKey(key fyne.KeyName) bool {
	switch key {
	case fyne.KeyF1, fyne.KeyF2, fyne.KeyF3, fyne.KeyF4,
		fyne.KeyF5, fyne.KeyF6, fyne.KeyF7, fyne.KeyF8,
		fyne.KeyF9, fyne.KeyF10, fyne.KeyF11, fyne.KeyF12:
		return true
	default:
		return false
	}
}

// ToWindowsHotkey converts a parsed Fyne key and modifier into Win32 RegisterHotKey
// modifier flags and a virtual-key code. Bare non-function keys (without modifiers)
// return ok=false so they do not hijack normal typing across other applications.
func ToWindowsHotkey(key fyne.KeyName, mod fyne.KeyModifier) (winMod uint32, vk uint32, ok bool) {
	if mod == 0 && !isFunctionKey(key) {
		return 0, 0, false
	}
	vk, found := keyToWindowsVK(key)
	if !found {
		return 0, 0, false
	}
	if mod&fyne.KeyModifierAlt != 0 {
		winMod |= windows.ModAlt
	}
	if mod&fyne.KeyModifierControl != 0 {
		winMod |= windows.ModControl
	}
	if mod&fyne.KeyModifierShift != 0 {
		winMod |= windows.ModShift
	}
	if mod&fyne.KeyModifierSuper != 0 {
		winMod |= windows.ModWin
	}
	return winMod, vk, true
}

func keyToWindowsVK(key fyne.KeyName) (uint32, bool) {
	if len(key) == 1 {
		c := key[0]
		if c >= 'A' && c <= 'Z' {
			return uint32(c), true
		}
		if c >= '0' && c <= '9' {
			return uint32(c), true
		}
	}
	switch key {
	case fyne.KeySpace:
		return 0x20, true
	case fyne.KeyReturn, fyne.KeyEnter:
		return 0x0D, true
	case fyne.KeyTab:
		return 0x09, true
	case fyne.KeyBackspace:
		return 0x08, true
	case fyne.KeyInsert:
		return 0x2D, true
	case fyne.KeyDelete:
		return 0x2E, true
	case fyne.KeyHome:
		return 0x24, true
	case fyne.KeyEnd:
		return 0x23, true
	case fyne.KeyPageUp:
		return 0x21, true
	case fyne.KeyPageDown:
		return 0x22, true
	case fyne.KeyLeft:
		return 0x25, true
	case fyne.KeyUp:
		return 0x26, true
	case fyne.KeyRight:
		return 0x27, true
	case fyne.KeyDown:
		return 0x28, true
	case fyne.KeyF1:
		return 0x70, true
	case fyne.KeyF2:
		return 0x71, true
	case fyne.KeyF3:
		return 0x72, true
	case fyne.KeyF4:
		return 0x73, true
	case fyne.KeyF5:
		return 0x74, true
	case fyne.KeyF6:
		return 0x75, true
	case fyne.KeyF7:
		return 0x76, true
	case fyne.KeyF8:
		return 0x77, true
	case fyne.KeyF9:
		return 0x78, true
	case fyne.KeyF10:
		return 0x79, true
	case fyne.KeyF11:
		return 0x7A, true
	case fyne.KeyF12:
		return 0x7B, true
	case fyne.KeyPlus, fyne.KeyEqual:
		return 0xBB, true
	case fyne.KeyMinus:
		return 0xBD, true
	case fyne.KeyComma:
		return 0xBC, true
	case fyne.KeyPeriod:
		return 0xBE, true
	case fyne.KeySlash:
		return 0xBF, true
	case fyne.KeySemicolon:
		return 0xBA, true
	case fyne.KeyApostrophe:
		return 0xDE, true
	case fyne.KeyLeftBracket:
		return 0xDB, true
	case fyne.KeyRightBracket:
		return 0xDD, true
	case fyne.KeyBackslash:
		return 0xDC, true
	case fyne.KeyBackTick:
		return 0xC0, true
	case fyne.KeyAsterisk:
		return 0x6A, true
	default:
		return 0, false
	}
}
