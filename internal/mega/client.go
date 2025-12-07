package mega

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Login conecta usando la sintaxis correcta (--auth-code al final)
func Login(user, pass, code2FA string) error {
	exec.Command("mega-logout").Run()

	args := []string{user, pass}
	if code2FA != "" {
		args = append(args, "--auth-code="+code2FA)
	}

	cmd := exec.Command("mega-login", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("falló mega-login: %s", string(out))
	}
	return nil
}

// GetWebDAVURL activa el servidor local de Mega
func GetWebDAVURL() (string, error) {
	cmd := exec.Command("mega-webdav", "/")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error webdav: %s", string(output))
	}

	// Buscamos la URL en la respuesta
	outStr := string(output)
	for _, word := range strings.Fields(outStr) {
		if strings.HasPrefix(word, "http") {
			return word, nil
		}
	}
	return "", fmt.Errorf("no se encontró URL en: %s", outStr)
}

// GetSpace analiza la salida exacta de mega-df que nos has pasado
func GetSpace() (int64, int64, error) {
	// Ejecutamos mega-df. En Linux forzamos inglés por seguridad,
	// pero el formato numérico suele ser estándar.
	cmd := exec.Command("mega-df")
	if os.Getenv("OS") != "Windows_NT" {
		cmd.Env = append(os.Environ(), "LC_ALL=C")
	}
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	// Buscamos la línea que dice "USED STORAGE"
	// Ejemplo: "USED STORAGE:   78281147   0.15% of 53687091200"
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "USED STORAGE") {
			// Regex explicada:
			// USED STORAGE: -> Texto literal
			// \s*(\d+)      -> Espacios y el primer numero (USADO)
			// .*of          -> Cualquier cosa hasta llegar a "of"
			// \s*(\d+)      -> Espacios y el segundo numero (TOTAL)
			re := regexp.MustCompile(`USED STORAGE:\s*(\d+).*of\s*(\d+)`)
			matches := re.FindStringSubmatch(line)

			if len(matches) >= 3 {
				used, _ := strconv.ParseInt(matches[1], 10, 64)
				total, _ := strconv.ParseInt(matches[2], 10, 64)
				return used, total, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("no se encontraron datos de espacio")
}

func Logout() {
	exec.Command("mega-logout").Run()
}

// IsLoggedIn comprueba si la sesión está activa
func IsLoggedIn() bool {
	err := exec.Command("mega-whoami").Run()
	return err == nil
}

func GetMountPath() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Nubes", "Mega")
}
