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
	"syscall" // Importante: Necesario para desacoplar el proceso
	"time"
)

// EnsureDaemon asegura que el servidor de Mega esté corriendo independiente de la App
func EnsureDaemon() error {
	// 1. Probamos si ya responde (para no lanzar otro proceso)
	if err := exec.Command("mega-whoami").Run(); err == nil {
		return nil // Ya está corriendo
	}

	// 2. Si no responde, lo iniciamos DESACOPLADO
	cmd := exec.Command("mega-cmd-server")

	// ESTA ES LA CLAVE: Setsid: true crea una nueva sesión.
	// El proceso deja de ser hijo de tu app y el sistema no lo mata al cerrar.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error iniciando servidor mega: %v", err)
	}

	// Damos unos segundos para que arranque antes de seguir
	time.Sleep(2 * time.Second)
	return nil
}

// Login conecta usando la sintaxis correcta
func Login(user, pass, code2FA string) error {
	EnsureDaemon() // Aseguramos que el servidor exista antes de intentar login

	exec.Command("mega-logout").Run()
	// ... (resto de tu función Login igual que antes) ...
