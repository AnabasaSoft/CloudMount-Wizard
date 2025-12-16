# CloudMount Wizard

<div align="center">

<img src="https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/Logo.png" alt="CloudMount Logo" width="200"/>

**Una interfaz gráfica moderna y elegante para Rclone en Linux**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![Fyne](https://img.shields.io/badge/Fyne-v2.7-6366F1?style=flat-square)](https://fyne.io)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![Linux](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat-square&logo=linux&logoColor=black)](https://www.linux.org/)

[Características](#características) • [Instalación](#instalación) • [Uso](#uso) • [Nubes Soportadas](#nubes-soportadas) • [Contribuir](#contribuir)

---

<img src="https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/Captura.png" alt="CloudMount Wizard Screenshot" width="100%"/>



</div>

---

## 📖 Descripción

**CloudMount Wizard** es una aplicación de escritorio que simplifica la gestión de almacenamiento en la nube bajo Linux. Diseñada con una interfaz gráfica intuitiva usando Fyne, permite montar tus servicios favoritos de almacenamiento en la nube como si fueran discos locales, sin necesidad de usar la terminal.

Con CloudMount Wizard puedes:
- ✨ Configurar conexiones de forma visual sin comandos complejos
- 🔄 Montar y desmontar nubes con un solo clic
- ⚙️ Ajustar opciones avanzadas (modo solo lectura, límites de caché y ancho de banda)
- 🚀 Habilitar montaje automático al inicio del sistema
- 📊 Visualizar el espacio usado y disponible en tiempo real

---

## ✨ Características

### 🎨 Interfaz Moderna
- Tema oscuro elegante y minimalista
- Sistema tray integrado para acceso rápido
- Diseño responsive y limpio

### ☁️ Soporte Multi-Nube
Conecta fácilmente con:
- **Servicios personales**: Google Drive, Dropbox, OneDrive, pCloud, Box, Yandex Disk
- **Autohospedados**: Nextcloud, Owncloud, WebDAV genérico
- **Almacenamiento S3**: AWS, MinIO, Wasabi, DigitalOcean Spaces
- **Otros**: Mega.nz y más

### 🔧 Funcionalidades Avanzadas
- **Automontaje**: Configura systemd para montar automáticamente al iniciar sesión
- **Opciones personalizables**:
  - Modo solo lectura
  - Límite de caché en disco
  - Límite de ancho de banda
- **Gestión completa**: Renombrar, eliminar y reconfigurar conexiones
- **Monitoreo de espacio**: Visualización en tiempo real del uso de almacenamiento

### 🛠️ Instalación Automatizada
- Detecta automáticamente si Rclone está instalado
- Instalador integrado compatible con:
  - Arch Linux / Manjaro
  - Ubuntu / Debian / Linux Mint / Pop!_OS
  - Fedora / RHEL / CentOS
  - openSUSE (Leap / Tumbleweed)

---

## 📦 Instalación

### 🚀 Instalación Rápida (Recomendada)

Elige el método según tu distribución:

#### Arch Linux / Manjaro (AUR)

> 📌 **Próximamente disponible en AUR**

```bash
# Con yay (próximamente)
yay -S cloudmount-wizard

# Con paru (próximamente)
paru -S cloudmount-wizard
```

Por ahora, puedes usar el **AppImage** o el **binario universal** (ver abajo).

#### Ubuntu / Debian / Linux Mint

```bash
# Descargar el paquete .deb desde releases
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/cloudmount-wizard_1.0.1_amd64.deb

# Instalar
sudo dpkg -i cloudmount-wizard_1.0.1_amd64.deb

# Instalar dependencias si es necesario
sudo apt-get install -f

# Ejecutar desde el menú de aplicaciones o terminal
cloudmount-wizard
```

#### Fedora / RHEL / CentOS

```bash
# Descargar el paquete .rpm desde releases
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/cloudmount-wizard-1.0.1-1.x86_64.rpm

# Instalar
sudo dnf install cloudmount-wizard-1.0.1-1.x86_64.rpm

# Ejecutar desde el menú de aplicaciones o terminal
cloudmount-wizard
```

#### openSUSE

```bash
# Descargar el paquete .rpm desde releases
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/cloudmount-wizard-1.0.1-1.x86_64.rpm

# Instalar
sudo zypper install cloudmount-wizard-1.0.1-1.x86_64.rpm

# Ejecutar desde el menú de aplicaciones o terminal
cloudmount-wizard
```

#### AppImage (Cualquier distribución) - Recomendado

El **AppImage** es la forma más fácil de ejecutar CloudMount Wizard en cualquier distribución Linux sin necesidad de instalación:

```bash
# Descargar el AppImage
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/CloudMount-Wizard.AppImage

# Hacer ejecutable
chmod +x CloudMount-Wizard.AppImage

# Ejecutar
./CloudMount-Wizard.AppImage
```

**Ventajas del AppImage:**
- ✅ No requiere instalación ni permisos de root
- ✅ Funciona en cualquier distribución Linux moderna
- ✅ Incluye todas las dependencias necesarias
- ✅ Fácil de actualizar (solo reemplaza el archivo)

Opcionalmente, puedes moverlo a un directorio en tu PATH:
```bash
mkdir -p ~/.local/bin
mv CloudMount-Wizard.AppImage ~/.local/bin/cloudmount-wizard
```

#### Binario Universal (Tar.gz)

```bash
# Descargar el binario comprimido
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/cloudmount-linux-amd64.tar.gz

# Extraer
tar -xzf cloudmount-linux-amd64.tar.gz

# Mover a /usr/local/bin (opcional)
sudo mv CloudMount-Wizard /usr/local/bin/cloudmount-wizard

# Hacer ejecutable
sudo chmod +x /usr/local/bin/cloudmount-wizard

# Ejecutar
cloudmount-wizard
```

### 🛠️ Prerequisitos

Las dependencias se instalan automáticamente con los paquetes .deb, .rpm y AppImage. Si usas el binario tar.gz, necesitarás:

```bash
# Ubuntu/Debian
sudo apt install fuse3 libgl1 libxrandr2 libxcursor1 libxinerama1 libxi6

# Fedora
sudo dnf install fuse3 mesa-libGL libXrandr libXcursor libXinerama libXi

# Arch Linux
sudo pacman -S fuse3 libgl libxrandr libxcursor libxinerama libxi
```

### 🔨 Compilar desde el código fuente

```bash
# Instalar Go (versión 1.21+)
# Ver: https://golang.org/doc/install

# Instalar dependencias de desarrollo
# Ubuntu/Debian
sudo apt install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev fuse3

# Fedora
sudo dnf install gcc mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libxkbcommon-devel wayland-devel fuse3

# Arch Linux
sudo pacman -S base-devel libgl xorg-server-devel libxkbcommon wayland fuse3

# Clonar el repositorio
git clone https://github.com/AnabasaSoft/CloudMount-Wizard.git
cd CloudMount-Wizard

# Compilar
go build -ldflags "-s -w" -o CloudMount-Wizard ./cmd/cloudmount

# Ejecutar
./CloudMount-Wizard
```

---

## 🚀 Uso

### Primera Ejecución

1. **Verificación de Rclone**: La aplicación verificará automáticamente si Rclone está instalado
2. **Instalación automática**: Si no está presente, podrás instalarlo con un solo clic
3. **Dashboard**: Una vez listo, accederás al panel principal de gestión

### Añadir una Nueva Nube

1. Haz clic en **"Nueva Conexión"**
2. Selecciona tu proveedor de nube
3. Sigue el asistente de configuración:
   - **OAuth** (Drive, Dropbox, OneDrive): Se abrirá tu navegador para autorizar
   - **Manual** (Nextcloud, WebDAV, Mega): Introduce tus credenciales
   - **S3**: Configura access key, secret key y endpoint

### 🔧 Funcionalidades Avanzadas

- **Automontaje**: Configura el inicio automático de la aplicación y el montaje de unidades.
- **Modo Silencioso**: Opción para iniciar la aplicación minimizada en la bandeja del sistema.
- **Visor de Logs**: Consola en tiempo real para ver la actividad interna de Rclone y Mega.
- **Opciones personalizables**:
  - Modo solo lectura
  - Límite de caché en disco
  - Límite de ancho de banda
- **Gestión completa**: Renombrar, eliminar y reconfigurar conexiones
- **Monitoreo de espacio**: Visualización en tiempo real del uso de almacenamiento

### Puntos de Montaje

Por defecto, las nubes se montan en:
```
~/Nubes/[NombreDeLaNube]
```

---

## ☁️ Nubes Soportadas

| Proveedor | Tipo | Autenticación |
|-----------|------|---------------|
| Google Drive | Personal | OAuth2 |
| Dropbox | Personal | OAuth2 |
| OneDrive | Personal | OAuth2 |
| pCloud | Personal | OAuth2 |
| Box | Personal | OAuth2 |
| Yandex Disk | Personal | OAuth2 |
| Mega.nz | Personal | Usuario/Contraseña |
| Nextcloud | Autohospedado | WebDAV |
| Owncloud | Autohospedado | WebDAV |
| WebDAV | Genérico | HTTP Basic Auth |
| Amazon S3 | Almacenamiento | Access/Secret Keys |
| MinIO | Autohospedado | Access/Secret Keys |
| Wasabi | Almacenamiento | Access/Secret Keys |
| DigitalOcean | Almacenamiento | Access/Secret Keys |

---

## ⚙️ Configuración Avanzada

### Opciones por Conexión

Cada conexión puede tener configuraciones específicas:

- **Modo Solo Lectura**: Previene modificaciones accidentales
- **Límite de Caché**: Controla el espacio en disco local (ej: `10G`)
- **Límite de Ancho de Banda**: Restringe la velocidad de transferencia (ej: `2M`)

### Barra de herramientas

- **Visor de Logs**: Abre una consola en tiempo real para diagnosticar problemas con Rclone o Mega.
- **Preferencias Generales**: (Icono ⚙️) Configura el arranque automático de la aplicación y el inicio minimizado (silencioso) en la bandeja del sistema.

### Archivos de Configuración

- **Rclone**: `~/.config/rclone/rclone.conf`
- **CloudMount**: `~/.config/cloudmount/settings.json`
- **Servicios systemd**: `~/.config/systemd/user/rclone-*.service`

---

## 🗃️ Arquitectura del Proyecto

```
cloudmount-wizard/
├── cmd/
│   └── cloudmount/
│       └── main.go           # Punto de entrada, UI principal
├── internal/
│   ├── rclone/
│   │   └── manager.go        # Gestión de Rclone
│   ├── settings/
│   │   └── settings.go       # Configuración persistente
│   └── system/
│       └── checker.go        # Detección e instalación
│       └── autostart.go      # Gestión de arranque automático (.desktop)
└── go.mod
```

---

## 🤝 Contribuir

¡Las contribuciones son bienvenidas! Si quieres mejorar CloudMount Wizard:

1. Haz un **Fork** del proyecto
2. Crea una rama para tu característica (`git checkout -b feature/AmazingFeature`)
3. Realiza tus cambios y haz commit (`git commit -m 'Add some AmazingFeature'`)
4. Sube los cambios (`git push origin feature/AmazingFeature`)
5. Abre un **Pull Request**

### Áreas de Mejora

- [ ] Soporte para más proveedores de nube
- [ ] Sincronización bidireccional
- [ ] Cifrado local de datos sensibles
- [ ] Traducción a otros idiomas
- [ ] Indicadores de velocidad de transferencia en tiempo real

---

## 🐛 Reportar Problemas

Si encuentras algún bug o tienes una sugerencia, por favor [abre un issue](https://github.com/anabasasoft/cloudmount-wizard/issues) en GitHub.

También puedes contactarnos directamente en: **anabasasoft@gmail.com**

---

## 📄 Licencia

Este proyecto está bajo la licencia MIT. Ver el archivo [LICENSE](LICENSE) para más detalles.

---

## 🙏 Agradecimientos

- [Rclone](https://rclone.org/) - El motor que hace posible todo
- [Fyne](https://fyne.io/) - Framework de UI multiplataforma para Go
- Comunidad Open Source - Por el apoyo y las contribuciones

---

<div align="center">

<img src="https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/AnabasaSoft.jpg" alt="Anabasa Software" width="120"/>

**Desarrollado con ❤️ por [Anabasa Software](https://anabasasoft.github.io)**

📧 Email: [anabasasoft@gmail.com](mailto:anabasasoft@gmail.com) • 🌐 Portafolio: [anabasasoft.github.io](https://anabasasoft.github.io)

⭐ Si te gusta este proyecto, dale una estrella en GitHub

</div>
