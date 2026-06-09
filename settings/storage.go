package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	UpdaterModeAutomatic = "Automatic"
	UpdaterModeManual    = "Manual"
	UpdaterModeIgnore    = "Ignore"
)

type Storage struct {
	file   string
	preset *Preset
	stored StoredSettings
}

type StoredSettings struct {
	Preset      string `yaml:"preset"`
	UpdaterMode string `yaml:"updater_mode,omitempty"`
	SkipVersion string `yaml:"skip_version,omitempty"`
}

func New(dir string) (*Storage, error) {
	s := &Storage{
		file: filepath.Join(dir, "settings.yaml"),
	}

	data, err := os.ReadFile(s.file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.stored = StoredSettings{Preset: Ancestra}
			s.preset = Presets[Ancestra]
			return s, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, &s.stored); err != nil {
		return nil, err
	}

	p, ok := Presets[s.stored.Preset]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q", s.stored.Preset)
	}
	s.preset = p

	return s, nil
}

func (s *Storage) Preset() *Preset {
	return s.preset
}

func (s *Storage) SwitchPreset(id string) error {
	p, ok := Presets[id]
	if !ok {
		return fmt.Errorf("unknown preset %q", id)
	}

	s.stored.Preset = id
	if err := s.save(); err != nil {
		return err
	}

	s.preset = p

	return nil
}

func (s *Storage) UpdaterMode() string {
	switch s.stored.UpdaterMode {
	case UpdaterModeAutomatic, UpdaterModeManual, UpdaterModeIgnore:
		return s.stored.UpdaterMode
	default:
		return UpdaterModeManual
	}
}

func (s *Storage) SkipVersion() string {
	return s.stored.SkipVersion
}

func (s *Storage) SetUpdaterMode(mode string) error {
	switch mode {
	case UpdaterModeAutomatic, UpdaterModeManual, UpdaterModeIgnore:
		s.stored.UpdaterMode = mode
		return s.save()
	default:
		return fmt.Errorf("unknown updater mode %q", mode)
	}
}

func (s *Storage) SetSkipVersion(v string) error {
	s.stored.SkipVersion = v
	return s.save()
}

func (s *Storage) save() error {
	data, err := yaml.Marshal(&s.stored)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.file), 0755); err != nil {
		return err
	}

	return os.WriteFile(s.file, data, 0644)
}
