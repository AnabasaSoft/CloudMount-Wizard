package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type RemoteOptions struct {
	ReadOnly  bool   `json:"read_only"`
	CacheSize string `json:"cache_size"` // Ej: "10G"
	BwLimit   string `json:"bw_limit"`   // Ej: "2M"
}

type AppConfig struct {
	Remotes map[string]RemoteOptions `json:"remotes"`
}

var (
	current AppConfig
	mutex   sync.Mutex
)

func init() {
	current.Remotes = make(map[string]RemoteOptions)
	load()
}

// --- GETTERS ---

func GetOptions(remoteName string) RemoteOptions {
	mutex.Lock()
	defer mutex.Unlock()
	return current.Remotes[remoteName]
}

// --- SETTERS ---

func SetOptions(remoteName string, opts RemoteOptions) error {
	mutex.Lock()
	defer mutex.Unlock()

	current.Remotes[remoteName] = opts
	return save()
}

// Helpers individuales para compatibilidad (opcional, pero útil)
func GetReadOnly(remoteName string) bool { return GetOptions(remoteName).ReadOnly }

// --- PERSISTENCIA ---

func getConfigPath() string {
	configDir, _ := os.UserConfigDir()
	dir := filepath.Join(configDir, "cloudmount")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "settings.json")
}

func load() {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &current)
	}
	if current.Remotes == nil {
		current.Remotes = make(map[string]RemoteOptions)
	}
}

func save() error {
	path := getConfigPath()
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil { return err }
	return os.WriteFile(path, data, 0644)
}
