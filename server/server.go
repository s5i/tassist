//go:build windows

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/exp"
	"golang.org/x/sync/errgroup"

	_ "embed"
)

func New(storagePath string) (*Server, error) {
	st, err := acc.New(storagePath)
	if err != nil {
		return nil, err
	}

	expCache, err := exp.NewCache()
	if err != nil {
		return nil, err
	}

	s := &Server{
		acc:   st,
		exp:   expCache,
		ready: make(chan bool),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndexHTML)
	mux.HandleFunc("/style.css", s.handleStyleCSS)
	mux.HandleFunc("/main.js", s.handleMainJS)
	mux.HandleFunc("/api/accounts/list", s.handleAccList)
	mux.HandleFunc("/api/accounts/rename", s.handleAccRename)
	mux.HandleFunc("/api/accounts/delete", s.handleAccDelete)
	mux.HandleFunc("/api/accounts/load", s.handleAccLoad)
	mux.HandleFunc("/api/accounts/store", s.handleAccStore)
	mux.HandleFunc("/api/exp/stats", s.handleExpStats)
	mux.HandleFunc("/api/exp/reset", s.handleExpReset)
	s.mux = mux

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.ln = ln
	defer s.ln.Close()

	close(s.ready)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return http.Serve(s.ln, s.mux)
	})
	eg.Go(func() error {
		return s.exp.Run(ctx)
	})

	return eg.Wait()
}

func (s *Server) Ready() <-chan bool {
	return s.ready
}

func (s *Server) Addr() string {
	return s.ln.Addr().String()
}

type Server struct {
	acc   *acc.Storage
	exp   *exp.Cache
	mux   *http.ServeMux
	ln    net.Listener
	ready chan bool
}

func (s *Server) handleIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) handleStyleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(styleCSS)
}

func (s *Server) handleMainJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(mainJS)
}

func (s *Server) handleAccList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.acc.ListRows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var out []entryJSON
	for _, row := range rows {
		out = append(out, entryJSON{ID: row.ID, Name: row.Name})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAccRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.acc.RenameRow(req.ID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAccDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.acc.DeleteRow(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAccLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	row, found, err := s.acc.FindRow(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}

	if err := acc.RegRestore(row.A, row.B, row.C); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAccStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a, b, c, err := acc.RegSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := uuid.New().String()[:8]
	name := req.Name
	if name == "" {
		req.Name = "Unnamed"
	}

	if err := s.acc.AddRow(id, name, a, b, c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entryJSON{ID: id, Name: name})
}

func (s *Server) handleExpReset(w http.ResponseWriter, r *http.Request) {
	s.exp.Reset()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Windows []int `json:"windows"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		req.Windows = []int{60, 600, 1800, 3600}
	}

	stats := map[string]int{}
	if latest, ok := s.exp.Latest(); ok {
		stats["latest"] = latest
	}

	for _, window := range req.Windows {
		delta, ok := s.exp.Delta(time.Duration(window) * time.Second)
		if !ok {
			continue
		}
		stats[fmt.Sprintf("eph%d", window)] = int(float64(time.Hour) / float64(window) * float64(delta))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

type entryJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	//go:embed static/index.html
	indexHTML []byte
	//go:embed static/style.css
	styleCSS []byte
	//go:embed static/main.js
	mainJS []byte
)
