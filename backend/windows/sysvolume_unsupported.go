//go:build !windows

package windows

import (
	"context"
	"errors"
)

func SystemVolumeSupported() bool {
	return false
}

func GetSystemVolume() (int, bool, error) {
	return 0, false, errors.New("system volume sync unsupported on this OS")
}

func SetSystemVolume(int) error {
	return errors.New("system volume sync unsupported on this OS")
}

func SetSystemMute(bool) error {
	return errors.New("system volume sync unsupported on this OS")
}

func StartSystemVolumeWatcher(context.Context, func(int, bool)) {}
