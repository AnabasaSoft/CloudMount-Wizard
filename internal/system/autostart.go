package system

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"fyne.io/fyne/v2/lang"
)

const desktopTemplate = `[Desktop Entry]
Type=Application
Name=CloudMount Wizard
Comment={{.Comment}}
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
	Comment  string
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

	// Obtener ruta del ejecutable actual. En una AppImage, os.Executable() apunta
	// al punto de montaje temporal (/tmp/.mount_xxx), que desaparece al cerrar;
	// la ruta real del fichero .AppImage viene en la variable APPIMAGE.
	exe := os.Getenv("APPIMAGE")
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return err
		}
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
		ExecPath: quoteExecArg(exe),
		Args:     args,
		IconPath: icon,
		Comment:  lang.L("Automatic cloud drive mounter"),
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

// quoteExecArg entrecomilla una ruta para la línea Exec= de un .desktop según la
// especificación, por si contiene espacios (ej: una AppImage en "~/Mis Apps")
func quoteExecArg(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	quoted := `"` + r.Replace(s) + `"`
	// Las barras invertidas se escapan otra vez: el .desktop se lee primero como
	// cadena (\\ -> \) y después se interpretan las comillas
	return strings.ReplaceAll(quoted, `\`, `\\`)
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

// IsAutostartMinimized indica si el autoarranque está configurado con --minimized
func IsAutostartMinimized() bool {
	path, err := getAutostartPath()
	if err != nil {
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "--minimized")
}
