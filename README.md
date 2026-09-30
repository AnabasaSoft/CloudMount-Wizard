# CloudMount Wizard

<div align="center">

<img src="https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/Logo.png" alt="CloudMount Logo" width="200"/>

**Una interfaz gráfica moderna y elegante para Rclone en Linux**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Fyne](https://img.shields.io/badge/Fyne-v2.7-6366F1?style=flat-square)](https://fyne.io)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![Linux](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat-square&logo=linux&logoColor=black)](https://www.linux.org/)
[![Release](https://img.shields.io/github/v/release/AnabasaSoft/CloudMount-Wizard?style=flat-square)](https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest)

[Características](#caracteristicas) • [Instalación](#instalacion) • [Uso](#uso) • [Nubes soportadas](#nubes-soportadas) • [Configuración](#configuracion) • [Contribuir](#contribuir)

---

<img src="https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/Captura.png" alt="CloudMount Wizard Screenshot" width="100%"/>

</div>

---

<a id="descripcion"></a>
## 📖 Descripción

**CloudMount Wizard** es una aplicación de escritorio que simplifica la gestión del almacenamiento en la nube en Linux. Con una interfaz gráfica intuitiva hecha con Fyne, permite montar tus servicios de almacenamiento en la nube como si fueran discos locales, sin necesidad de usar la terminal.

Con CloudMount Wizard puedes:
- ✨ Configurar conexiones de forma visual, sin comandos complejos
- 🔄 Montar y desmontar nubes con un solo clic
- ⚙️ Ajustar opciones avanzadas (solo lectura, límites de caché y de ancho de banda)
- 🚀 Montar tus unidades automáticamente al abrir la aplicación
- 📊 Ver el espacio usado y disponible de cada nube

---

<a id="caracteristicas"></a>
## ✨ Características

### 🎨 Interfaz moderna
- Tema oscuro elegante y minimalista
- Icono en la bandeja del sistema para acceso rápido
- Disponible en **español**, **inglés** y **euskera** (se elige automáticamente según el idioma del sistema o desde *Preferencias*)

### ☁️ Soporte multinube
- **Servicios personales**: Google Drive, Dropbox, OneDrive, pCloud, Box y Mega.nz
- **Autohospedados**: Nextcloud, ownCloud y cualquier servidor WebDAV
- **Almacenamiento S3**: AWS, MinIO, Wasabi y compatibles (DigitalOcean Spaces, etc.)

### 🔧 Funcionalidades
- **Automontaje**: cada unidad puede montarse sola al abrir CloudMount
- **Arranque con la sesión**: la aplicación puede iniciarse al entrar en el escritorio, también minimizada en la bandeja
- **Opciones por unidad**: solo lectura, límite de caché en disco y límite de ancho de banda
- **Espacio en disco**: uso y capacidad de cada nube montada
- **Visor de logs**: registro en tiempo real de cada unidad y un log global con la actividad de la aplicación
- **Aviso de nuevas versiones**: CloudMount te avisa cuando hay una release nueva en GitHub, con el enlace para descargarla

### 🛠️ Instalación asistida
- Detecta si Rclone está instalado y, si no, lo instala con un clic
- Para Mega.nz instala automáticamente **MEGAcmd** (el cliente oficial de MEGA)
- Compatible con:
  - openSUSE (Tumbleweed, Slowroll, Leap)
  - Ubuntu / Debian / Linux Mint / Pop!_OS / elementary OS
  - Fedora
  - Arch Linux / Manjaro / EndeavourOS

---

<a id="instalacion"></a>
## 📦 Instalación

Descarga el paquete de tu distribución desde la [última release](https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest). En los comandos de abajo, cambia `VERSION` por la versión que quieras instalar.

> 🔏 Los paquetes `.deb` y `.rpm` están **firmados con la clave GPG de AnabasaSoft** ([`firma/anabasasoft_public.asc`](firma/anabasasoft_public.asc), huella `FBFF BB1A DAA6 F42A 0520  65F7 2737 A9E2 209A 05C2`). Importarla permite comprobar que el paquete es original y no ha sido modificado.
>
> Descarga la clave a un fichero antes de importarla: algunas versiones de `rpm` fallan si se les pasa por la entrada estándar (`curl | rpm --import -`).

#### Ubuntu / Debian / Linux Mint

```bash
VERSION=1.3.0
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/download/v${VERSION}/cloudmount-wizard_${VERSION}_amd64.deb

# Opcional: verificar la firma antes de instalar (requiere el paquete dpkg-sig)
curl -sL https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/firma/anabasasoft_public.asc -o /tmp/anabasasoft_public.asc
gpg --import /tmp/anabasasoft_public.asc
dpkg-sig --verify cloudmount-wizard_${VERSION}_amd64.deb

sudo apt install ./cloudmount-wizard_${VERSION}_amd64.deb
sudo apt install rclone fuse3
```

#### Fedora

```bash
VERSION=1.3.0
curl -sL https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/firma/anabasasoft_public.asc -o /tmp/anabasasoft_public.asc
sudo rpm --import /tmp/anabasasoft_public.asc
sudo dnf install https://github.com/AnabasaSoft/CloudMount-Wizard/releases/download/v${VERSION}/cloudmount-wizard-${VERSION}-1.x86_64.rpm
sudo dnf install rclone fuse3
```

#### openSUSE

```bash
VERSION=1.3.0
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/download/v${VERSION}/cloudmount-wizard-${VERSION}-1.x86_64.rpm
curl -sL https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/firma/anabasasoft_public.asc -o /tmp/anabasasoft_public.asc
sudo rpm --import /tmp/anabasasoft_public.asc
sudo zypper install ./cloudmount-wizard-${VERSION}-1.x86_64.rpm
sudo zypper install rclone fuse3
```

#### Arch Linux / Manjaro (AUR)

Disponible en AUR como [`cloudmount-wizard-bin`](https://aur.archlinux.org/packages/cloudmount-wizard-bin):

```bash
# Con yay
yay -S cloudmount-wizard-bin

# Con paru
paru -S cloudmount-wizard-bin
```

#### AppImage (cualquier distribución)

El **AppImage** es la forma más sencilla de probar CloudMount Wizard en cualquier distribución:

```bash
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/CloudMount-Wizard.AppImage
chmod +x CloudMount-Wizard.AppImage
./CloudMount-Wizard.AppImage
```

**Ventajas del AppImage:**
- ✅ No requiere instalación ni permisos de root
- ✅ Funciona en cualquier distribución Linux moderna
- ✅ Fácil de actualizar: solo hay que sustituir el fichero

> Para usar el AppImage necesitas **FUSE** y tener instalados **Rclone** y **fuse3** (la aplicación puede instalar Rclone por ti).

#### Binario universal (tar.gz)

```bash
wget https://github.com/AnabasaSoft/CloudMount-Wizard/releases/latest/download/cloudmount-linux-amd64.tar.gz
tar -xzf cloudmount-linux-amd64.tar.gz
sudo install -m 755 CloudMount-Wizard /usr/local/bin/cloudmount-wizard
cloudmount-wizard
```

<a id="requisitos"></a>
### 🛠️ Requisitos

- **Rclone**: imprescindible. Si no está instalado, la aplicación ofrece instalarlo.
- **FUSE 3**: necesario para montar las unidades (`fuse3`).
- **MEGAcmd**: solo para Mega.nz. La aplicación lo instala al conectar tu cuenta.
- **Bibliotecas gráficas** (solo para el binario tar.gz; normalmente ya están en cualquier escritorio):

```bash
# Ubuntu/Debian
sudo apt install libgl1 libxrandr2 libxcursor1 libxinerama1 libxi6 libxxf86vm1

# Fedora
sudo dnf install mesa-libGL libXrandr libXcursor libXinerama libXi libXxf86vm

# openSUSE
sudo zypper install Mesa-libGL1 libXrandr2 libXcursor1 libXinerama1 libXi6 libXxf86vm1

# Arch Linux
sudo pacman -S libgl libxrandr libxcursor libxinerama libxi libxxf86vm
```

<a id="compilar"></a>
### 🔨 Compilar desde el código fuente

Necesitas **Go 1.25 o superior** ([instrucciones](https://go.dev/doc/install)) y las bibliotecas de desarrollo de Fyne:

```bash
# Ubuntu/Debian
sudo apt install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev

# Fedora
sudo dnf install gcc mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel libxkbcommon-devel wayland-devel

# openSUSE
sudo zypper install gcc Mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel libxkbcommon-devel wayland-devel

# Arch Linux
sudo pacman -S base-devel libgl libxcursor libxrandr libxinerama libxi libxxf86vm libxkbcommon wayland
```

```bash
git clone https://github.com/AnabasaSoft/CloudMount-Wizard.git
cd CloudMount-Wizard
go build -ldflags "-s -w -X main.version=1.3.0" -o CloudMount-Wizard ./cmd/cloudmount
./CloudMount-Wizard
```

> Con `-X main.version=...` la aplicación conoce su versión y puede avisar de actualizaciones. Si lo omites, se considera una compilación de desarrollo y no busca versiones nuevas.

---

<a id="uso"></a>
## 🚀 Uso

### Primera ejecución

1. **Comprobación de Rclone**: la aplicación verifica si Rclone está instalado.
2. **Instalación automática**: si falta, puedes instalarlo con un clic.
3. **Panel principal**: una vez listo, accedes al panel con tus unidades.

### Añadir una nube

1. Pulsa **«Nueva»**.
2. Elige tu proveedor.
3. Sigue el asistente:
   - **OAuth** (Google Drive, Dropbox, OneDrive, pCloud, Box): se abre el navegador para que autorices el acceso.
   - **Mega.nz**: introduce tu email, tu contraseña y, si lo tienes activado, el código 2FA. Tus credenciales no se guardan en la configuración de Rclone.
   - **Nextcloud / WebDAV**: introduce la URL del servidor, tu usuario y tu contraseña.
   - **S3**: elige el proveedor y escribe la access key, la secret key y, si hace falta, el endpoint.

### Panel principal

Cada unidad muestra su estado (**MONTADO**, **OFF** o **SESIÓN OK** en Mega), el espacio usado y botones para:
- **Montar** y **desmontar** la unidad
- **Abrir** la carpeta en el gestor de archivos
- **Ajustes** de la unidad (⚙️)
- **Eliminar** la unidad (🗑️)

En la barra superior tienes el **visor de logs**, las **preferencias generales** (⚙️) y el botón **«Nueva»**.

### Puntos de montaje

Las nubes se montan en:
```
~/Nubes/[NombreDeLaNube]
```

---

<a id="nubes-soportadas"></a>
## ☁️ Nubes soportadas

| Proveedor | Tipo | Autenticación |
|-----------|------|---------------|
| Google Drive | Personal | OAuth2 |
| Dropbox | Personal | OAuth2 |
| OneDrive | Personal | OAuth2 |
| pCloud | Personal | OAuth2 |
| Box | Personal | OAuth2 |
| Mega.nz | Personal | Usuario/Contraseña (vía MEGAcmd) |
| Nextcloud | Autohospedado | WebDAV |
| ownCloud | Autohospedado | WebDAV (opción «WebDAV») |
| WebDAV | Genérico | Usuario/Contraseña |
| Amazon S3 | Almacenamiento | Access/Secret Keys |
| MinIO | Autohospedado | Access/Secret Keys |
| Wasabi | Almacenamiento | Access/Secret Keys |
| Otros S3 (DigitalOcean Spaces…) | Almacenamiento | Access/Secret Keys + endpoint |

---

<a id="configuracion"></a>
## ⚙️ Configuración

### Ajustes de cada unidad

- **Solo lectura**: evita modificaciones accidentales.
- **Límite de caché**: controla el espacio en disco local (ej: `10G`).
- **Ancho de banda**: limita la velocidad de transferencia (ej: `2M`).
- **Automontaje**: monta la unidad al abrir CloudMount.

Si la unidad está montada, los cambios se aplican al desmontarla y volver a montarla.

### Preferencias generales

- **Arrancar al iniciar sesión** y **iniciar minimizado** en la bandeja del sistema.
- **Idioma** de la interfaz: automático, español, inglés o euskera. Se aplica al momento, sin reiniciar la aplicación.
- **Versión instalada** y botón para **buscar actualizaciones**.

### Ficheros

| Qué | Dónde |
|-----|-------|
| Configuración de Rclone | `~/.config/rclone/rclone.conf` |
| Ajustes de CloudMount | `~/.config/cloudmount/settings.json` |
| Log global de la aplicación | `~/.config/rclone/cloudmount.log` |
| Log de cada unidad | `~/.config/rclone/cloudmount-[Nombre].log` |
| Arranque automático | `~/.config/autostart/com.anabasasoft.cloudmount.desktop` |

Los logs de cada unidad rotan al llegar a 5 MB.

---

<a id="arquitectura"></a>
## 🗃️ Arquitectura del proyecto

```
cloudmount-wizard/
├── cmd/
│   └── cloudmount/
│       ├── main.go           # Punto de entrada e interfaz gráfica
│       ├── icon.go           # Icono embebido
│       └── translations/     # Traducciones (es.json, eu.json)
├── internal/
│   ├── mega/
│   │   └── client.go         # Integración con MEGAcmd
│   ├── rclone/
│   │   └── manager.go        # Montaje y gestión de Rclone
│   ├── settings/
│   │   └── settings.go       # Configuración persistente
│   ├── system/
│   │   ├── checker.go        # Detección de la distro e instalación de dependencias
│   │   └── autostart.go      # Arranque automático (.desktop)
│   └── update/
│       └── checker.go        # Aviso de nuevas versiones
└── go.mod
```

### Traducciones

Los textos del código están en inglés y se traducen con los ficheros de `cmd/cloudmount/translations/`. Para añadir un idioma, copia `es.json` como `<código>.json` (por ejemplo, `fr.json`), traduce los valores y añade el código a `supportedLanguages` y al selector de idioma de `main.go`.

---

<a id="contribuir"></a>
## 🤝 Contribuir

¡Las contribuciones son bienvenidas! Si quieres mejorar CloudMount Wizard:

1. Haz un **fork** del proyecto
2. Crea una rama para tu cambio (`git checkout -b feature/MiMejora`)
3. Haz commit de tus cambios (`git commit -m 'Añade MiMejora'`)
4. Sube la rama (`git push origin feature/MiMejora`)
5. Abre un **pull request**

### Áreas de mejora

- [x] Paquete en AUR
- [x] Traducción al inglés y al euskera
- [ ] Más idiomas (¡se agradecen traducciones!)
- [ ] Soporte para más proveedores de nube
- [ ] Sincronización bidireccional
- [ ] Indicadores de velocidad de transferencia en tiempo real

---

<a id="reportar-problemas"></a>
## 🐛 Reportar problemas

Si encuentras algún fallo o tienes una sugerencia, [abre un issue](https://github.com/AnabasaSoft/CloudMount-Wizard/issues) en GitHub. Si tienes un problema al montar una unidad, adjunta su log (lo encontrarás en el visor de logs).

También puedes escribirnos a: **anabasasoft@gmail.com**

---

<a id="licencia"></a>
## 📄 Licencia

Este proyecto está bajo la licencia MIT. Consulta el fichero [LICENSE](LICENSE) para más detalles.

---

## 🙏 Agradecimientos

- [Rclone](https://rclone.org/): el motor que lo hace todo posible
- [Fyne](https://fyne.io/): framework de interfaz multiplataforma para Go
- [MEGAcmd](https://mega.io/cmd): cliente oficial de MEGA
- La comunidad open source, por su apoyo y sus contribuciones

---

<div align="center">

<img src="https://raw.githubusercontent.com/AnabasaSoft/CloudMount-Wizard/main/AnabasaSoft.jpg" alt="Anabasa Software" width="120"/>

**Desarrollado con ❤️ por [Anabasa Software](https://anabasasoft.github.io)**

📧 Email: [anabasasoft@gmail.com](mailto:anabasasoft@gmail.com) • 🌐 Portafolio: [anabasasoft.github.io](https://anabasasoft.github.io)

⭐ Si te gusta este proyecto, dale una estrella en GitHub

</div>

<div align="center">
  <br/>
  <p><code>>_ sudo buy-me-a-coffee --theme=dark --force</code></p>
  <a href="https://www.buymeacoffee.com/danitxu" target="_blank">
    <img src="https://cdn.buymeacoffee.com/buttons/v2/default-black.png" alt="Buy Me A Coffee" style="height: 50px !important;width: 180px !important; box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;-webkit-box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;">
  </a>
  <br/>
</div>
