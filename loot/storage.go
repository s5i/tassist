//go:build windows

package loot

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Storage struct {
	path   string
	prices map[int]int
}

type storedPrices struct {
	Prices map[int]int `yaml:"prices"`
}

func NewStorage(dir string) (*Storage, error) {
	s := &Storage{
		path:   filepath.Join(dir, "loot_prices.yaml"),
		prices: map[int]int{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) Prices() map[int]int {
	ret := make(map[int]int, len(s.prices))
	for k, v := range s.prices {
		ret[k] = v
	}
	return ret
}

func (s *Storage) SetPrices(prices map[int]int) error {
	next := make(map[int]int, len(s.prices))
	for k, v := range s.prices {
		next[k] = v
	}
	for k, v := range prices {
		if v <= 0 {
			delete(next, k)
		} else {
			next[k] = v
		}
	}
	return s.save(next)
}

func (s *Storage) Reset() error {
	return s.save(map[int]int{})
}

func (s *Storage) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	stored := storedPrices{}
	if err := yaml.Unmarshal(data, &stored); err != nil {
		return err
	}
	if stored.Prices != nil {
		s.prices = stored.Prices
	}
	return nil
}

func (s *Storage) save(prices map[int]int) error {
	data, err := yaml.Marshal(storedPrices{Prices: prices})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return err
	}
	s.prices = prices
	return nil
}
