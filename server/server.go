//go:build windows

package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/s5i/taccount/registry"
	"github.com/s5i/taccount/storage"
)

type Server struct {
	mu       sync.Mutex
	entries  *[]storage.Entry
	yamlPath string
	ln       net.Listener
}

func New(entries *[]storage.Entry, yamlPath string) *Server {
	return &Server{entries: entries, yamlPath: yamlPath}
}

func (s *Server) ListenAndServe() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/list", s.handleList)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/delete", s.handleDelete)
	mux.HandleFunc("/api/load", s.handleLoad)
	mux.HandleFunc("/api/save", s.handleSave)

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	return fmt.Sprintf("http://%s", ln.Addr()), nil
}

func (s *Server) save() {
	if err := storage.Save(s.yamlPath, *s.entries); err != nil {
		log.Printf("Failed to save: %v", err)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

type entryJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]entryJSON, len(*s.entries))
	for i, e := range *s.entries {
		out[i] = entryJSON{ID: e.ID, Name: e.HumanName}
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

	s.mu.Lock()
	storage.RenameEntry(s.entries, req.ID, req.Name)
	s.save()
	s.mu.Unlock()

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

	s.mu.Lock()
	storage.DeleteEntry(s.entries, req.ID)
	s.save()
	s.mu.Unlock()

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

	s.mu.Lock()
	var found *storage.Entry
	for i, e := range *s.entries {
		if e.ID == req.ID {
			found = &(*s.entries)[i]
			break
		}
	}
	s.mu.Unlock()

	if found == nil {
		http.Error(w, "entry not found", http.StatusNotFound)
		return
	}

	if err := registry.Restore([]byte(found.A), []byte(found.B), string(found.C)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
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
	if req.Name == "" {
		req.Name = "Unnamed"
	}

	a, b, c, err := registry.Snapshot()
	if err != nil {
		http.Error(w, fmt.Sprintf("registry snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()

	e := storage.Entry{
		ID:        uuid.New().String()[:8],
		HumanName: req.Name,
		A:         storage.Bytes(a),
		B:         storage.Bytes(b),
		C:         storage.String(c),
	}
	storage.AddEntry(s.entries, e)
	s.save()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entryJSON{ID: e.ID, Name: e.HumanName})
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Tibiantis Account Switcher</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'Segoe UI', system-ui, sans-serif;
    background: #1a1a2e;
    color: #e0e0e0;
    display: flex;
    justify-content: center;
    padding: 40px 16px;
    min-height: 100vh;
  }
  .container { width: 100%; max-width: 420px; }
  h1 {
    font-size: 1.3rem;
    font-weight: 600;
    color: #e94560;
    margin-bottom: 20px;
    text-align: center;
    letter-spacing: .5px;
  }
  #list { display: flex; flex-direction: column; gap: 8px; }
  .entry {
    display: flex;
    align-items: center;
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 10px 14px;
    cursor: pointer;
    transition: background .15s, border-color .15s;
  }
  .entry:hover { background: #1a2a50; border-color: #e94560; }
  .entry .name {
    flex: 1;
    font-size: .95rem;
    outline: none;
    background: transparent;
    color: #e0e0e0;
    border: none;
    border-bottom: 1px solid transparent;
    padding: 2px 0;
    font-family: inherit;
  }
  .entry .name:focus { border-bottom-color: #e94560; }
  .btn {
    background: none;
    border: 1px solid #0f3460;
    color: #aaa;
    border-radius: 6px;
    padding: 4px 10px;
    margin-left: 6px;
    cursor: pointer;
    font-size: .8rem;
    transition: color .15s, border-color .15s;
  }
  .btn:hover { color: #e94560; border-color: #e94560; }
  .btn.restore { color: #53d769; border-color: #2e5e3a; }
  .btn.restore:hover { color: #7fff9a; border-color: #53d769; }
  .btn.delete:hover { color: #ff4444; border-color: #ff4444; }
  .empty {
    text-align: center;
    color: #555;
    margin-top: 40px;
    font-style: italic;
  }
  .toast {
    position: fixed; bottom: 24px; left: 50%; transform: translateX(-50%);
    background: #0f3460; color: #e0e0e0; padding: 10px 24px;
    border-radius: 8px; font-size: .85rem; opacity: 0;
    transition: opacity .3s; pointer-events: none; z-index: 10;
  }
  .toast.show { opacity: 1; }
</style>
</head>
<body>
<div class="container">
  <h1>Tibiantis Account Switcher</h1>
  <div id="list"></div>

  <div id="toast" class="toast"></div>
</div>
<script>
const listEl = document.getElementById('list');
const toastEl = document.getElementById('toast');
let toastTimer;

function toast(msg) {
  toastEl.textContent = msg;
  toastEl.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toastEl.classList.remove('show'), 2000);
}

async function load() {
  const resp = await fetch('/api/list');
  const entries = await resp.json();
  listEl.innerHTML = '';

  for (const e of entries) {
    const div = document.createElement('div');
    div.className = 'entry';
    div.innerHTML =
      '<input class="name" value="' + esc(e.name) + '" data-id="' + esc(e.id) + '" />' +
      '<button class="btn load" data-id="' + esc(e.id) + '">Load</button>' +
      '<button class="btn delete" data-id="' + esc(e.id) + '">Delete</button>';
    listEl.appendChild(div);
  }

  const div = document.createElement('div');
  div.id = 'snapshot'; 
  div.className = 'entry';
  div.innerHTML =
    '<input class="name" id="new-name" placeholder="New account name" />' +
    '<button class="btn save">Save</button>'
  listEl.appendChild(div);
}

function esc(s) { const d = document.createElement('div'); d.textContent = s; return d.innerHTML.replace(/"/g, '&quot;'); }

listEl.addEventListener('click', async (ev) => {
  const btn = ev.target.closest('.btn');
  if (!btn) return;
  const id = btn.dataset.id;

  if (btn.classList.contains('load')) {
    const r = await fetch('/api/load', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({id}) });
    toast(r.ok ? 'Loaded!' : 'Error: ' + await r.text());
  } else if (btn.classList.contains('delete')) {
    if (!confirm('Delete this entry?')) return;
    const r = await fetch('/api/delete', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({id}) });
    if (r.ok) { toast('Deleted'); load(); } else { toast('Error: ' + await r.text()); }
  } else if (btn.classList.contains('save')) {
    const nameInput = document.getElementById('new-name');
    const name = nameInput.value.trim() || 'Unnamed';
    const r = await fetch('/api/save', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({name}) });
    if (r.ok) { nameInput.value = ''; toast('Saved!'); load(); } else { toast('Error: ' + await r.text()); }
  }
});

let renameTimer;
listEl.addEventListener('input', (ev) => {
  if (!ev.target.classList.contains('name')) return;
  clearTimeout(renameTimer);
  const id = ev.target.dataset.id;
  const name = ev.target.value;
  renameTimer = setTimeout(async () => {
    await fetch('/api/rename', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({id, name}) });
  }, 500);
});

// Double-click entry row to restore.
listEl.addEventListener('dblclick', async (ev) => {
  const entry = ev.target.closest('.entry');
  if (!entry || ev.target.classList.contains('btn')) return;
  const id = entry.querySelector('.name').dataset.id;
  const r = await fetch('/api/load', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({id}) });
  toast(r.ok ? 'Loaded!' : 'Error: ' + await r.text());
});

load();
</script>
</body>
</html>`
