package shortcuts

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/supersonic-app/supersonic/backend"
)

func TestParseAndFormatShortcut(t *testing.T) {
	tests := []struct {
		spec    string
		wantKey fyne.KeyName
		wantMod fyne.KeyModifier
		wantOK  bool
		wantFmt string
	}{
		{"Space", fyne.KeySpace, 0, true, "Space"},
		{"Ctrl+Right", fyne.KeyRight, fyne.KeyModifierControl, true, "Ctrl+Right"},
		{"ctrl+shift+s", fyne.KeyS, fyne.KeyModifierControl | fyne.KeyModifierShift, true, "Ctrl+Shift+S"},
		{"Alt+M", fyne.KeyM, fyne.KeyModifierAlt, true, "Alt+M"},
		{"+", fyne.KeyPlus, 0, true, "+"},
		{"Ctrl++", fyne.KeyPlus, fyne.KeyModifierControl, true, "Ctrl++"},
		{"F8", fyne.KeyF8, 0, true, "F8"},
		{"", "", 0, false, ""},
		{"InvalidKey+Foo", "", 0, false, ""},
	}

	for _, tc := range tests {
		k, m, ok := ParseShortcut(tc.spec)
		if ok != tc.wantOK {
			t.Errorf("ParseShortcut(%q) ok = %v, want %v", tc.spec, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if k != tc.wantKey || m != tc.wantMod {
			t.Errorf("ParseShortcut(%q) = (%q, %v), want (%q, %v)", tc.spec, k, m, tc.wantKey, tc.wantMod)
		}
		if got := FormatShortcut(k, m); got != tc.wantFmt {
			t.Errorf("FormatShortcut(%q, %v) = %q, want %q", k, m, got, tc.wantFmt)
		}
	}
}

func TestDefaultMusicShortcutsValid(t *testing.T) {
	cfg := backend.DefaultShortcutsConfig()
	for _, action := range MusicShortcutActions {
		spec := action.Get(&cfg)
		k, m, ok := ParseShortcut(spec)
		if !ok {
			t.Errorf("default shortcut for %s (%q) failed to parse", action.ID, spec)
			continue
		}
		if _, _, ok := ToWindowsHotkey(k, m); !ok {
			t.Errorf("default shortcut for %s (%q) should be globally registerable on Windows", action.ID, spec)
		}
	}
}

func TestToWindowsHotkey(t *testing.T) {
	if _, _, ok := ToWindowsHotkey(fyne.KeySpace, 0); ok {
		t.Error("bare Space should not register as global Windows hotkey")
	}
	if _, _, ok := ToWindowsHotkey(fyne.KeyM, 0); ok {
		t.Error("bare M should not register as global Windows hotkey")
	}
	if _, vk, ok := ToWindowsHotkey(fyne.KeyF8, 0); !ok || vk != 0x77 {
		t.Errorf("F8 global hotkey = (vk=0x%x, ok=%v), want (0x77, true)", vk, ok)
	}
	if mod, vk, ok := ToWindowsHotkey(fyne.KeyRight, fyne.KeyModifierControl|fyne.KeyModifierAlt); !ok || mod == 0 || vk != 0x27 {
		t.Errorf("Ctrl+Alt+Right = (mod=0x%x, vk=0x%x, ok=%v)", mod, vk, ok)
	}
}
