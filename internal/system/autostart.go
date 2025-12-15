package system

import (
	"os"
	"path/filepath"
	"text/template"
)

const desktopTemplate = `[Desktop Entry]
Type=Application
Name=CloudMount Wizard
Comment=Montador de nubes automático
Exec={{.ExecPath}} {{.Args}}
Icon={{.IconPath}}
Terminal=false
Categories=Utility;
X-GNOME-Autostart-enabled=true
`

type desktopConfig struct {
	ExecPath string
	Args     string
	IconPath string
}

func getAutostartPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// Ruta estándar en Linux: ~/.config/autostart/
	dir := filepath.Join(configDir, "autostart")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "com.anabasasoft.cloudmount.desktop"), nil
}

// SetAutostart activa o desactiva el inicio automático
func SetAutostart(enabled bool, minimized bool) error {
	path, err := getAutostartPath()
	if err != nil {
		return err
	}

	if !enabled {
		// Si se desactiva, borramos el archivo
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
		return nil
	}

	// Obtener ruta del ejecutable actual
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Argumentos: Si quiere minimizado, añadimos el flag
	args := ""
	if minimized {
		args = "--minimized"
	}

	// Icono (intentamos buscar uno genérico o usamos el binario si tiene)
	// En producción deberías instalar el icono en /usr/share/icons
	icon := "system-file-manager"

	data := desktopConfig{
		ExecPath: exe,
		Args:     args,
		IconPath: icon,
	}

	// Crear el archivo .desktop
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := template.New("desktop").Parse(desktopTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(f, data)
}

// IsAutostartEnabled verifica si el archivo existe
func IsAutostartEnabled() bool {
	path, err := getAutostartPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
