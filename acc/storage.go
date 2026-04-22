//go:build windows

package acc

import (
	"encoding/base64"
	"os"

	"gopkg.in/yaml.v3"
)

type Storage struct {
	path    string
	entries []*row
}

func New(path string) (*Storage, error) {
	s := &Storage{path: path}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (y *Storage) FindRow(id string) (*row, bool, error) {
	for _, e := range y.entries {
		if id == e.ID {
			return e, true, nil
		}
	}

	return nil, false, nil
}

func (y *Storage) ListRows() ([]*row, error) {
	rows := make([]*row, len(y.entries))
	copy(rows, y.entries)

	return rows, nil
}

func (y *Storage) AddRow(id, name string, a, b, c []byte) error {
	rows := make([]*row, len(y.entries))
	copy(rows, y.entries)

	rows = append(rows, &row{id, name, a, b, c})

	if err := y.save(rows); err != nil {
		return err
	}

	return nil
}

func (y *Storage) DeleteRow(id string) error {
	rows := make([]*row, 0, len(y.entries))

	for _, e := range y.entries {
		if id == e.ID {
			continue
		}
		rows = append(rows, e)
	}

	if err := y.save(rows); err != nil {
		return err
	}

	return nil
}

func (y *Storage) RenameRow(id, newName string) error {
	rows := make([]*row, len(y.entries))
	copy(rows, y.entries)

	for _, e := range rows {
		if id == e.ID {
			e.Name = newName
		}
	}

	if err := y.save(rows); err != nil {
		return err
	}

	return nil
}

func (y *Storage) load() error {
	data, err := os.ReadFile(y.path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &y.entries); err != nil {
		return err
	}

	return nil
}

func (y *Storage) save(entries []*row) error {
	data, err := yaml.Marshal(entries)
	if err != nil {
		return err
	}

	if err := os.WriteFile(y.path, data, 0644); err != nil {
		return err
	}

	y.entries = entries
	return nil
}

type row struct {
	ID   string      `yaml:"id"`
	Name string      `yaml:"name"`
	A    base64Bytes `yaml:"a"`
	B    base64Bytes `yaml:"b"`
	C    base64Bytes `yaml:"c"`
}

type base64Bytes []byte

func (b base64Bytes) MarshalYAML() (any, error) {
	return base64.StdEncoding.EncodeToString(b), nil
}

func (b *base64Bytes) UnmarshalYAML(node *yaml.Node) error {
	data, err := base64.StdEncoding.DecodeString(node.Value)
	if err != nil {
		return err
	}
	*b = data
	return nil
}
