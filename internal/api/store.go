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

var (
	errInvalidFileID  = errors.New("invalid game ID in file name")
	errGameIDMismatch = errors.New("game ID mismatch")
)

// fileStore persists each game as <dir>/<gameID>.json (a gameservice.Snapshot).
type fileStore struct{ dir string }

// Save writes the game file atomically (temp file + rename).
func (fs fileStore) Save(snap gameservice.Snapshot) error {
	id := snap.Config.GameID

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode game %s: %w", id, err)
	}

	tmp, err := os.CreateTemp(fs.dir, "."+id+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename

	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}

	if err != nil {
		tmp.Close()
		return fmt.Errorf("write game %s: %w", id, err)
	}

	err = tmp.Close()
	if err != nil {
		return fmt.Errorf("close game file %s: %w", id, err)
	}

	err = os.Rename(tmp.Name(), filepath.Join(fs.dir, id+".json"))
	if err != nil {
		return fmt.Errorf("rename game file %s: %w", id, err)
	}

	return nil
}

// loadAll reads every game file in dir and imports (replays) it. Broken files are logged and skipped.
func (s *Server) loadAll(dir string) error {
	if dir == "" {
		return nil
	}

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read data dir: %w", err)
	}

	for _, de := range entries {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}

		err := s.loadFile(strings.TrimSuffix(name, ".json"), filepath.Join(dir, name))
		if err != nil {
			s.log.Error("skipping game file", "file", name, "err", err)
		}
	}

	s.log.Info("games loaded", "count", s.svc.Count(), "dir", dir)

	return nil
}

func (s *Server) loadFile(id, path string) error {
	if !gameservice.ValidGameID(id) {
		return errInvalidFileID
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read game file: %w", err)
	}

	var snap gameservice.Snapshot

	err = json.Unmarshal(raw, &snap)
	if err != nil {
		return fmt.Errorf("decode game file: %w", err)
	}

	if snap.Config.GameID != id {
		return fmt.Errorf("%w: file %q, config %q", errGameIDMismatch, id, snap.Config.GameID)
	}

	_, err = s.svc.Import(snap)
	if err != nil {
		return fmt.Errorf("import game %s: %w", id, err)
	}

	return nil
}
