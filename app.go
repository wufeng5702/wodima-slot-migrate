package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wodima-slot-migrate/internal/android"
	"wodima-slot-migrate/internal/migrate"
	"wodima-slot-migrate/internal/steam"
)

// App holds the Wails context used for runtime calls (dialogs, events).
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so we can invoke
// runtime methods (file dialogs, events) from bound methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// DetectSteam returns the Steam install path and every user that owns the game's
// remote save directory. Called automatically by the frontend on startup.
func (a *App) DetectSteam() (steam.Info, error) {
	return steam.Detect()
}

// PickRemoteManually opens a directory picker so the user can choose a Steam
// remote folder manually (e.g. when Steam is installed in a portable location).
func (a *App) PickRemoteManually() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Steam remote save directory",
	})
}

// PickAndroidDBManually opens a file picker so the user can choose a game.db
// file previously copied to the PC.
func (a *App) PickAndroidDBManually() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Android game.db",
		Filters: []runtime.FileFilter{
			{DisplayName: "game.db (*.db, *.sqlite)", Pattern: "*.db;*.sqlite;game.db"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
}

// AutoFetchAndroidDB extracts the bundled adb, locates a connected device and
// pulls game.db into a per-user cache directory. Returns the local path.
func (a *App) AutoFetchAndroidDB() (string, error) {
	adbPath, err := android.EnsureADB()
	if err != nil {
		return "", fmt.Errorf("prepare adb: %w", err)
	}
	devices, err := android.ListDevices(adbPath)
	if err != nil {
		return "", err
	}
	var serial string
	for _, d := range devices {
		if d.State == "device" {
			serial = d.Serial
			break
		}
	}
	if serial == "" {
		if len(devices) > 0 {
			return "", android.ErrUnauthorized
		}
		return "", android.ErrNoDevice
	}
	dst, err := cachedGameDBPath()
	if err != nil {
		return "", err
	}
	if err := android.PullGameDB(adbPath, serial, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// ReadAndroidSlots reads all rows from the slot table of the given game.db.
func (a *App) ReadAndroidSlots(dbPath string) ([]android.SlotRow, error) {
	return android.ReadSlots(dbPath)
}

// Migrate runs the backup-and-write migration for each selected slot row.
func (a *App) Migrate(remotePath string, selections []migrate.SlotSelection) ([]migrate.MigrateResult, error) {
	if remotePath == "" {
		return nil, fmt.Errorf("remote path is empty")
	}
	if len(selections) == 0 {
		return nil, fmt.Errorf("no slot selected")
	}
	return migrate.Run(remotePath, selections), nil
}

// StartWifiServer starts an HTTP server on the local network for receiving
// game.db via Wi-Fi upload from the user's phone.
func (a *App) StartWifiServer() (*android.WifiResult, error) {
	dst, err := cachedGameDBPath()
	if err != nil {
		return nil, err
	}
	result, err := android.StartWifiServer(dst)
	if err != nil {
		runtime.LogDebug(a.ctx, "Wi-Fi server start failed: "+err.Error())
		return nil, err
	}
	runtime.LogDebug(a.ctx, "Wi-Fi server started: "+result.URL)
	return result, nil
}

// StopWifiServer stops the currently running Wi-Fi server.
func (a *App) StopWifiServer() error {
	android.StopWifiServer()
	return nil
}

// CheckWifiUpload polls the Wi-Fi server for upload completion.
// Returns the local file path when upload is done, empty string otherwise.
func (a *App) CheckWifiUpload(token string) (string, error) {
	return android.CheckWifiUpload(token)
}

// cachedGameDBPath returns the destination where pulled game.db will be stored.
// It lives in the user cache dir so repeated migrations do not pollute the OS temp.
func cachedGameDBPath() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "wodima-migrate")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "game.db"), nil
}
