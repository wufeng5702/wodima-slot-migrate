package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Verifies the backup-and-write flow against a mock remote directory:
//   - existing Slot0.json is renamed to Slot0.json.<ts>.bak
//   - new content is written
//   - absent Slot1.json is created without a backup
func TestRunBackupAndWrite(t *testing.T) {
	dir := t.TempDir()

	// Seed an existing Slot0.json that should be backed up.
	old0 := []byte("OLD-SLOT0")
	if err := os.WriteFile(filepath.Join(dir, "Slot0.json"), old0, 0o644); err != nil {
		t.Fatal(err)
	}

	sels := []SlotSelection{
		{ID: 1, SlotIndex: 0, JSONString: "NEW-SLOT0"},
		{ID: 2, SlotIndex: 1, JSONString: "NEW-SLOT1"},
	}

	results := Run(dir, sels)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Success {
			t.Fatalf("Slot%d migration failed: %s", r.SlotIndex, r.Error)
		}
	}

	// Slot0: new content + a .bak with the old content.
	got0, err := os.ReadFile(filepath.Join(dir, "Slot0.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got0) != "NEW-SLOT0" {
		t.Fatalf("Slot0.json content = %q, want NEW-SLOT0", got0)
	}
	if results[0].BackupFile == "" || !strings.HasSuffix(results[0].BackupFile, ".bak") {
		t.Fatalf("Slot0 backup not recorded: %q", results[0].BackupFile)
	}
	bak, err := os.ReadFile(results[0].BackupFile)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(bak) != "OLD-SLOT0" {
		t.Fatalf("backup content = %q, want OLD-SLOT0", bak)
	}

	// Slot1: created directly, no backup expected.
	got1, err := os.ReadFile(filepath.Join(dir, "Slot1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got1) != "NEW-SLOT1" {
		t.Fatalf("Slot1.json content = %q, want NEW-SLOT1", got1)
	}
	if results[1].BackupFile != "" {
		t.Fatalf("Slot1 should have no backup, got %q", results[1].BackupFile)
	}

	// Backup filename format check: Slot0.json.YYYYMMDD_HHMMSS.bak
	base := filepath.Base(results[0].BackupFile)
	if !strings.HasPrefix(base, "Slot0.json.") || !strings.HasSuffix(base, ".bak") {
		t.Fatalf("backup name format wrong: %q", base)
	}
	t.Logf("backup file: %s", base)
}
