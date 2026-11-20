//go:build !windows

package windows

const (
	ModAlt     uint32 = 0x0001
	ModControl uint32 = 0x0002
	ModShift   uint32 = 0x0004
	ModWin     uint32 = 0x0008
)

type HotkeyBinding struct {
	ID        int
	Modifiers uint32
	VK        uint32
	Handler   func()
}

type GlobalHotkeyManager struct{}

func GlobalHotkeysSupported() bool {
	return false
}

func NewGlobalHotkeyManager() *GlobalHotkeyManager {
	return &GlobalHotkeyManager{}
}

func (m *GlobalHotkeyManager) SetBindings(_ []HotkeyBinding) map[int]bool {
	return nil
}

func (m *GlobalHotkeyManager) Clear() {}

func (m *GlobalHotkeyManager) Close() {}
