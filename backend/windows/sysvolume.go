//go:build windows

package windows

/*
#cgo LDFLAGS: -lole32
int win_get_system_volume(int *out_vol, int *out_muted);
int win_set_system_volume(int vol);
int win_set_system_mute(int muted);
*/
import "C"

import (
	"context"
	"errors"
	"time"
)

func SystemVolumeSupported() bool {
	return true
}

func GetSystemVolume() (vol int, muted bool, err error) {
	var cVol, cMuted C.int
	if C.win_get_system_volume(&cVol, &cMuted) != 0 {
		return 0, false, errors.New("failed to query Windows system volume")
	}
	return int(cVol), cMuted != 0, nil
}

func SetSystemVolume(vol int) error {
	if C.win_set_system_volume(C.int(vol)) != 0 {
		return errors.New("failed to set Windows system volume")
	}
	return nil
}

func SetSystemMute(muted bool) error {
	m := C.int(0)
	if muted {
		m = 1
	}
	if C.win_set_system_mute(m) != 0 {
		return errors.New("failed to set Windows system mute")
	}
	return nil
}

// StartSystemVolumeWatcher polls the Windows default audio render endpoint
// and invokes onChange whenever the master volume or mute state changes.
func StartSystemVolumeWatcher(ctx context.Context, onChange func(vol int, muted bool)) {
	go func() {
		lastVol := -1
		lastMuted := false
		if v, m, err := GetSystemVolume(); err == nil {
			lastVol = v
			lastMuted = m
		}

		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				v, m, err := GetSystemVolume()
				if err != nil {
					continue
				}
				if v != lastVol || m != lastMuted {
					lastVol = v
					lastMuted = m
					if onChange != nil {
						onChange(v, m)
					}
				}
			}
		}
	}()
}
