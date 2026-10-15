//go:build windows

package instance

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

const stillActive = 259 // STILL_ACTIVE on Windows

const pidFileName = "tassist.pid"

// Acquire tries to become the sole instance. If another instance is already
// running, alreadyRunning is true and release is nil.
func Acquire(dir string) (release func(), alreadyRunning bool, err error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, false, err
	}

	path := filepath.Join(dir, pidFileName)
	if pid, err := readPID(path); err == nil && pid > 0 && isRunningInstance(pid) {
		return nil, true, nil
	}

	_ = os.Remove(path)

	pid := os.Getpid()
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return nil, false, err
	}

	return func() { _ = os.Remove(path) }, false, nil
}

func readPID(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(bytes.TrimSpace(b)))
}

func isRunningInstance(pid int) bool {
	if !processAlive(pid) {
		return false
	}

	ours, err := os.Executable()
	if err != nil {
		return false
	}

	theirs, err := processImageName(pid)
	if err != nil {
		return false
	}

	return sameExecutable(ours, theirs)
}

func processAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	return code == stillActive
}

func processImageName(pid int) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	buf := make([]uint16, 32*1024)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func sameExecutable(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
