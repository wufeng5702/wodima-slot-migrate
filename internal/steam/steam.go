// Package steam locates the local Steam installation and the user save data
// directory for a specific Steam app.
package steam

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows/registry"
)

// GameAppID is the Steam app id for "我在地府打麻将".
const GameAppID = "3444020"

// User SteamUser represents one Steam user account that has save data for the game.
type User struct {
	SteamID    string `json:"steamId"`
	RemotePath string `json:"remotePath"`
}

// Info bundles the Steam install path and the discovered user accounts.
type Info struct {
	SteamPath string `json:"steamPath"`
	Users     []User `json:"users"`
}

var steamIDRe = regexp.MustCompile(`^\d+$`)

type Service struct {
	app *application.App
}

func NewService(app *application.App) *Service {
	return &Service{app: app}
}

// DetectRequest carries arguments for Detect.
type DetectRequest struct{}

// DetectResponse is the return envelope for Detect.
type DetectResponse struct {
	Info Info `json:"info"`
}

// Detect is a convenience wrapper returning the full Info.
func (s *Service) Detect(req *DetectRequest) (*DetectResponse, error) {
	sp, err := s.detectSteamPath()
	if err != nil {
		return &DetectResponse{Info: Info{}}, err
	}

	users, err := s.detectUsers(sp)
	if err != nil {
		return &DetectResponse{Info: Info{SteamPath: sp}}, err
	}

	return &DetectResponse{Info: Info{SteamPath: sp, Users: users}}, nil
}

// PickRemoteManuallyRequest carries arguments for PickRemoteManually.
type PickRemoteManuallyRequest struct{}

// PickRemoteManuallyResponse is the return envelope for PickRemoteManually.
type PickRemoteManuallyResponse struct {
	Path string `json:"path"`
}

// PickRemoteManually opens a directory picker so the user can choose a Steam
// remote folder manually (e.g. when Steam is installed in a portable location).
func (s *Service) PickRemoteManually(req *PickRemoteManuallyRequest) (*PickRemoteManuallyResponse, error) {
	if s == nil {
		return nil, errors.New("service is not initialized")
	}

	if s.app == nil {
		return nil, errors.New("app is not initialized")
	}

	if s.app.Dialog == nil {
		return nil, errors.New("dialog is not initialized")
	}

	path, err := s.app.Dialog.OpenFile().
		SetTitle("Select Steam remote save directory").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	if err != nil {
		return nil, err
	}

	return &PickRemoteManuallyResponse{Path: path}, nil
}

// detectSteamPath reads the Steam install directory from the registry.
// It first checks HKCU (per-user install), then falls back to HKLM 32-bit view
// (typical for the official Steam installer on 64-bit Windows).
func (s *Service) detectSteamPath() (string, error) {
	if p, err := readString(registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath"); err == nil {
		return normalize(p), nil
	}
	if p, err := readString(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"); err == nil {
		return normalize(p), nil
	}
	if p, err := readString(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, "InstallPath"); err == nil {
		return normalize(p), nil
	}
	return "", errors.New("steam install path not found in registry")
}

// detectUsers scans {steamPath}\userdata for numeric sub-directories that
// contain {GameAppID}\remote. The returned users are sorted by steamID.
func (s *Service) detectUsers(steamPath string) ([]User, error) {
	userdata := filepath.Join(steamPath, "userdata")
	entries, err := os.ReadDir(userdata)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var users []User
	for _, e := range entries {
		if !e.IsDir() || !steamIDRe.MatchString(e.Name()) {
			continue
		}
		remote := filepath.Join(userdata, e.Name(), GameAppID, "remote")
		if fi, err := os.Stat(remote); err != nil || !fi.IsDir() {
			continue
		}
		users = append(users, User{
			SteamID:    e.Name(),
			RemotePath: remote,
		})
	}
	return users, nil
}

func readString(k registry.Key, path, value string) (string, error) {
	key, err := registry.OpenKey(k, path, registry.QUERY_VALUE|registry.WOW64_32KEY)
	if err != nil {
		return "", err
	}
	defer key.Close()
	v, _, err := key.GetStringValue(value)
	return v, err
}

// normalize converts forward slashes from the registry into OS-native separators.
func normalize(p string) string {
	if p == "" {
		return p
	}
	return filepath.Clean(strings.ReplaceAll(p, "/", string(os.PathSeparator)))
}
