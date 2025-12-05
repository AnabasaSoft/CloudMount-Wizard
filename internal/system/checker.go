package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CheckRclone busca si el binario 'rclone' existe en el PATH
func CheckRclone() bool {
	_, err := exec.LookPath("rclone")
	return err == nil
}

// InstallRclone detecta la distro e instala rclone con permisos de root
func InstallRclone() error {
	distro := getDistroID()
	var cmd *exec.Cmd

	// Normalizamos a minúsculas para evitar errores
	distro = strings.ToLower(distro)

	switch {
		// CASO ARCH / MANJARO
		case strings.Contains(distro, "manjaro") || strings.Contains(distro, "arch"):
			cmd = exec.Command("pkexec", "pacman", "-S", "--noconfirm", "rclone")

			// CASO DEBIAN / UBUNTU / MINT / POP!_OS
			case strings.Contains(distro, "ubuntu") || strings.Contains(distro, "debian") ||
			strings.Contains(distro, "mint") || strings.Contains(distro, "pop"):
			cmd = exec.Command("pkexec", "apt-get", "install", "-y", "rclone")

			// CASO REDHAT / FEDORA
		case strings.Contains(distro, "fedora") || strings.Contains(distro, "rhel") || strings.Contains(distro, "centos"):
			cmd = exec.Command("pkexec", "dnf", "install", "-y", "rclone")

			// CASO OPENSUSE (Leap, Tumbleweed, Suse)
		case strings.Contains(distro, "suse"):
			// Zypper es el gestor de OpenSUSE
			// --non-interactive evita preguntas de confirmación
			cmd = exec.Command("pkexec", "zypper", "install", "--non-interactive", "rclone")

		default:
			return fmt.Errorf("no se ha podido identificar el gestor de paquetes para: %s. Instálalo manualmente", distro)
	}

	// Ejecutamos el comando
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fallo al instalar: %v\nOutput: %s", err, string(output))
	}
	return nil
}

func getDistroID() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.ReplaceAll(id, "\"", "")
			return strings.ToLower(id)
		}
	}
	return "unknown"
}
