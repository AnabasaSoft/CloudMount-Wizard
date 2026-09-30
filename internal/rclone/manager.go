package rclone

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2/lang"

	"github.com/anabasasoft/cloudmount-wizard/internal/settings"
)

// GetConfigDir obtiene la ruta de configuración de Rclone
func GetConfigDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "rclone")
}

// GetLogFilePath devuelve la ruta del archivo de logs.
// Si se pasa un remoteName, genera un log específico (cloudmount-Drive.log).
// Si se pasa cadena vacía, devuelve el log global.
func GetLogFilePath(remoteName string) string {
	fileName := "cloudmount.log"
	if remoteName != "" {
		fileName = fmt.Sprintf("cloudmount-%s.log", remoteName)
	}
	return filepath.Join(GetConfigDir(), fileName)
}

type Quota struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
	Free  int64 `json:"free"`
	Trash int64 `json:"trashed"`
}

func MountRemote(remoteName string) (string, error) {
	mountPoint := GetMountPath(remoteName)

	// --- FASE DE LIMPIEZA (ANTI-DUPLICADOS) ---
	// 1. Matamos específicamente el proceso rclone que esté montando ESTA unidad.
	// El patrón busca "rclone mount NombreRemoto:" para no matar otras nubes.
	cmdKill := exec.Command("pkill", "-f", fmt.Sprintf("rclone mount %s:", remoteName))
	cmdKill.Run()

	// 2. Comprobamos si el punto de montaje sigue ocupado
	if IsMounted(mountPoint) {
		// Intento 1: Desmontaje normal
		exec.Command("fusermount", "-u", mountPoint).Run()
		time.Sleep(500 * time.Millisecond) // Damos medio segundo al sistema

		// Intento 2: Si sigue montado (quizás bloqueado), forzamos (lazy unmount)
		if IsMounted(mountPoint) {
			exec.Command("fusermount", "-u", "-z", mountPoint).Run()
			time.Sleep(500 * time.Millisecond)
		}
	}
	// ------------------------------------------

	// Crear carpeta si no existe
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return "", fmt.Errorf("%s: %v", lang.L("Could not create the mount folder"), err)
	}

	// Configuración de rclone
	opts := settings.GetOptions(remoteName)
	args := []string{
		"mount", remoteName + ":", mountPoint,
		"--daemon",
		"--vfs-cache-mode", "full",
		"--volname", remoteName,
		"--log-level", "INFO",
		"--log-file", GetLogFilePath(remoteName),
	}

	if opts.ReadOnly {
		args = append(args, "--read-only")
	}
	if opts.CacheSize != "" {
		args = append(args, "--vfs-cache-max-size", opts.CacheSize)
	}
	if opts.BwLimit != "" {
		args = append(args, "--bwlimit", opts.BwLimit)
	}
	if supportsLogRotation() {
		// Rotamos el log para que no crezca sin límite (con INFO llega a decenas de MB)
		args = append(args, "--log-file-max-size", "5M", "--log-file-max-backups", "1")
	}

	// Con --daemon el proceso padre espera a que el montaje esté listo (--daemon-wait)
	// y termina con el código de salida real. No capturamos stdout/stderr: el hijo
	// demonizado heredaría la tubería y CombinedOutput no volvería nunca.
	// Los detalles del error quedan en el fichero de log de la unidad.
	cmd := exec.Command("rclone", args...)
	if err := cmd.Run(); err != nil {
		return "", errors.New(lang.L("rclone could not mount {{.Name}} ({{.Error}}).\nCheck the log: {{.Log}}",
			map[string]any{"Name": remoteName, "Error": err.Error(), "Log": GetLogFilePath(remoteName)}))
	}

	return mountPoint, nil
}

var (
	logRotationOnce      sync.Once
	logRotationSupported bool
)

// supportsLogRotation indica si el rclone instalado admite --log-file-max-size.
// Las versiones antiguas (las de algunos repos de distros) fallarían al montar
// con un flag desconocido, así que lo comprobamos una vez y lo recordamos.
func supportsLogRotation() bool {
	logRotationOnce.Do(func() {
		out, err := exec.Command("rclone", "help", "flags", "log-file").Output()
		logRotationSupported = err == nil && bytes.Contains(out, []byte("--log-file-max-size"))
	})
	return logRotationSupported
}

func CreateConfig(name, provider string) error {
	cmd := exec.Command("rclone", "config", "create", name, provider)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s:\n%s", lang.L("rclone could not create the configuration"), strings.TrimSpace(string(output)))
	}
	return nil
}

// CreateConfigWithOpts crea un remote con las opciones dadas. Si hay "pass", se
// ofusca antes pasándola por stdin, para que la contraseña en claro no aparezca
// en la línea de comandos (visible con ps para cualquier usuario del equipo)
func CreateConfigWithOpts(name, provider string, opts map[string]string) error {
	args := []string{"config", "create", name, provider}
	if pass, ok := opts["pass"]; ok && pass != "" {
		obscured, err := obscurePassword(pass)
		if err != nil {
			return err
		}
		opts["pass"] = obscured
		args = append(args, "--no-obscure")
	}
	for key, value := range opts {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}
	cmd := exec.Command("rclone", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s:\n%s", lang.L("rclone could not create the configuration"), strings.TrimSpace(string(out)))
	}
	return nil
}

// obscurePassword ofusca una contraseña con "rclone obscure -", que la lee de stdin
func obscurePassword(pass string) (string, error) {
	cmd := exec.Command("rclone", "obscure", "-")
	cmd.Stdin = strings.NewReader(pass)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %v", lang.L("Error obscuring the password"), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func ListRemotes() ([]string, error) {
	cmd := exec.Command("rclone", "listremotes")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var remotes []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if name := strings.TrimSuffix(line, ":"); name != "" {
			remotes = append(remotes, name)
		}
	}
	return remotes, nil
}

func GetQuota(remoteName string) (*Quota, error) {
	cmd := exec.Command("rclone", "about", remoteName+":", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var q Quota
	if err := json.Unmarshal(output, &q); err != nil {
		return nil, err
	}
	return &q, nil
}

func FormatBytes(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := int(math.Floor(math.Log(float64(size)) / math.Log(1024)))
	if i >= len(units) {
		i = len(units) - 1
	}
	return fmt.Sprintf("%.2f %s", float64(size)/math.Pow(1024, float64(i)), units[i])
}

func UnmountRemote(remoteName string) error {
	mountPoint := GetMountPath(remoteName)
	if !IsMounted(mountPoint) {
		return nil // Nada que desmontar
	}
	if exec.Command("fusermount", "-u", mountPoint).Run() == nil {
		return nil
	}
	// Si está ocupado, desmontaje "lazy": se completa cuando se libere
	if out, err := exec.Command("fusermount", "-u", "-z", mountPoint).CombinedOutput(); err != nil {
		return errors.New(lang.L("Could not unmount {{.Name}}: {{.Output}}", map[string]any{"Name": remoteName, "Output": strings.TrimSpace(string(out))}))
	}
	return nil
}

func DeleteRemote(remoteName string) error {
	if err := UnmountRemote(remoteName); err != nil {
		return err // No borramos la configuración de una unidad que sigue montada
	}
	if out, err := exec.Command("rclone", "config", "delete", remoteName).CombinedOutput(); err != nil {
		return errors.New(lang.L("Could not delete {{.Name}} from rclone: {{.Output}}", map[string]any{"Name": remoteName, "Output": strings.TrimSpace(string(out))}))
	}
	os.Remove(GetMountPath(remoteName)) // Solo borra la carpeta si está vacía
	os.Remove(GetLogFilePath(remoteName))
	return settings.DeleteOptions(remoteName)
}

// IsMounted comprueba si path es exactamente un punto de montaje activo.
// Compara el campo completo de /proc/mounts: buscar subcadenas confundía
// ~/Nubes/Drive con ~/Nubes/Drive2 o con ~/Nubes/GoogleDrive.
func IsMounted(path string) bool {
	content, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	target := filepath.Clean(path)
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && unescapeMountPath(fields[1]) == target {
			return true
		}
	}
	return false
}

// unescapeMountPath deshace los escapes octales de /proc/mounts (\040 = espacio, etc.)
func unescapeMountPath(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+4 <= len(s) {
			if c, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(c))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func OpenFileManager(path string) {
	exec.Command("xdg-open", path).Start()
}

func GetMountPath(remoteName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Nubes", remoteName)
}
