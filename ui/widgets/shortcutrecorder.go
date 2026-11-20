package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/supersonic-app/supersonic/ui/shortcuts"
)

// ShortcutRecorder is an interactive button widget that captures a keyboard
// shortcut (with optional Ctrl/Alt/Shift/Super modifiers) when clicked.
type ShortcutRecorder struct {
	widget.BaseWidget

	OnChanged func(spec string)

	value     string
	recording bool
	heldMods  fyne.KeyModifier
	button    *widget.Button
}

var (
	_ fyne.Focusable    = (*ShortcutRecorder)(nil)
	_ desktop.Keyable   = (*ShortcutRecorder)(nil)
	_ fyne.Shortcutable = (*ShortcutRecorder)(nil)
)

func NewShortcutRecorder(initial string, onChanged func(string)) *ShortcutRecorder {
	r := &ShortcutRecorder{
		value:     initial,
		OnChanged: onChanged,
	}
	r.button = widget.NewButton("", func() {
		if r.recording {
			r.stopRecording()
		} else {
			r.startRecording()
		}
	})
	r.ExtendBaseWidget(r)
	r.syncButton()
	return r
}

func (r *ShortcutRecorder) Value() string {
	return r.value
}

func (r *ShortcutRecorder) SetValue(spec string) {
	r.value = spec
	r.recording = false
	r.heldMods = 0
	r.syncButton()
}

func (r *ShortcutRecorder) startRecording() {
	r.recording = true
	r.heldMods = 0
	r.syncButton()
	if c := fyne.CurrentApp().Driver().CanvasForObject(r); c != nil {
		c.Focus(r)
	}
}

func (r *ShortcutRecorder) stopRecording() {
	if !r.recording {
		return
	}
	r.recording = false
	r.heldMods = 0
	r.syncButton()
}

func (r *ShortcutRecorder) commitShortcut(spec string) {
	r.value = spec
	r.recording = false
	r.heldMods = 0
	r.syncButton()
	if c := fyne.CurrentApp().Driver().CanvasForObject(r); c != nil {
		c.Unfocus()
	}
	if r.OnChanged != nil {
		r.OnChanged(spec)
	}
}

func (r *ShortcutRecorder) syncButton() {
	if r.recording {
		r.button.SetText(lang.L("Press keys..."))
		r.button.Importance = widget.HighImportance
	} else if r.value == "" {
		r.button.SetText(lang.L("Not set"))
		r.button.Importance = widget.LowImportance
	} else {
		r.button.SetText(r.value)
		r.button.Importance = widget.MediumImportance
	}
	r.button.Refresh()
}

func (r *ShortcutRecorder) MinSize() fyne.Size {
	ms := r.button.MinSize()
	if ms.Width < 150 {
		ms.Width = 150
	}
	return ms
}

func (r *ShortcutRecorder) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.button)
}

func (r *ShortcutRecorder) FocusGained() {}

func (r *ShortcutRecorder) FocusLost() {
	r.stopRecording()
}

func (r *ShortcutRecorder) TypedRune(_ rune) {}

func (r *ShortcutRecorder) TypedKey(e *fyne.KeyEvent) {
	if !r.recording {
		return
	}
	if e.Name == fyne.KeyEscape {
		r.stopRecording()
		return
	}
	if shortcuts.IsModifierKey(e.Name) || e.Name == fyne.KeyUnknown {
		return
	}
	if spec := shortcuts.FormatShortcut(e.Name, r.heldMods); spec != "" {
		r.commitShortcut(spec)
	}
}

func (r *ShortcutRecorder) KeyDown(e *fyne.KeyEvent) {
	if !r.recording {
		return
	}
	if mod := shortcuts.ModifierForKey(e.Name); mod != 0 {
		r.heldMods |= mod
		return
	}
	if e.Name == fyne.KeyEscape {
		r.stopRecording()
		return
	}
	if e.Name == fyne.KeyUnknown {
		return
	}
	if spec := shortcuts.FormatShortcut(e.Name, r.heldMods); spec != "" {
		r.commitShortcut(spec)
	}
}

func (r *ShortcutRecorder) KeyUp(e *fyne.KeyEvent) {
	if mod := shortcuts.ModifierForKey(e.Name); mod != 0 {
		r.heldMods &^= mod
	}
}

func (r *ShortcutRecorder) TypedShortcut(s fyne.Shortcut) {
	if !r.recording {
		return
	}
	if cs, ok := s.(*desktop.CustomShortcut); ok {
		if spec := shortcuts.FormatShortcut(cs.KeyName, cs.Modifier); spec != "" {
			r.commitShortcut(spec)
		}
	}
}
