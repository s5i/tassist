//go:build windows

package startup

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "TAssistant"
)

func Configure(enabled bool, openBrowser bool) error {
	if !enabled {
		return remove()
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	val := fmt.Sprintf(`"%s" -open_browser=%t`, exe, openBrowser)
	return set(val)
}

func set(command string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	return k.SetStringValue(valueName, command)
}

func remove() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	switch err := k.DeleteValue(valueName); err {
	case nil, registry.ErrNotExist:
		return nil
	default:
		return err
	}
}
