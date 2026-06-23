//go:build windows

package hotkey

import (
	"fmt"
	"strings"
	"sync"
	"syscall"

	"github.com/winlabs/gowin32"
	"golang.org/x/sys/windows"
)

var (
	user32          = windows.NewLazyDLL("user32.dll")
	procEnumWindows = user32.NewProc("EnumWindows")
)

var windowFinder = func() *winFinder {
	wf := &winFinder{}
	wf.cb = windows.NewCallback(func(h windows.Handle, _ uintptr) uintptr {
		text, err := gowin32.GetWindowText(syscall.Handle(h))
		if err != nil {
			return 1
		}
		if strings.TrimSpace(text) == wf.title {
			wf.found = true
			return 0
		}
		return 1
	})
	return wf
}()

type winFinder struct {
	mu    sync.Mutex
	cb    uintptr
	title string
	found bool
}

func IsClientRunning(windowTitle string) bool {
	windowFinder.mu.Lock()
	defer windowFinder.mu.Unlock()

	windowFinder.title = windowTitle
	windowFinder.found = false
	procEnumWindows.Call(windowFinder.cb, 0)
	return windowFinder.found
}

func ErrClientRunning(windowTitle string) error {
	if IsClientRunning(windowTitle) {
		return fmt.Errorf("close the Tibia client first")
	}
	return nil
}
