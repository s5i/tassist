//go:build windows

package registry

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// The unescaped tab is intentional.
const regPath = "SOFTWARE\tibiantis\\Credentials"

func Snapshot() (a, b, c []byte, err error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	a, _, err = k.GetBinaryValue("A")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read A: %w", err)
	}

	b, _, err = k.GetBinaryValue("B")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read B: %w", err)
	}

	cStr, _, err := k.GetStringValue("C")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read C: %w", err)
	}

	return a, b, []byte(cStr), nil
}

func Restore(a, b, c []byte) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetBinaryValue("A", a); err != nil {
		return fmt.Errorf("write A: %w", err)
	}
	if err := k.SetBinaryValue("B", b); err != nil {
		return fmt.Errorf("write B: %w", err)
	}
	if err := k.SetStringValue("C", string(c)); err != nil {
		return fmt.Errorf("write C: %w", err)
	}
	return nil
}
