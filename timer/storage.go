//go:build windows

package timer

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"gopkg.in/yaml.v3"
)

type Storage struct {
	path   string
	timers []*Timer
}

func NewStorage(dir string) (*Storage, error) {
	s := &Storage{
		path: filepath.Join(dir, "timers.yaml"),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) Find(id string) (*Timer, bool, error) {
	for _, e := range s.timers {
		if id == e.ID {
			return e, true, nil
		}
	}

	return nil, false, nil
}

func (s *Storage) List() ([]*Timer, error) {
	return slices.Clone(s.timers), nil
}

func (s *Storage) AddOrUpdate(t *Timer) error {
	entries := slices.Clone(s.timers)
	for i, e := range entries {
		if e.ID == t.ID {
			entries[i] = t
			return s.save(entries)
		}
	}

	entries = append(entries, t)
	return s.save(entries)
}

func (s *Storage) Delete(id string) error {
	entries := slices.Clone(s.timers)
	for i, e := range entries {
		if id == e.ID {
			entries[i] = entries[len(s.timers)-1]
			entries = entries[:len(s.timers)-1]
			return s.save(entries)
		}
	}

	return nil
}

func (s *Storage) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if err := yaml.Unmarshal(data, &s.timers); err != nil {
		return err
	}

	return nil
}

func (s *Storage) save(entries []*Timer) error {
	data, err := yaml.Marshal(entries)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return err
	}

	s.timers = entries
	return nil
}

type Timer struct {
	ID      string        `yaml:"id"`
	Name    string        `yaml:"name"`
	Period  time.Duration `yaml:"period"`
	Active  bool          `yaml:"active"`
	Loop    bool          `yaml:"loop"`
	Sound   bool          `yaml:"sound"`
	Started time.Time     `yaml:"started"`
	Acked   time.Time     `yaml:"acked"`
}
