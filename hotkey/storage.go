//go:build windows

package hotkey

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ID      string         `yaml:"id" json:"id"`
	Server  string         `yaml:"server" json:"server"`
	Name    string         `yaml:"name" json:"name"`
	Hotkeys map[int]Hotkey `yaml:"hotkeys" json:"hotkeys"`
}

type Storage struct {
	path    string
	entries []*Config
}

func NewStorage(dir string) (*Storage, error) {
	s := &Storage{
		path: filepath.Join(dir, "hotkeys.yaml"),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) List(server string) []*Config {
	var out []*Config
	for _, e := range s.entries {
		if e.Server == server {
			out = append(out, e)
		}
	}
	return out
}

func (s *Storage) Find(id string) (*Config, bool) {
	for _, e := range s.entries {
		if e.ID == id {
			return e, true
		}
	}
	return nil, false
}

func (s *Storage) Add(cfg *Config) error {
	entries := make([]*Config, len(s.entries))
	copy(entries, s.entries)
	entries = append(entries, cfg)
	return s.save(entries)
}

func (s *Storage) Update(cfg *Config) error {
	entries := make([]*Config, len(s.entries))
	copy(entries, s.entries)
	for i, e := range entries {
		if e.ID == cfg.ID {
			entries[i] = cfg
			return s.save(entries)
		}
	}
	return errors.New("config not found")
}

func (s *Storage) Delete(id string) error {
	entries := make([]*Config, 0, len(s.entries))
	for _, e := range s.entries {
		if e.ID == id {
			continue
		}
		entries = append(entries, e)
	}
	return s.save(entries)
}

func (s *Storage) Rename(id, newName string) error {
	entries := make([]*Config, len(s.entries))
	copy(entries, s.entries)
	for _, e := range entries {
		if e.ID == id {
			e.Name = newName
			return s.save(entries)
		}
	}
	return errors.New("config not found")
}

func (s *Storage) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := yaml.Unmarshal(data, &s.entries); err != nil {
		return err
	}
	return nil
}

func (s *Storage) save(entries []*Config) error {
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
	s.entries = entries
	return nil
}
