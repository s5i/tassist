//go:build windows

package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/s5i/taccount/registry"
	"github.com/s5i/taccount/storage"

	_ "embed"
)

func New(storagePath string) (*Server, error) {
	st, err := storage.New(storagePath)
	if err != nil {
		return nil, err
	}
	return &Server{storage: st}, nil
}

type Server struct {
	storage *storage.YAML
	ln      net.Listener
}

func (s *Server) ListenAndServe() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndexHTML)
	mux.HandleFunc("/style.css", s.handleStyleCSS)
	mux.HandleFunc("/main.js", s.handleMainJS)
	mux.HandleFunc("/api/list", s.handleList)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/delete", s.handleDelete)
	mux.HandleFunc("/api/load", s.handleLoad)
	mux.HandleFunc("/api/store", s.handleStore)

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	return fmt.Sprintf("http://%s", ln.Addr()), nil
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

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.storage.ListRows()
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

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
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

	if err := s.storage.RenameRow(req.ID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := s.storage.DeleteRow(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLoad(w http.ResponseWriter, r *http.Request) {
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

	row, found, err := s.storage.FindRow(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}

	if err := registry.Restore(row.A, row.B, row.C); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStore(w http.ResponseWriter, r *http.Request) {
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

	a, b, c, err := registry.Snapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := uuid.New().String()[:8]
	name := req.Name
	if name == "" {
		req.Name = "Unnamed"
	}

	if err := s.storage.AddRow(id, name, a, b, c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entryJSON{ID: id, Name: name})
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
