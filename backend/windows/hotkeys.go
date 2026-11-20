//go:build windows

package windows

import (
	"log"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ModAlt     uint32 = 0x0001
	ModControl uint32 = 0x0002
	ModShift   uint32 = 0x0004
	ModWin     uint32 = 0x0008

	wmHotkey    = 0x0312
	wmQuit      = 0x0012
	wmAppReload = 0x8001 // WM_APP + 1
	pmNoRemove  = 0x0000
)

var (
	user32DLL             = windows.NewLazySystemDLL("user32.dll")
	kernel32DLL           = windows.NewLazySystemDLL("kernel32.dll")
	procRegisterHotKey    = user32DLL.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32DLL.NewProc("UnregisterHotKey")
	procGetMessageW       = user32DLL.NewProc("GetMessageW")
	procPeekMessageW      = user32DLL.NewProc("PeekMessageW")
	procPostThreadMessage = user32DLL.NewProc("PostThreadMessageW")
	procGetCurrentThread  = kernel32DLL.NewProc("GetCurrentThreadId")
)

type winMSG struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type HotkeyBinding struct {
	ID        int
	Modifiers uint32
	VK        uint32
	Handler   func()
}

type reloadRequest struct {
	bindings []HotkeyBinding
	resultCh chan map[int]bool
}

// GlobalHotkeyManager registers and dispatches Win32 global hotkeys on a
// dedicated message-loop OS thread so shortcuts work even when unfocused or minimized.
type GlobalHotkeyManager struct {
	threadID uint32
	ready    chan struct{}

	mu         sync.Mutex
	closed     bool
	pendingReq *reloadRequest
}

func GlobalHotkeysSupported() bool {
	return true
}

func NewGlobalHotkeyManager() *GlobalHotkeyManager {
	m := &GlobalHotkeyManager{
		ready: make(chan struct{}),
	}
	go m.loop()
	<-m.ready
	return m
}

func (m *GlobalHotkeyManager) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tid, _, _ := procGetCurrentThread.Call()
	m.threadID = uint32(tid)

	// Ensure a Win32 message queue exists for this thread before signaling ready.
	var msg winMSG
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmNoRemove)
	close(m.ready)

	registeredIDs := make([]int, 0)
	handlers := make(map[int]func())

	unregisterAll := func() {
		for _, id := range registeredIDs {
			procUnregisterHotKey.Call(0, uintptr(id))
		}
		registeredIDs = registeredIDs[:0]
		handlers = make(map[int]func())
	}

	defer unregisterAll()

	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		switch msg.Message {
		case wmHotkey:
			id := int(msg.WParam)
			if h := handlers[id]; h != nil {
				go h()
			}
		case wmAppReload:
			m.mu.Lock()
			req := m.pendingReq
			m.pendingReq = nil
			m.mu.Unlock()
			if req == nil {
				continue
			}

			unregisterAll()
			result := make(map[int]bool, len(req.bindings))
			for _, b := range req.bindings {
				r, _, err := procRegisterHotKey.Call(0, uintptr(b.ID), uintptr(b.Modifiers), uintptr(b.VK))
				if r != 0 {
					registeredIDs = append(registeredIDs, b.ID)
					handlers[b.ID] = b.Handler
					result[b.ID] = true
				} else {
					log.Printf("Global hotkey register failed (id=%d mod=0x%x vk=0x%x): %v", b.ID, b.Modifiers, b.VK, err)
					result[b.ID] = false
				}
			}
			req.resultCh <- result
		}
	}
}

// SetBindings replaces all currently registered global hotkeys and returns a map
// indicating which binding IDs were successfully registered.
func (m *GlobalHotkeyManager) SetBindings(bindings []HotkeyBinding) map[int]bool {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	req := &reloadRequest{
		bindings: bindings,
		resultCh: make(chan map[int]bool, 1),
	}
	m.pendingReq = req
	m.mu.Unlock()

	r, _, _ := procPostThreadMessage.Call(uintptr(m.threadID), wmAppReload, 0, 0)
	if r == 0 {
		return nil
	}
	return <-req.resultCh
}

func (m *GlobalHotkeyManager) Clear() {
	m.SetBindings(nil)
}

func (m *GlobalHotkeyManager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	m.mu.Unlock()

	procPostThreadMessage.Call(uintptr(m.threadID), wmQuit, 0, 0)
}
