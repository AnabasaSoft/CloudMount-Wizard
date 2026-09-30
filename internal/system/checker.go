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
	"strconv"
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
	info := getLinuxDistro()
	family := distroFamily(info)

	// 2. Obtener URL de descarga calculada
	pkg, err := getMegaPackage(family, info, runtime.GOARCH)
	if err != nil {
		fmt.Println("No se detectó distro soportada, abriendo web:", err)
		return openBrowser("https://mega.io/cmd")
	}

	// 3. Descargar paquete
	tmpPath := filepath.Join(os.TempDir(), pkg.filename)
	fmt.Printf("Descargando %s...\n", pkg.url)
	if err := downloadFile(pkg.url, tmpPath); err != nil {
		return fmt.Errorf("error descarga: %v", err)
	}
	defer os.Remove(tmpPath) // Limpieza al terminar

	// 4. Instalar (Requiere Root -> pkexec)
	return installPackage(family, tmpPath, pkg.keyURL)
}

// --- HERRAMIENTAS INTERNAS ---

// getLinuxDistro lee /etc/os-release (ID, ID_LIKE, VERSION_ID, UBUNTU_CODENAME...)
func getLinuxDistro() map[string]string {
	info := make(map[string]string)
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return info
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if key, value, ok := strings.Cut(scanner.Text(), "="); ok {
			info[key] = strings.Trim(value, `"'`)
		}
	}
	return info
}

// distroFamily agrupa la distro según el paquete de MEGA que le sirve.
// Se mira primero ID y luego ID_LIKE, así Mint o Pop!_OS caen en "ubuntu"
// y Manjaro o EndeavourOS en "arch".
func distroFamily(info map[string]string) string {
	ids := append([]string{info["ID"]}, strings.Fields(info["ID_LIKE"])...)
	for _, id := range ids {
		switch {
		case id == "ubuntu":
			return "ubuntu"
		case id == "debian":
			return "debian"
		case id == "rhel" || id == "centos":
			return "" // MEGA no publica paquetes para RHEL y derivadas
		case id == "fedora":
			return "fedora"
		case strings.HasPrefix(id, "opensuse"):
			return "opensuse"
		case id == "arch":
			return "arch"
		}
	}
	return ""
}

// ubuntuCodenames traduce UBUNTU_CODENAME a versión para las derivadas de Ubuntu
// (Mint, elementary...), cuyo VERSION_ID es el suyo propio y no el de Ubuntu
var ubuntuCodenames = map[string]string{
	"focal":    "20.04",
	"jammy":    "22.04",
	"noble":    "24.04",
	"oracular": "24.10",
	"plucky":   "25.04",
	"questing": "25.10",
}

type megaPackage struct {
	url      string
	filename string
	keyURL   string // Clave GPG del repo (solo se usa en openSUSE)
}

// getMegaPackage construye la URL oficial del paquete de MEGAcmd según la distro
func getMegaPackage(family string, info map[string]string, arch string) (megaPackage, error) {
	const baseURL = "https://mega.nz/linux/repo"

	// Mapeo de Arquitectura: Deb usa amd64/arm64 y Rpm/Arch x86_64/aarch64
	var debArch, rpmArch string
	switch arch {
	case "amd64":
		debArch, rpmArch = "amd64", "x86_64"
	case "arm64":
		debArch, rpmArch = "arm64", "aarch64"
	default:
		return megaPackage{}, fmt.Errorf("arquitectura no soportada: %s", arch)
	}

	deb := func(distroName string) megaPackage {
		filename := fmt.Sprintf("megacmd-%s_%s.deb", distroName, debArch)
		return megaPackage{url: fmt.Sprintf("%s/%s/%s/%s", baseURL, distroName, debArch, filename), filename: filename}
	}
	rpm := func(distroName string) megaPackage {
		filename := fmt.Sprintf("megacmd-%s.%s.rpm", distroName, rpmArch)
		return megaPackage{
			url:      fmt.Sprintf("%s/%s/%s/%s", baseURL, distroName, rpmArch, filename),
			filename: filename,
			keyURL:   fmt.Sprintf("%s/%s/repodata/repomd.xml.key", baseURL, distroName),
		}
	}

	// Mapeo de Distro
	switch family {
	case "ubuntu":
		version := ubuntuCodenames[info["UBUNTU_CODENAME"]]
		if info["ID"] == "ubuntu" {
			version = info["VERSION_ID"]
		}
		if version == "" {
			return megaPackage{}, fmt.Errorf("versión de Ubuntu desconocida")
		}
		return deb("xUbuntu_" + version), nil

	case "debian":
		version := info["VERSION_ID"]
		if info["ID"] != "debian" {
			// Derivadas: la versión base de Debian está en /etc/debian_version (ej: "12.5")
			content, _ := os.ReadFile("/etc/debian_version")
			version, _, _ = strings.Cut(strings.TrimSpace(string(content)), ".")
		}
		if _, err := strconv.Atoi(version); err != nil {
			return megaPackage{}, fmt.Errorf("versión de Debian desconocida: %q", version)
		}
		return deb("Debian_" + version), nil

	case "fedora":
		return rpm("Fedora_" + info["VERSION_ID"]), nil

	case "opensuse":
		switch info["ID"] {
		case "opensuse-tumbleweed", "opensuse-slowroll":
			return rpm("openSUSE_Tumbleweed"), nil
		case "opensuse-leap":
			return rpm("openSUSE_Leap_" + info["VERSION_ID"]), nil
		}
		return megaPackage{}, fmt.Errorf("variante de openSUSE desconocida: %s", info["ID"])

	case "arch":
		// MEGA solo publica el paquete de Arch para x86_64
		if arch != "amd64" {
			return megaPackage{}, fmt.Errorf("arquitectura no soportada en Arch: %s", arch)
		}
		filename := "megacmd-x86_64.pkg.tar.zst"
		return megaPackage{url: fmt.Sprintf("%s/Arch_Extra/x86_64/%s", baseURL, filename), filename: filename}, nil
	}

	return megaPackage{}, fmt.Errorf("distro desconocida: %s", info["ID"])
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

func installPackage(family, filepath, keyURL string) error {
	var cmd *exec.Cmd

	// Usamos pkexec para pedir contraseña gráfica
	switch family {
	case "ubuntu", "debian":
		cmd = exec.Command("pkexec", "apt-get", "install", "-y", filepath)
	case "fedora":
		cmd = exec.Command("pkexec", "dnf", "install", "-y", filepath)
	case "opensuse":
		// zypper no instala en modo no interactivo un rpm firmado con una clave
		// desconocida, así que importamos antes la clave del repo de MEGA.
		// Todo en un solo pkexec para pedir la contraseña una única vez.
		cmd = exec.Command("pkexec", "sh", "-c", `rpm --import "$1" && zypper --non-interactive install "$2"`,
			"sh", keyURL, filepath)
	case "arch":
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
