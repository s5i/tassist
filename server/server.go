//go:build windows

package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/exp"
	"github.com/s5i/tassist/hotkey"
	"github.com/s5i/tassist/loot"
	"github.com/s5i/tassist/online"
	"github.com/s5i/tassist/ping"
	"github.com/s5i/tassist/settings"
	"github.com/s5i/tassist/timer"

	"golang.org/x/sync/errgroup"

	_ "embed"
)

var ErrRestart = fmt.Errorf("server is restarting...")

func New(dataDir, tmpDir string, accStorage *acc.Storage, expCache *exp.Cache, pinger *ping.Pinger, online *online.Online, version string, stStorage *settings.Storage, tmStorage *timer.Storage, lootSvc *loot.Service, lootStorage *loot.Storage, hkStorage *hotkey.Storage) (*Server, error) {
	s := &Server{
		dataDir:              dataDir,
		tmpDir:               tmpDir,
		acc:                  accStorage,
		exp:                  expCache,
		pinger:               pinger,
		online:               online,
		version:              version,
		stStorage:            stStorage,
		tmStorage:            tmStorage,
		loot:                 lootSvc,
		lootStorage:          lootStorage,
		hkStorage:            hkStorage,
		keepalive:            time.Now(),
		keepaliveCheckPeriod: 2 * time.Second,
		keepaliveFails:       3,
		keepaliveTimeout:     15 * time.Second,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/beep.mp3", s.handleBeep)
	mux.HandleFunc("/", s.handleStatic)
	mux.HandleFunc("/api/keepalive", s.handleKeepalive)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/accounts/list", s.handleAccList)
	mux.HandleFunc("/api/accounts/rename", s.handleAccRename)
	mux.HandleFunc("/api/accounts/delete", s.handleAccDelete)
	mux.HandleFunc("/api/accounts/load", s.handleAccLoad)
	mux.HandleFunc("/api/accounts/store", s.handleAccStore)
	mux.HandleFunc("/api/exp/stats", s.handleExpStats)
	mux.HandleFunc("/api/exp/start", s.handleExpStart)
	mux.HandleFunc("/api/exp/stop", s.handleExpStop)
	mux.HandleFunc("/api/exp/pause", s.handleExpPause)
	mux.HandleFunc("/api/exp/unpause", s.handleExpUnpause)
	mux.HandleFunc("/api/exp/reset", s.handleExpReset)
	mux.HandleFunc("/api/world/ping", s.handleWorldPing)
	mux.HandleFunc("/api/world/online", s.handleWorldOnline)
	mux.HandleFunc("/api/preset/switch", s.handlePresetSwitch)
	mux.HandleFunc("/api/preset/list", s.handlePresetList)
	mux.HandleFunc("/api/update/settings", s.handleUpdateSettings)
	mux.HandleFunc("/api/update/check", s.handleUpdateCheck)
	mux.HandleFunc("/api/update/execute", s.handleUpdateExecute)
	mux.HandleFunc("/api/update/skip", s.handleUpdateSkip)
	mux.HandleFunc("/api/timers/add", s.handleTimerAdd)
	mux.HandleFunc("/api/timers/start", s.handleTimerStart)
	mux.HandleFunc("/api/timers/stop", s.handleTimerStop)
	mux.HandleFunc("/api/timers/ack", s.handleTimerAck)
	mux.HandleFunc("/api/timers/loop", s.handleTimerLoop)
	mux.HandleFunc("/api/timers/sound", s.handleTimerSound)
	mux.HandleFunc("/api/timers/autoack", s.handleTimerAutoAck)
	mux.HandleFunc("/api/timers/remove", s.handleTimerRemove)
	mux.HandleFunc("/api/timers/list", s.handleTimerList)
	mux.HandleFunc("/api/loot/items", s.handleLootItems)
	mux.HandleFunc("/api/loot/prices", s.handleLootPrices)
	mux.HandleFunc("/api/loot/process", s.handleLootProcess)
	mux.HandleFunc("/api/hotkeys/list", s.handleHotkeysList)
	mux.HandleFunc("/api/hotkeys/store", s.handleHotkeysStore)
	mux.HandleFunc("/api/hotkeys/load", s.handleHotkeysLoad)
	mux.HandleFunc("/api/hotkeys/rename", s.handleHotkeysRename)
	mux.HandleFunc("/api/hotkeys/delete", s.handleHotkeysDelete)
	mux.HandleFunc("/api/hotkeys/update", s.handleHotkeysUpdate)
	mux.HandleFunc("/api/hotkeys/detail", s.handleHotkeysDetail)
	mux.HandleFunc("/api/settings/client-paths", s.handleClientPaths)
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

	log.Printf("Listening on %s", s.ln.Addr())

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.srv = &http.Server{Handler: s.mux}

	eg.Go(func() error {
		return s.srv.Serve(ln)
	})

	eg.Go(func() error {
		<-ctx.Done()
		s.srv.Shutdown(ctx)
		return ctx.Err()
	})

	eg.Go(func() error {
		fails := 0
		for {
			s.keepaliveMu.Lock()
			if time.Now().After(s.keepalive.Add(s.keepaliveTimeout)) {
				fails++
			} else {
				fails = 0
			}
			s.keepaliveMu.Unlock()

			if fails >= s.keepaliveFails {
				log.Printf("Last %d ping checks showed no activity within the last %v; quitting.", s.keepaliveFails, s.keepaliveTimeout)
				cancel()
				return context.Canceled
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.keepaliveCheckPeriod):
			}
		}
	})

	eg.Go(func() error {
		exec.Command("explorer", "http://"+s.ln.Addr().String()).Start()
		return nil
	})

	eg.Go(func() error {
		s.maybeAutoUpdate()
		return nil
	})

	err = eg.Wait()

	if s.restart {
		return ErrRestart
	}

	return err
}

type Server struct {
	dataDir     string
	tmpDir      string
	acc         *acc.Storage
	exp         *exp.Cache
	pinger      *ping.Pinger
	online      *online.Online
	stStorage   *settings.Storage
	tmStorage   *timer.Storage
	loot        *loot.Service
	lootStorage *loot.Storage
	hkStorage   *hotkey.Storage

	srv *http.Server
	mux *http.ServeMux
	ln  net.Listener

	version string

	cancel               func()
	keepalive            time.Time
	keepaliveTimeout     time.Duration
	keepaliveFails       int
	keepaliveCheckPeriod time.Duration
	keepaliveMu          sync.Mutex

	updateReady   bool
	updaterPath   string
	updaterSource string
	updateMu      sync.Mutex

	restart bool
}

func (s *Server) handleAccList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.acc.ListRows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type accRow struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := []accRow{} // JSON nil != empty slice.
	for _, row := range rows {
		out = append(out, accRow{ID: row.ID, Name: row.Name})
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

	log.Printf("Account ID=%q renamed to %q.", req.ID, req.Name)

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

	log.Printf("Account ID=%q deleted.", req.ID)

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

	if err := acc.RegRestore(s.stStorage.Preset().RegistryPath, row.A, row.B, row.C); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Account %q (ID=%q) loaded.", row.Name, row.ID)

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

	a, b, c, err := acc.RegSnapshot(s.stStorage.Preset().RegistryPath)
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

	log.Printf("Account %q (ID=%q) stored.", name, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (s *Server) handleExpReset(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session reset.")
	s.exp.Reset()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStart(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session started.")
	s.exp.Start()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStop(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session stopped.")
	s.exp.Stop()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpPause(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session paused.")
	s.exp.Pause()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpUnpause(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session unpaused.")
	s.exp.Unpause()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStats(w http.ResponseWriter, r *http.Request) {
	stats := s.exp.Stats()
	ret := struct {
		Level        int `json:"level,omitempty"`
		TotalExp     int `json:"total_exp,omitempty"`
		RemainingExp int `json:"remaining_exp,omitempty"`

		SessionDelta       int `json:"session_delta,omitempty"`
		SessionDurationSec int `json:"session_duration_sec,omitempty"`
		SessionRate        int `json:"session_rate,omitempty"`

		Running bool `json:"running"`
		Paused  bool `json:"paused"`
	}{
		Level:        stats.Level,
		TotalExp:     stats.TotalExp,
		RemainingExp: stats.RemainingExp,

		SessionDelta:       stats.SessionDelta,
		SessionDurationSec: int(stats.SessionDuration / time.Second),
		SessionRate:        stats.SessionRate,

		Running: stats.Running,
		Paused:  stats.Paused,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handleWorldPing(w http.ResponseWriter, r *http.Request) {
	stats := s.pinger.Stats()
	ret := struct {
		OK         bool    `json:"ok"`
		RTTMSec    int     `json:"rtt_msec"`
		PacketLoss float64 `json:"packet_loss"`
	}{
		OK:         stats.OK,
		RTTMSec:    int(stats.RTT / time.Millisecond),
		PacketLoss: stats.PacketLoss,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handleWorldOnline(w http.ResponseWriter, r *http.Request) {
	online, ok := s.online.Get()
	ret := struct {
		OK     bool `json:"ok"`
		Online int  `json:"online"`
	}{
		OK:     ok,
		Online: online,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handleKeepalive(w http.ResponseWriter, r *http.Request) {
	s.keepaliveMu.Lock()
	defer s.keepaliveMu.Unlock()
	s.keepalive = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Version string `json:"version"`
	}{Version: s.version})
}

func (s *Server) handlePresetSwitch(w http.ResponseWriter, r *http.Request) {
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

	if err := s.stStorage.SwitchPreset(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))

	s.restart = true
	s.cancel()
}

func (s *Server) handlePresetList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Active    string   `json:"active"`
		Available []string `json:"available"`
	}{
		Active:    s.stStorage.Preset().World,
		Available: slices.Sorted(maps.Keys(settings.Presets)),
	})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Mode        string `json:"mode"`
			SkipVersion string `json:"skip_version,omitempty"`
		}{
			Mode:        s.stStorage.UpdaterMode(),
			SkipVersion: s.stStorage.SkipVersion(),
		})
	case http.MethodPost:
		var req struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.stStorage.SetUpdaterMode(req.Mode); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Updater mode set to %q.", req.Mode)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	default:
		http.Error(w, "GET/POST only", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUpdateSkip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}

	if err := s.stStorage.SetSkipVersion(req.Version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Version %q skipped.", req.Version)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleUpdateExecute(w http.ResponseWriter, r *http.Request) {
	if !s.performUpdateExecute() {
		http.Error(w, "Update is not ready.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) maybeAutoUpdate() {
	if s.stStorage.UpdaterMode() != settings.UpdaterModeAutomatic {
		return
	}

	result := s.performUpdateCheck()
	if !result.Available {
		return
	}

	log.Printf("Auto-updating to %s...", result.Version)
	s.performUpdateExecute()
}

func (s *Server) performUpdateExecute() bool {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	if !s.updateReady {
		return false
	}

	go exec.Command("cmd", "/C", "start", s.updaterPath, os.Args[0], s.updaterSource).Run()
	time.Sleep(3 * time.Second)

	s.cancel()
	return true
}

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.stStorage.UpdaterMode() == settings.UpdaterModeDisabled {
		json.NewEncoder(w).Encode(struct {
			Available bool `json:"available"`
		}{})
		return
	}

	json.NewEncoder(w).Encode(s.performUpdateCheck())
}

type updateCheckResult struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
}

func (s *Server) performUpdateCheck() updateCheckResult {
	ret := updateCheckResult{}

	matched, err := regexp.MatchString(`^v\d+\.\d+\.\d+$`, s.version)
	if err != nil || !matched {
		return ret
	}

	resp, err := http.Get("https://api.github.com/repos/s5i/tassist/releases/latest")
	if err != nil {
		return ret
	}
	defer resp.Body.Close()

	var releaseData struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releaseData); err != nil {
		return ret
	}

	if releaseData.TagName == s.version {
		return ret
	}

	if releaseData.TagName == s.stStorage.SkipVersion() {
		return ret
	}

	var tassistURL, updaterURL string
	for _, asset := range releaseData.Assets {
		switch asset.Name {
		case "tassist.exe":
			tassistURL = asset.BrowserDownloadURL
		case "updater.exe":
			updaterURL = asset.BrowserDownloadURL
		}
	}
	if tassistURL == "" || updaterURL == "" {
		return ret
	}

	sourcePath := filepath.Join(s.tmpDir, fmt.Sprintf("tassist_%s.exe", releaseData.TagName))
	if err := downloadFile(sourcePath, tassistURL); err != nil {
		return ret
	}

	updaterPath := filepath.Join(s.tmpDir, "updater.exe")
	if err := downloadFile(updaterPath, updaterURL); err != nil {
		return ret
	}

	ret.Available = true
	ret.Version = releaseData.TagName

	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	s.updateReady = true
	s.updaterPath = updaterPath
	s.updaterSource = sourcePath

	return ret
}

func downloadFile(path string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func (s *Server) handleTimerAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name    string `json:"name"`
		Period  string `json:"period"`
		Loop    bool   `json:"loop"`
		Sound   bool   `json:"sound"`
		AutoAck bool   `json:"auto_ack"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	period, err := time.ParseDuration(req.Period)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid period: %v", err), http.StatusBadRequest)
		return
	}
	if period <= 0 {
		http.Error(w, "timer period must be positive", http.StatusBadRequest)
		return
	}

	t := &timer.Timer{
		ID:      uuid.New().String()[:8],
		Name:    req.Name,
		Period:  period,
		Loop:    req.Loop,
		Sound:   req.Sound,
		AutoAck: req.AutoAck,
	}

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) added.", t.Name, t.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID string `json:"id"`
	}{ID: t.ID})
}

func (s *Server) handleTimerStart(w http.ResponseWriter, r *http.Request) {
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

	t, found, err := s.tmStorage.Find(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Timer not found.", http.StatusNotFound)
		return
	}

	t.Started = time.Now()
	t.Acked = t.Started
	t.Active = true

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) started.", t.Name, t.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerStop(w http.ResponseWriter, r *http.Request) {
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

	t, found, err := s.tmStorage.Find(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Timer not found.", http.StatusNotFound)
		return
	}

	t.Started = time.Time{}
	t.Acked = t.Started
	t.Active = false

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) stopped.", t.Name, t.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerAck(w http.ResponseWriter, r *http.Request) {
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

	t, found, err := s.tmStorage.Find(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Timer not found.", http.StatusNotFound)
		return
	}

	if !t.Loop && t.Active && t.Acked.Add(t.Period).Before(time.Now()) {
		t.Started = time.Time{}
		t.Acked = t.Started
		t.Active = false
	} else {
		t.Acked = t.Started.Add(t.Period * (time.Since(t.Started) / t.Period))
	}

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) acked.", t.Name, t.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerLoop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID   string `json:"id"`
		Loop bool   `json:"loop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, found, err := s.tmStorage.Find(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Timer not found.", http.StatusNotFound)
		return
	}

	t.Loop = req.Loop
	if !t.Loop {
		t.Started = t.Started.Add(t.Period * (time.Since(t.Started) / t.Period))
	}

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) loop set to %v.", t.Name, t.ID, req.Loop)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerSound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID    string `json:"id"`
		Sound bool   `json:"sound"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, found, err := s.tmStorage.Find(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Timer not found.", http.StatusNotFound)
		return
	}

	t.Sound = req.Sound

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) sound set to %v.", t.Name, t.ID, req.Sound)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerAutoAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID      string `json:"id"`
		AutoAck bool   `json:"auto_ack"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, found, err := s.tmStorage.Find(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Timer not found.", http.StatusNotFound)
		return
	}

	t.AutoAck = req.AutoAck

	if err := s.tmStorage.AddOrUpdate(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer %q (ID=%q) auto_ack set to %v.", t.Name, t.ID, req.AutoAck)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerRemove(w http.ResponseWriter, r *http.Request) {
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

	if err := s.tmStorage.Delete(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Timer ID=%q removed.", req.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleTimerList(w http.ResponseWriter, r *http.Request) {
	timers, err := s.tmStorage.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type timerJSON struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Period    int    `json:"period"`
		Active    bool   `json:"active"`
		Loop      bool   `json:"loop"`
		Sound     bool   `json:"sound"`
		AutoAck   bool   `json:"auto_ack"`
		Firing    bool   `json:"firing"`
		Remaining int    `json:"remaining"`
	}

	out := []timerJSON{} // JSON nil != empty slice.
	for _, t := range timers {
		firing := t.Active && t.Acked.Add(t.Period).Before(time.Now())
		lastLoopStarted := t.Started.Add(t.Period)
		if t.Loop {
			lastLoopStarted = lastLoopStarted.Add(t.Period * (time.Since(t.Started) / t.Period))
		}
		remaining := max(time.Until(lastLoopStarted), 0)

		out = append(out, timerJSON{
			ID:        t.ID,
			Name:      t.Name,
			Period:    int(t.Period / time.Second),
			Active:    t.Active,
			Loop:      t.Loop,
			Sound:     t.Sound,
			AutoAck:   t.AutoAck,
			Firing:    firing,
			Remaining: int(remaining / time.Second),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleClientPaths(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		out := make(map[string]string)
		for id := range settings.Presets {
			if p := s.stStorage.ClientPath(id); p != "" {
				out[id] = p
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	case http.MethodPost:
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for world, path := range req {
			if err := s.stStorage.SetClientPath(world, path); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Client paths updated.")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	default:
		http.Error(w, "GET/POST only", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHotkeysList(w http.ResponseWriter, r *http.Request) {
	configs := s.hkStorage.List(s.stStorage.Preset().Server)
	type hkRow struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := []hkRow{}
	for _, c := range configs {
		out = append(out, hkRow{ID: c.ID, Name: c.Name})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleHotkeysStore(w http.ResponseWriter, r *http.Request) {
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

	cfgPath := s.stStorage.CfgPath()
	if cfgPath == "" {
		http.Error(w, "Client path not configured. Set it in Settings.", http.StatusBadRequest)
		return
	}

	hk, err := hotkey.ReadConfig(cfgPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := uuid.New().String()[:8]
	name := req.Name
	if name == "" {
		name = "Unnamed"
	}

	cfg := &hotkey.Config{
		ID:      id,
		Server:  s.stStorage.Preset().Server,
		Name:    name,
		Hotkeys: hk,
	}
	if err := s.hkStorage.Add(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Hotkey config %q (ID=%q) stored.", name, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (s *Server) handleHotkeysLoad(w http.ResponseWriter, r *http.Request) {
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

	if err := hotkey.ErrClientRunning(s.stStorage.Preset().ClientWindowTitle); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	cfg, found := s.hkStorage.Find(req.ID)
	if !found {
		http.Error(w, "Config not found.", http.StatusNotFound)
		return
	}

	cfgPath := s.stStorage.CfgPath()
	if cfgPath == "" {
		http.Error(w, "Client path not configured. Set it in Settings.", http.StatusBadRequest)
		return
	}

	if err := hotkey.WriteConfig(cfgPath, cfg.Hotkeys); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Hotkey config %q (ID=%q) loaded.", cfg.Name, cfg.ID)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHotkeysRename(w http.ResponseWriter, r *http.Request) {
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
	if err := s.hkStorage.Rename(req.ID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Hotkey config ID=%q renamed to %q.", req.ID, req.Name)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHotkeysDelete(w http.ResponseWriter, r *http.Request) {
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
	if err := s.hkStorage.Delete(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Hotkey config ID=%q deleted.", req.ID)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHotkeysUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID      string                `json:"id"`
		Hotkeys map[int]hotkey.Hotkey `json:"hotkeys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, found := s.hkStorage.Find(req.ID)
	if !found {
		http.Error(w, "Config not found.", http.StatusNotFound)
		return
	}

	cfg.Hotkeys = req.Hotkeys
	if err := s.hkStorage.Update(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Hotkey config %q (ID=%q) updated.", cfg.Name, cfg.ID)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleHotkeysDetail(w http.ResponseWriter, r *http.Request) {
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

	cfg, found := s.hkStorage.Find(req.ID)
	if !found {
		http.Error(w, "Config not found.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleBeep(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile(filepath.Join(s.dataDir, "beep.mp3"))
	if err != nil {
		s.handleStatic(w, r)
		return
	}

	w.Header().Set("Content-Type", contentType("/beep.mp3"))
	if _, err := w.Write(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	f := r.URL.Path
	if f == "/" {
		f = "/index.html"
	}

	content, err := staticData.ReadFile(path.Join("static", f))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType(f))
	if _, err := w.Write(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleLootItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.loot.Items()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleLootPrices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(s.lootStorage.Prices()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		var req struct {
			Prices map[int]int `json:"prices"`
			Reset  bool        `json:"reset"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var err error
		if req.Reset {
			err = s.lootStorage.Reset()
		} else {
			err = s.lootStorage.SetPrices(req.Prices)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(s.lootStorage.Prices()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLootProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	const maxUpload = 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing image field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	counts, err := s.loot.ProcessPNG(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ret := make(map[string]int, len(counts))
	for id, count := range counts {
		ret[fmt.Sprintf("%d", id)] = count
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//go:embed static/*
var staticData embed.FS

func contentType(fName string) string {
	split := strings.Split(fName, ".")
	ext := split[len(split)-1]

	switch ext {
	case "", "html":
		return "text/html; charset=utf-8"
	case "js":
		return "application/javascript; charset=utf-8"
	case "css":
		return "text/css; charset=utf-8"
	case "ico":
		return "image/x-icon"
	case "png":
		return "image/png"
	case "mp3":
		return "audio/mp3"
	default:
		return "text/plain; charset=utf-8"
	}
}
