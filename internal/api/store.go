package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ruderspiel/internal/gameservice"
)

// fileStore persists each game as <dir>/<gameID>.json (a gameservice.Snapshot).
type fileStore struct{ dir string }

// Save writes the game file atomically (temp file + rename).
func (fs fileStore) Save(snap gameservice.Snapshot) error {
	id := snap.Config.GameID
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(fs.dir, "."+id+"-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(fs.dir, id+".json"))
}

// loadAll reads every game file in dir and imports (replays) it. Broken files are logged and skipped.
func (s *Server) loadAll(dir string) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, de := range entries {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		if err := s.loadFile(strings.TrimSuffix(name, ".json"), filepath.Join(dir, name)); err != nil {
			s.log.Error("skipping game file", "file", name, "err", err)
		}
	}
	s.log.Info("games loaded", "count", s.svc.Count(), "dir", dir)
	return nil
}

func (s *Server) loadFile(id, path string) error {
	if !gameservice.ValidGameID(id) {
		return errors.New("invalid game ID in file name")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var snap gameservice.Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return err
	}
	if snap.Config.GameID != id {
		return fmt.Errorf("game ID mismatch: file %q, config %q", id, snap.Config.GameID)
	}
	_, err = s.svc.Import(snap)
	return err
}
