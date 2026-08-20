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
)

// SlotSelection is a single row chosen by the user for migration.
type SlotSelection struct {
	ID         int64  `json:"id"`
	SlotIndex  int    `json:"slotIndex"`
	JSONString string `json:"jsonString"`
}

// MigrateResult is the outcome of migrating one SlotSelection.
type MigrateResult struct {
	ID         int64  `json:"id"`
	SlotIndex  int    `json:"slotIndex"`
	TargetFile string `json:"targetFile"`
	BackupFile string `json:"backupFile"`
	Success    bool   `json:"success"`
	Error      string `json:"error"`
}

var slotIndexRe = regexp.MustCompile(`^[0-2]$`)

// Run performs the backup-and-write for each selection. Each selection is
// processed independently: a failure on one does not stop the others, and the
// per-item outcome is reported back so the UI can show a precise log.
//
// Backup policy: an existing target file is renamed to
// `Slot{X}.json.{YYYYMMDD_HHMMSS}.bak`. If a backup with the same name already
// exists (sub-second retry within the same second), a numeric suffix is appended.
func Run(remotePath string, selections []SlotSelection) []MigrateResult {
	results := make([]MigrateResult, 0, len(selections))
	for _, s := range selections {
		results = append(results, migrateOne(remotePath, s))
	}
	return results
}

func migrateOne(remotePath string, s SlotSelection) MigrateResult {
	res := MigrateResult{
		ID:        s.ID,
		SlotIndex: s.SlotIndex,
	}
	if !slotIndexRe.MatchString(fmt.Sprintf("%d", s.SlotIndex)) {
		res.Error = "invalid slotIndex (must be 0, 1 or 2)"
		return res
	}
	target := filepath.Join(remotePath, fmt.Sprintf("Slot%d.json", s.SlotIndex))
	res.TargetFile = target

	// Backup existing file if present.
	if _, err := os.Stat(target); err == nil {
		backup, berr := backupName(remotePath, s.SlotIndex)
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
	if err := os.WriteFile(target, []byte(s.JSONString), 0o644); err != nil {
		res.Error = fmt.Sprintf("write new save: %v", err)
		return res
	}
	res.Success = true
	return res
}

// backupName returns a non-existing backup path for the given slot using a
// timestamp; if the path already exists it appends a counter suffix.
func backupName(remotePath string, slotIndex int) (string, error) {
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
