// Package steam locates the local Steam installation and the user save data
// directory for a specific Steam app.
package steam

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// GameAppID is the Steam app id for "我在地府打麻将".
const GameAppID = "3444020"

// SteamUser represents one Steam user account that has save data for the game.
type SteamUser struct {
	SteamID    string `json:"steamId"`
	RemotePath string `json:"remotePath"`
}

// Info bundles the Steam install path and the discovered user accounts.
type Info struct {
	SteamPath string      `json:"steamPath"`
	Users     []SteamUser `json:"users"`
}

var steamIDRe = regexp.MustCompile(`^\d+$`)

// DetectPath reads the Steam install directory from the registry.
// It first checks HKCU (per-user install), then falls back to HKLM 32-bit view
// (typical for the official Steam installer on 64-bit Windows).
func DetectPath() (string, error) {
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

// DetectUsers scans {steamPath}\userdata for numeric sub-directories that
// contain {GameAppID}\remote. The returned users are sorted by steamID.
func DetectUsers(steamPath string) ([]SteamUser, error) {
	userdata := filepath.Join(steamPath, "userdata")
	entries, err := os.ReadDir(userdata)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var users []SteamUser
	for _, e := range entries {
		if !e.IsDir() || !steamIDRe.MatchString(e.Name()) {
			continue
		}
		remote := filepath.Join(userdata, e.Name(), GameAppID, "remote")
		if fi, err := os.Stat(remote); err != nil || !fi.IsDir() {
			continue
		}
		users = append(users, SteamUser{
			SteamID:    e.Name(),
			RemotePath: remote,
		})
	}
	return users, nil
}

// Detect is a convenience wrapper returning the full Info.
func Detect() (Info, error) {
	sp, err := DetectPath()
	if err != nil {
		return Info{}, err
	}
	users, err := DetectUsers(sp)
	if err != nil {
		return Info{SteamPath: sp}, err
	}
	return Info{SteamPath: sp, Users: users}, nil
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
