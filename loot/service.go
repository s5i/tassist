//go:build windows

package loot

import (
	"bytes"
	"fmt"
	"image"
	"sort"
	"sync"

	"github.com/s5i/tloot/process"
	"golang.org/x/sync/errgroup"

	_ "image/png"
)

type Item struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Value    int    `json:"value"`
	Market   bool   `json:"market"`
	Category string `json:"category"`
}

type Service struct {
	items   map[int]process.ItemConfig
	sprites process.SpriteMap
}

func NewService() *Service {
	return &Service{
		items:   process.LoadItems(),
		sprites: process.LoadSprites(),
	}
}

func (s *Service) Items() []Item {
	ret := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		ret = append(ret, itemToAPI(item))
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Category != ret[j].Category {
			return ret[i].Category < ret[j].Category
		}
		return ret[i].Name < ret[j].Name
	})
	return ret
}

func (s *Service) ProcessPNG(data []byte) (map[int]int, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	proc := process.NewCallbackProcessor(img, s.items, s.sprites)
	counts := map[int]int{}
	var mu sync.Mutex

	var g errgroup.Group
	for id, item := range s.items {
		if item.ForceDisabled {
			continue
		}
		id := id
		g.Go(func() error {
			res, err := proc.Process(id)
			if err != nil {
				return fmt.Errorf("process item %d: %w", id, err)
			}
			if res.Count > 0 {
				mu.Lock()
				counts[id] = res.Count
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return counts, nil
}

func itemToAPI(item process.ItemConfig) Item {
	value := item.Value
	market := false
	if value < 0 {
		value = -value
		market = true
	}
	return Item{
		ID:       item.ID,
		Name:     item.Name,
		Value:    value,
		Market:   market,
		Category: item.Category,
	}
}
