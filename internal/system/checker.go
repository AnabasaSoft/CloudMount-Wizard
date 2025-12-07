package system

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// --- SECCIÓN RCLONE (Restaurada) ---

// CheckRclone verifica si rclone está instalado
func CheckRclone() bool {
	_, err := exec.LookPath("rclone")
	return err == nil
}

// InstallRclone intenta instalar rclone automáticamente
func InstallRclone() error {
	switch runtime.GOOS {
		case "linux", "darwin":
			// Script oficial de instalación (requiere sudo interno)
			cmd := exec.Command("sh", "-c", "curl https://rclone.org/install.sh | sudo bash")
			return cmd.Run()
		case "windows":
			if _, err := exec.LookPath("winget"); err == nil {
				return exec.Command("winget", "install", "Rclone.Rclone").Run()
			}
			return openBrowser("https://rclone.org/downloads")
		default:
			return openBrowser("https://rclone.org/downloads")
	}
}

// --- SECCIÓN MEGACMD (Nueva) ---

// CheckMegaCmd verifica si el comando 'mega-login' existe
func CheckMegaCmd() bool {
	_, err := exec.LookPath("mega-login")
	return err == nil
}

// InstallMegaCmd orquesta la descarga e instalación automática
func InstallMegaCmd() error {
	if runtime.GOOS != "linux" {
		return openBrowser("https://mega.io/cmd")
	}

	// 1. Detectar Distro
	distroID, versionID := getLinuxDistro()
	arch := runtime.GOARCH

	// 2. Obtener URL de descarga calculada
	url, filename, err := getMegaURL(distroID, versionID, arch)
	if err != nil {
		fmt.Println("No se detectó distro soportada, abriendo web:", err)
		return openBrowser("https://mega.io/cmd")
	}

	// 3. Descargar paquete
	tmpPath := filepath.Join(os.TempDir(), filename)
	fmt.Printf("Descargando %s...\n", url)
	if err := downloadFile(url, tmpPath); err != nil {
		return fmt.Errorf("error descarga: %v", err)
	}
	defer os.Remove(tmpPath) // Limpieza al terminar

	// 4. Instalar (Requiere Root -> pkexec)
	return installPackage(distroID, tmpPath)
}

// --- HERRAMIENTAS INTERNAS ---

// getLinuxDistro lee /etc/os-release para saber qué Linux es
func getLinuxDistro() (id, version string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	return id, version
}

// getMegaURL construye la URL oficial de Mega según la distro
func getMegaURL(id, version, arch string) (string, string, error) {
	baseURL := "https://mega.nz/linux/repo"
	var distroName, ext, repoArch string

	// Mapeo de Arquitectura
	if arch == "amd64" {
		repoArch = "amd64" // Deb
		if id == "fedora" || id == "centos" || id == "opensuse" {
			repoArch = "x86_64" // Rpm
		}
	} else if arch == "arm64" {
		repoArch = "arm64"
		if id == "fedora" || id == "centos" {
			repoArch = "aarch64"
		}
	} else {
		return "", "", fmt.Errorf("arquitectura no soportada: %s", arch)
	}

	// Mapeo de Distro
	switch id {
		case "ubuntu", "linuxmint", "pop", "elementary":
			distroName = "xUbuntu_" + version
			ext = "deb"
		case "debian", "kali", "parrot":
			distroName = "Debian_" + version
			ext = "deb"
		case "fedora":
			distroName = "Fedora_" + version
			ext = "rpm"
		case "arch", "manjaro":
			distroName = "Arch_Extra"
			repoArch = "x86_64"
			ext = "pkg.tar.zst"
			// --- CORRECCIÓN ARCH LINUX ---
			// URL correcta: .../megacmd-x86_64.pkg.tar.zst
			filename := fmt.Sprintf("megacmd-%s.%s", repoArch, ext)
			url := fmt.Sprintf("%s/%s/%s/%s", baseURL, distroName, repoArch, filename)
			return url, filename, nil
		default:
			return "", "", fmt.Errorf("distro desconocida: %s", id)
	}

	// Construcción estándar (Deb/Rpm)
	filename := fmt.Sprintf("megacmd-%s_%s.%s", distroName, repoArch, ext)
	if ext == "rpm" {
		filename = fmt.Sprintf("megacmd-%s.%s.%s", distroName, repoArch, ext)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", baseURL, distroName, repoArch, filename)
	return url, filename, nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("servidor devolvió %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func installPackage(distroID, filepath string) error {
	var cmd *exec.Cmd

	// Usamos pkexec para pedir contraseña gráfica
	switch distroID {
		case "ubuntu", "debian", "linuxmint", "pop", "kali":
			cmd = exec.Command("pkexec", "apt-get", "install", "-y", filepath)
		case "fedora", "centos":
			cmd = exec.Command("pkexec", "dnf", "install", "-y", filepath)
		case "arch", "manjaro":
			cmd = exec.Command("pkexec", "pacman", "-U", "--noconfirm", filepath)
		default:
			return fmt.Errorf("gestor de paquetes no soportado")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falló instalación: %s", string(output))
	}
	return nil
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
		case "linux":
			err = exec.Command("xdg-open", url).Start()
		case "windows":
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		case "darwin":
			err = exec.Command("open", url).Start()
		default:
			err = fmt.Errorf("no se puede abrir navegador")
	}
	return err
}
