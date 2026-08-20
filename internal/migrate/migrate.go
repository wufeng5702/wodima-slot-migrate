// Package migrate backs up the existing Steam save file and writes the new
// jsonString coming from the Android database.
package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SlotSelection is a single row chosen by the user for migration.
type SlotSelection struct {
	ID         int64  `json:"id"`
	SlotIndex  int    `json:"slotIndex"`
	JSONString string `json:"jsonString"`
}

// Result MigrateResult is the outcome of migrating one SlotSelection.
type Result struct {
	ID         int64  `json:"id"`
	SlotIndex  int    `json:"slotIndex"`
	TargetFile string `json:"targetFile"`
	BackupFile string `json:"backupFile"`
	Success    bool   `json:"success"`
	Error      string `json:"error"`
}

var slotIndexRe = regexp.MustCompile(`^[0-2]$`)

type Service struct {
	app *application.App
}

func NewService(app *application.App) *Service {
	return &Service{app: app}
}

// MigrateRequest carries arguments for Migrate.
type MigrateRequest struct {
	RemotePath string          `json:"remotePath"`
	Selections []SlotSelection `json:"selections"`
}

// MigrateResponse is the return envelope for Migrate.
type MigrateResponse struct {
	Results []Result `json:"results"`
}

// Migrate performs the backup-and-write for each selection. Each selection is
// processed independently: a failure on one does not stop the others, and the
// per-item outcome is reported back so the UI can show a precise log.
//
// Backup policy: an existing target file is renamed to
// `Slot{X}.json.{YYYYMMDD_HHMMSS}.bak`. If a backup with the same name already
// exists (sub-second retry within the same second), a numeric suffix is appended.
func (s *Service) Migrate(req *MigrateRequest) (*MigrateResponse, error) {
	results := make([]Result, 0, len(req.Selections))
	for _, slot := range req.Selections {
		results = append(results, s.migrateOne(req.RemotePath, slot))
	}
	return &MigrateResponse{Results: results}, nil
}

func (s *Service) migrateOne(remotePath string, slot SlotSelection) Result {
	res := Result{
		ID:        slot.ID,
		SlotIndex: slot.SlotIndex,
	}
	if !slotIndexRe.MatchString(fmt.Sprintf("%d", slot.SlotIndex)) {
		res.Error = "invalid slotIndex (must be 0, 1 or 2)"
		return res
	}
	target := filepath.Join(remotePath, fmt.Sprintf("Slot%d.json", slot.SlotIndex))
	res.TargetFile = target

	// Backup existing file if present.
	if _, err := os.Stat(target); err == nil {
		backup, berr := s.backupName(remotePath, slot.SlotIndex)
		if berr != nil {
			res.Error = fmt.Sprintf("compute backup name: %v", berr)
			return res
		}
		if err := os.Rename(target, backup); err != nil {
			res.Error = fmt.Sprintf("backup existing save: %v", err)
			return res
		}
		res.BackupFile = backup
	} else if !errors.Is(err, os.ErrNotExist) {
		res.Error = fmt.Sprintf("stat target: %v", err)
		return res
	}

	// Write the new content (UTF-8, no BOM).
	if err := os.WriteFile(target, []byte(slot.JSONString), 0o644); err != nil {
		res.Error = fmt.Sprintf("write new save: %v", err)
		return res
	}
	res.Success = true
	return res
}

// backupName returns a non-existing backup path for the given slot using a
// timestamp; if the path already exists it appends a counter suffix.
func (s *Service) backupName(remotePath string, slotIndex int) (string, error) {
	base := fmt.Sprintf("Slot%d.json", slotIndex)
	stamp := time.Now().Format("20060102_150405")
	candidate := filepath.Join(remotePath, base+"."+stamp+".bak")
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
		candidate = filepath.Join(remotePath, fmt.Sprintf("%s.%s.%d.bak", base, stamp, i))
	}
}
