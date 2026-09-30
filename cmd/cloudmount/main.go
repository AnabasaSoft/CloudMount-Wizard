package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/anabasasoft/cloudmount-wizard/internal/mega"
	"github.com/anabasasoft/cloudmount-wizard/internal/rclone"
	"github.com/anabasasoft/cloudmount-wizard/internal/settings"
	"github.com/anabasasoft/cloudmount-wizard/internal/system"
	"github.com/anabasasoft/cloudmount-wizard/internal/update"
)

// version la inyecta el workflow de release con -ldflags "-X main.version=1.2.3".
// En las compilaciones locales vale "dev" y no se buscan actualizaciones.
var version = "dev"

func main() {
	minimizedFlag := flag.Bool("minimized", false, "Iniciar minimizado")
	flag.Parse()

	setupAppLog()
	log.Printf("CloudMount %s iniciado (minimizado: %v)", version, *minimizedFlag)

	myApp := app.NewWithID("com.anabasasoft.cloudmount")
	myApp.SetIcon(resourceIconPng)
	myApp.Settings().SetTheme(&myTheme{})

	myWindow := myApp.NewWindow("CloudMount Wizard")
	myWindow.Resize(fyne.NewSize(850, 650))

	if desk, ok := myApp.(desktop.App); ok {
		m := fyne.NewMenu("CloudMount",
			fyne.NewMenuItem("Mostrar Panel", func() {
				myWindow.Show()
				myWindow.RequestFocus()
			}),
			fyne.NewMenuItem("Salir", func() { myApp.Quit() }),
		)
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	myWindow.SetCloseIntercept(func() { myWindow.Hide() })

	if system.CheckRclone() {
		// Mostrar dashboard inmediatamente
		ShowDashboard(myWindow)

		// Ejecutar automontaje en segundo plano SIN BLOQUEAR
		go func() {
			// Pequeña pausa para que la UI se renderice primero
			time.Sleep(300 * time.Millisecond)

			// Obtener lista de remotes
			remotes, err := rclone.ListRemotes()
			if err != nil {
				return // Si falla, no pasa nada
			}

			// Persistencia Mega: solo arrancamos su servidor si hay una unidad Mega
			for _, r := range remotes {
				if r == "Mega" {
					go mega.EnsureDaemon()
				}
			}

			// Automontaje en paralelo
			var wg sync.WaitGroup
			for _, rName := range remotes {
				opts := settings.GetOptions(rName)
				if opts.MountOnStart {
					// Lanzar cada montaje en su propia goroutine
					wg.Add(1)
					go func(name string) {
						defer wg.Done()
						_, _ = mountRemote(name) // mountRemote ya deja el resultado en el log
					}(rName)

					// Pequeña pausa entre inicios
					time.Sleep(150 * time.Millisecond)
				}
			}

			// Refrescar UI cuando hayan terminado todos los montajes
			wg.Wait()
			fyne.Do(func() {
				ShowDashboard(myWindow)
			})
		}()
	} else {
		// Rclone no instalado
		content := container.NewVBox(
			widget.NewLabelWithStyle("Rclone no encontrado", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Se requiere Rclone para usar esta aplicacion."),
			widget.NewButton("Instalar Rclone", func() { installRclone(myWindow) }),
		)
		myWindow.SetContent(container.NewCenter(content))
	}

	go startUpdateChecks(myApp, myWindow)

	if *minimizedFlag {
		myApp.Run()
	} else {
		myWindow.ShowAndRun()
	}
}

// skippedVersionKey guarda en las preferencias la versión que el usuario pidió no volver a avisar
const skippedVersionKey = "update_skipped_version"

// startUpdateChecks busca versiones nuevas en GitHub al arrancar y después una vez al
// día, porque la app puede pasar días abierta en la bandeja del sistema
func startUpdateChecks(a fyne.App, w fyne.Window) {
	if version == "dev" {
		log.Printf("Compilación de desarrollo: no se buscan actualizaciones")
		return
	}

	time.Sleep(10 * time.Second) // Dejamos que termine antes el automontaje
	notified := ""               // Para no repetir el aviso de la misma versión en esta sesión
	for {
		rel, err := update.CheckLatest(version)
		switch {
		case err != nil:
			log.Printf("No se pudo comprobar si hay actualizaciones: %v", err)
		case rel != nil && rel.Tag != notified && rel.Tag != a.Preferences().String(skippedVersionKey):
			notified = rel.Tag
			log.Printf("Nueva versión disponible: %s", rel.Tag)
			// Por si la app está minimizada en la bandeja y no se ve la ventana
			a.SendNotification(fyne.NewNotification("CloudMount Wizard",
				"Nueva versión disponible: "+rel.Tag))
			fyne.Do(func() { showUpdateDialog(w, rel, true) })
		}
		time.Sleep(24 * time.Hour)
	}
}

// showUpdateDialog avisa de una versión nueva con el enlace a su release.
// Con allowSkip se ofrece no volver a avisar de esa versión.
func showUpdateDialog(w fyne.Window, rel *update.Release, allowSkip bool) {
	releaseURL, err := url.Parse(rel.URL)
	if err != nil {
		return
	}
	msg := widget.NewLabel(fmt.Sprintf("Hay una nueva versión de CloudMount Wizard: %s\n(tienes la %s).", rel.Tag, version))
	content := container.NewVBox(msg, widget.NewHyperlink("Ver la release en GitHub", releaseURL))

	checkSkip := widget.NewCheck("No volver a avisar de esta versión", nil)
	if allowSkip {
		content.Add(checkSkip)
	}

	dialog.ShowCustomConfirm("Nueva versión disponible", "Descargar", "Más tarde", content, func(ok bool) {
		if checkSkip.Checked {
			fyne.CurrentApp().Preferences().SetString(skippedVersionKey, rel.Tag)
		}
		if ok {
			if err := fyne.CurrentApp().OpenURL(releaseURL); err != nil {
				dialog.ShowError(err, w)
			}
		}
	}, w)
}

// mountRemote monta una unidad; en Mega arranca antes el servidor y su WebDAV local
func mountRemote(name string) (string, error) {
	if name == "Mega" {
		if err := mega.EnsureDaemon(); err != nil {
			log.Printf("Mega: %v", err)
		}
		if _, err := mega.GetWebDAVURL(); err != nil {
			log.Printf("Mega: %v", err)
		}
	}
	mountPoint, err := rclone.MountRemote(name)
	if err != nil {
		log.Printf("Error montando %s: %v", name, err)
	} else {
		log.Printf("Montado %s en %s", name, mountPoint)
	}
	return mountPoint, err
}

// setupAppLog envía los eventos de la propia app (montajes, altas, errores...) a
// cloudmount.log, que es el log "Global" del visor. Los mensajes de rclone de
// cada unidad van a su propio fichero. Si pasa de 1 MB se guarda como .old.
func setupAppLog() {
	path := rclone.GetLogFilePath("")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	if info, err := os.Stat(path); err == nil && info.Size() > 1024*1024 {
		os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return // Sin fichero, el log sigue saliendo por stderr
	}
	log.SetOutput(f)
}

// installRclone instala rclone y, si todo va bien, pasa directamente al panel
func installRclone(w fyne.Window) {
	w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewLabel("Instalando Rclone..."), widget.NewProgressBarInfinite(), layout.NewSpacer()))
	go func() {
		err := system.InstallRclone()
		if err != nil {
			log.Printf("Error instalando Rclone: %v", err)
		}
		fyne.Do(func() {
			if system.CheckRclone() {
				ShowDashboard(w)
				return
			}
			content := container.NewVBox(
				widget.NewLabelWithStyle("Rclone no encontrado", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Instala Rclone y vuelve a abrir la aplicacion."),
				widget.NewButton("Reintentar", func() { installRclone(w) }),
			)
			w.SetContent(container.NewCenter(content))
			if err != nil {
				dialog.ShowError(err, w)
			}
		})
	}()
}

// readTail devuelve como mucho los últimos maxBytes del fichero, empezando en una
// línea completa, para no leer entero un log de varios MB cada segundo
func readTail(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	offset := info.Size() - maxBytes
	if offset <= 0 {
		return io.ReadAll(f)
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	// Descartamos la primera línea, que casi seguro está cortada
	if i := bytes.IndexByte(content, '\n'); i >= 0 {
		content = content[i+1:]
	}
	return content, nil
}

// ShowLogViewer muestra la ventana de logs con selector de unidad
func ShowLogViewer() {
	logContent := widget.NewMultiLineEntry()
	logContent.Wrapping = fyne.TextWrapOff
	logContent.TextStyle = fyne.TextStyle{Monospace: true}
	logContent.SetMinRowsVisible(20)

	const globalOption = "Global (cloudmount.log)"

	// Estado actual: la ruta se cambia desde la UI y se lee desde el bucle de lectura
	var pathMu sync.Mutex
	logPath := rclone.GetLogFilePath("")

	refresh := make(chan struct{}, 1) // Fuerza una lectura inmediata al cambiar de unidad
	stop := make(chan struct{})       // Se cierra al cerrar la ventana

	// Función de lectura
	readAndShowLogs := func() {
		pathMu.Lock()
		path := logPath
		pathMu.Unlock()

		content, err := readTail(path, 256*1024)
		if err != nil {
			msg := "Esperando logs..."
			if !os.IsNotExist(err) {
				msg = fmt.Sprintf("Error leyendo logs en %s:\n%v", path, err)
			}
			// Solo actualizamos si el mensaje cambia para no parpadear
			fyne.Do(func() {
				if logContent.Text != msg {
					logContent.SetText(msg)
				}
			})
			return
		}

		lines := strings.Split(string(content), "\n")
		start := 0
		if len(lines) > 300 {
			start = len(lines) - 300 // Mostrar solo las últimas 300 líneas
		}
		display := strings.Join(lines[start:], "\n")

		fyne.Do(func() {
			currentText := logContent.Text
			if display != currentText {
				logContent.SetText(display)
				logContent.Refresh()
				// Autoscroll al final
				logContent.CursorRow = len(lines)
			}
		})
	}

	// Obtener lista de remotes para el selector
	remotes, _ := rclone.ListRemotes()
	options := []string{globalOption}
	for _, r := range remotes {
		options = append(options, r)
	}

	// Selector de archivo de log
	combo := widget.NewSelect(options, func(selected string) {
		currentRemote := ""
		if selected != globalOption {
			currentRemote = selected
		}

		// Actualizar ruta y limpiar vista
		pathMu.Lock()
		logPath = rclone.GetLogFilePath(currentRemote)
		pathMu.Unlock()
		logContent.SetText("Cargando " + selected + "...")

		// Pedir una lectura inmediata sin bloquear si ya hay una pendiente
		select {
		case refresh <- struct{}{}:
		default:
		}
	})

	// Seleccionar el primero por defecto (o Global)
	combo.SetSelectedIndex(0)

	logWindow := fyne.CurrentApp().NewWindow("Visor de Logs")

	// Layout
	header := container.NewVBox(
		widget.NewLabelWithStyle("Selecciona Unidad:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		combo,
		widget.NewSeparator(),
	)

	logWindow.SetContent(container.NewBorder(
		header,
		nil, nil, nil,
		container.NewPadded(container.NewVScroll(logContent)),
	))
	logWindow.Resize(fyne.NewSize(800, 600))

	logWindow.SetOnClosed(func() { close(stop) })

	logWindow.Show()

	// Único bucle de lectura: cada segundo o al cambiar de unidad, hasta cerrar la ventana
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			readAndShowLogs()
			select {
			case <-stop:
				return
			case <-ticker.C:
			case <-refresh:
			}
		}
	}()
}

// remoteState es el estado de una unidad, consultado fuera del hilo de la UI
type remoteState struct {
	name        string
	mounted     bool
	megaSession bool // Solo en Mega: hay sesión iniciada en MEGAcmd
}

// dashboardSeq numera cada petición de refresco para descartar resultados viejos
// si se piden varios refrescos seguidos
var dashboardSeq atomic.Uint64

// loadRemoteStates lanza los procesos externos (rclone, mega-whoami) para conocer el estado
func loadRemoteStates() []remoteState {
	remotes, _ := rclone.ListRemotes()
	states := make([]remoteState, 0, len(remotes))
	for _, name := range remotes {
		st := remoteState{name: name, mounted: rclone.IsMounted(rclone.GetMountPath(name))}
		if name == "Mega" {
			st.megaSession = mega.IsLoggedIn()
		}
		states = append(states, st)
	}
	return states
}

// ShowDashboard consulta el estado de las unidades en segundo plano y después
// pinta el panel, para no bloquear la UI con procesos externos
func ShowDashboard(w fyne.Window) {
	seq := dashboardSeq.Add(1)
	go func() {
		states := loadRemoteStates()
		fyne.Do(func() {
			if seq == dashboardSeq.Load() {
				renderDashboard(w, states)
			}
		})
	}()
}

// renderDashboard muestra la lista de unidades y herramientas
func renderDashboard(w fyne.Window, states []remoteState) {
	// Cabecera y herramientas globales
	title := widget.NewLabelWithStyle("Mis Unidades", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	addBtn := widget.NewButtonWithIcon("Nueva", theme.ContentAddIcon(), func() { ShowCloudSelection(w) })
	logBtn := widget.NewButtonWithIcon("Logs", theme.VisibilityIcon(), ShowLogViewer)
	configBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), ShowGlobalSettings)

	listContainer := container.NewVBox()

	// Generar tarjetas para cada nube
	for _, st := range states {
		name := st.name
		mountPath := rclone.GetMountPath(name)
		isMounted := st.mounted
		opts := settings.GetOptions(name)

		isMega := (name == "Mega")
		displayName := name
		if isMega {
			displayName = "MEGA (Oficial)"
		}

		// Estado visual
		statusTxt := "OFF"
		statusIcon := theme.ContentClearIcon()

		if isMounted {
			statusTxt = "MONTADO"
			statusIcon = theme.ConfirmIcon()
		} else if st.megaSession {
			statusTxt = "SESION OK"
			statusIcon = theme.InfoIcon()
		}

		// Calculo de espacio (asincrono). Sin montar ni sesión no hay datos que pedir
		quotaTxt := binding.NewString()
		quotaTxt.Set("—")
		quotaVal := binding.NewFloat()

		if isMounted || st.megaSession {
			quotaTxt.Set("Calculando...")
			go func() {
				if isMega {
					used, total, err := mega.GetSpace()
					if err == nil && total > 0 {
						fyne.Do(func() {
							quotaTxt.Set(fmt.Sprintf("%s / %s", rclone.FormatBytes(used), rclone.FormatBytes(total)))
							quotaVal.Set(float64(used) / float64(total))
						})
						return
					}
				}
				q, err := rclone.GetQuota(name)
				if err == nil && q.Total > 0 {
					fyne.Do(func() {
						quotaTxt.Set(fmt.Sprintf("%s / %s", rclone.FormatBytes(q.Used), rclone.FormatBytes(q.Total)))
						quotaVal.Set(float64(q.Used) / float64(q.Total))
					})
				} else {
					fyne.Do(func() { quotaTxt.Set("—") })
				}
			}()
		}

		// Botones de accion
		btnMount := widget.NewButton("Montar Disco", func() {
			go func() {
				_, err := mountRemote(name)
				fyne.Do(func() {
					ShowDashboard(w)
					if err != nil {
						dialog.ShowError(err, w)
					}
				})
			}()
		})

		btnUnmount := widget.NewButton("Desmontar", func() {
			go func() {
				err := rclone.UnmountRemote(name)
				if err != nil {
					log.Printf("Error desmontando %s: %v", name, err)
				} else {
					log.Printf("Desmontado %s", name)
				}
				fyne.Do(func() {
					ShowDashboard(w)
					if err != nil {
						dialog.ShowError(err, w)
					}
				})
			}()
		})

		btnOpen := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
			rclone.OpenFileManager(mountPath)
		})

		// Estado de botones
		if isMounted {
			btnMount.Disable()
			btnUnmount.Enable()
			btnOpen.Enable()
		} else {
			btnMount.Enable()
			btnUnmount.Disable()
			btnOpen.Disable()
		}

		btnSettings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			checkRead := widget.NewCheck("Solo Lectura", nil)
			checkRead.Checked = opts.ReadOnly

			entryCache := widget.NewEntry()
			entryCache.Text = opts.CacheSize
			entryCache.PlaceHolder = "Ej: 10G"

			entryBw := widget.NewEntry()
			entryBw.Text = opts.BwLimit
			entryBw.PlaceHolder = "Ej: 2M"

			checkAutoMount := widget.NewCheck("Montar al abrir CloudMount", nil)
			checkAutoMount.Checked = opts.MountOnStart

			items := []*widget.FormItem{
				widget.NewFormItem("Solo Lectura:", checkRead),
				widget.NewFormItem("Limite Cache:", entryCache),
				widget.NewFormItem("Ancho Banda:", entryBw),
				widget.NewFormItem("Automontaje:", checkAutoMount),
			}

			d := dialog.NewForm("Ajustes "+displayName, "Guardar", "Cancelar", items, func(ok bool) {
				if ok {
					// Partimos de las opciones guardadas para no perder campos que no están en el formulario
					newOpts := settings.GetOptions(name)
					newOpts.ReadOnly = checkRead.Checked
					newOpts.CacheSize = strings.TrimSpace(entryCache.Text)
					newOpts.BwLimit = strings.TrimSpace(entryBw.Text)
					newOpts.MountOnStart = checkAutoMount.Checked
					if err := settings.SetOptions(name, newOpts); err != nil {
						dialog.ShowError(err, w)
						return
					}

					if isMounted {
						dialog.ShowInformation("Cambios", "Desmonta y monta la unidad para aplicar los limites.", w)
					} else {
						ShowDashboard(w)
					}
				}
			}, w)
			d.Resize(fyne.NewSize(400, 350))
			d.Show()
		})

		btnDelete := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			msg := "Eliminar configuracion de " + displayName + "?"
			if isMega {
				msg = "Cerrar sesion y eliminar Mega?"
			}
			dialog.ShowConfirm("Borrar", msg, func(ok bool) {
				if ok {
					go func() {
						if isMega {
							mega.Logout()
						}
						err := rclone.DeleteRemote(name)
						if err != nil {
							log.Printf("Error eliminando %s: %v", name, err)
						} else {
							log.Printf("Eliminada la unidad %s", name)
						}
						fyne.Do(func() {
							ShowDashboard(w)
							if err != nil {
								dialog.ShowError(err, w)
							}
						})
					}()
				}
			}, w)
		})

		// Ensamblaje de la tarjeta
		cardContent := container.NewVBox(
			container.NewHBox(
				widget.NewIcon(statusIcon),
				widget.NewLabelWithStyle(displayName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				layout.NewSpacer(),
				widget.NewLabel(statusTxt),
			),
			widget.NewSeparator(),
			container.NewBorder(nil, nil, widget.NewLabelWithData(quotaTxt), nil, widget.NewProgressBarWithData(quotaVal)),
			widget.NewSeparator(),
			container.NewHBox(btnMount, btnUnmount, btnOpen, layout.NewSpacer(), btnSettings, btnDelete),
		)

		listContainer.Add(widget.NewCard("", "", cardContent))
	}

	if len(listContainer.Objects) == 0 {
		listContainer.Add(widget.NewLabel("No hay unidades configuradas. Pulsa 'Nueva' para empezar."))
	}

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(title, layout.NewSpacer(), logBtn, configBtn, addBtn),
			widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(container.NewVScroll(listContainer)),
	)

	w.SetContent(content)
}

// ShowCloudSelection pantalla de seleccion
func ShowCloudSelection(w fyne.Window) {
	configState := binding.NewString()
	configState.Set("IDLE")

	configState.AddListener(binding.NewDataListener(func() {
		val, _ := configState.Get()
		if strings.HasPrefix(val, "DONE:") {
			remoteName := val[5:]
			log.Printf("Unidad %s creada", remoteName)
			dialog.ShowConfirm("Exito", "Cuenta '"+remoteName+"' guardada.\nMontar ahora?", func(ok bool) {
				if ok {
					go func() {
						_, err := mountRemote(remoteName)
						fyne.Do(func() {
							ShowDashboard(w)
							if err != nil {
								dialog.ShowError(err, w)
							}
						})
					}()
				} else {
					ShowDashboard(w)
				}
			}, w)
		} else if strings.HasPrefix(val, "ERROR:") {
			log.Printf("Error creando unidad: %s", val[6:])
			// Volvemos a la lista: configureOAuth deja la ventana en "Autorizando..." sin botones
			ShowCloudSelection(w)
			dialog.ShowError(errors.New(val[6:]), w)
		}
	}))

	configureMega := func() {
		if !system.CheckMegaCmd() {
			dialog.ShowConfirm("Instalar", "Se necesita MEGAcmd.\nInstalar automaticamente?", func(ok bool) {
				if ok {
					w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewLabel("Instalando MEGAcmd..."), widget.NewProgressBarInfinite(), layout.NewSpacer()))
					go func() {
						err := system.InstallMegaCmd()
						if err != nil {
							log.Printf("Error instalando MEGAcmd: %v", err)
						}
						fyne.Do(func() {
							if err != nil {
								ShowCloudSelection(w)
								dialog.ShowError(err, w)
							} else {
								ShowCloudSelection(w)
								dialog.ShowInformation("Instalado", "Vuelve a conectar.", w)
							}
						})
					}()
				}
			}, w)
			return
		}

		entryUser := widget.NewEntry()
		entryUser.PlaceHolder = "Email"
		entryPass := widget.NewPasswordEntry()
		entryPass.PlaceHolder = "Contraseña"
		entry2FA := widget.NewEntry()
		entry2FA.PlaceHolder = "Codigo 2FA"

		d := dialog.NewForm("Conectar Mega", "Login", "Cancelar", []*widget.FormItem{
			widget.NewFormItem("Email:", entryUser),
			widget.NewFormItem("Pass:", entryPass),
			widget.NewFormItem("2FA:", entry2FA),
		}, func(ok bool) {
			if ok {
				w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewLabel("Conectando..."), widget.NewProgressBarInfinite(), layout.NewSpacer()))
				go func() {
					err := mega.Login(strings.TrimSpace(entryUser.Text), strings.TrimSpace(entryPass.Text), strings.TrimSpace(entry2FA.Text))
					if err != nil {
						log.Printf("Error en login de Mega: %v", err)
						fyne.Do(func() {
							ShowCloudSelection(w)
							dialog.ShowError(fmt.Errorf("Login fallo: %v", err), w)
						})
						return
					}
					webdavURL, errUrl := mega.GetWebDAVURL()
					if errUrl != nil {
						log.Printf("Error activando WebDAV de Mega: %v", errUrl)
						fyne.Do(func() {
							ShowCloudSelection(w)
							dialog.ShowError(fmt.Errorf("Error puente: %v", errUrl), w)
						})
						return
					}
					// El WebDAV local de MEGAcmd no pide autenticación: no guardamos
					// el email ni la contraseña en rclone.conf
					opts := map[string]string{
						"url":    webdavURL,
						"vendor": "other",
					}
					if err := rclone.CreateConfigWithOpts("Mega", "webdav", opts); err != nil {
						log.Printf("Error guardando configuracion de Mega: %v", err)
						fyne.Do(func() {
							ShowCloudSelection(w)
							dialog.ShowError(fmt.Errorf("Error guardando configuracion: %v", err), w)
						})
						return
					}

					log.Printf("Mega configurado")
					fyne.Do(func() {
						dialog.ShowInformation("Conectado", "Mega configurado.", w)
						ShowDashboard(w)
					})
				}()
			}
		}, w)
		d.Resize(fyne.NewSize(450, 300))
		d.Show()
	}

	configureOAuth := func(name, provider string) {
		input := widget.NewEntry()
		input.PlaceHolder = "Nombre"
		dialog.ShowCustomConfirm("Configurar "+name, "Ok", "Cancel", input, func(ok bool) {
			if ok && input.Text != "" {
				w.SetContent(widget.NewLabel("Autorizando..."))
				go func() {
					if err := rclone.CreateConfig(input.Text, provider); err != nil {
						configState.Set("ERROR:" + err.Error())
					} else {
						configState.Set("DONE:" + input.Text)
					}
				}()
			}
		}, w)
	}

	// configureManual da de alta un remote WebDAV; vendor es "nextcloud" u "other"
	configureManual := func(title, vendor string) {
		entryName := widget.NewEntry()
		entryURL := widget.NewEntry()
		entryURL.PlaceHolder = "https://..."
		entryUser := widget.NewEntry()
		entryPass := widget.NewPasswordEntry()
		d := dialog.NewForm(title, "Ok", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Nombre:", entryName),
			widget.NewFormItem("URL:", entryURL),
			widget.NewFormItem("User:", entryUser),
			widget.NewFormItem("Pass:", entryPass),
		}, func(ok bool) {
			if ok {
				opts := map[string]string{
					"url":    entryURL.Text,
					"user":   entryUser.Text,
					"pass":   entryPass.Text,
					"vendor": vendor,
				}
				go func() {
					if err := rclone.CreateConfigWithOpts(entryName.Text, "webdav", opts); err != nil {
						configState.Set("ERROR:" + err.Error())
					} else {
						configState.Set("DONE:" + entryName.Text)
					}
				}()
			}
		}, w)
		d.Resize(fyne.NewSize(500, 350))
		d.Show()
	}

	configureS3 := func() {
		entryName := widget.NewEntry()
		entryProvider := widget.NewSelect([]string{"AWS", "Minio", "Wasabi", "Other"}, nil)
		entryAccess := widget.NewEntry()
		entrySecret := widget.NewPasswordEntry()
		entryEndpoint := widget.NewEntry()
		d := dialog.NewForm("Configurar S3", "Ok", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Nombre:", entryName),
			widget.NewFormItem("Prov:", entryProvider),
			widget.NewFormItem("Access:", entryAccess),
			widget.NewFormItem("Secret:", entrySecret),
			widget.NewFormItem("Endpoint:", entryEndpoint),
		}, func(ok bool) {
			if ok {
				opts := map[string]string{
					"provider":          entryProvider.Selected,
					"env_auth":          "false",
					"access_key_id":     entryAccess.Text,
					"secret_access_key": entrySecret.Text,
				}
				if entryEndpoint.Text != "" {
					opts["endpoint"] = entryEndpoint.Text
				}
				go func() {
					if err := rclone.CreateConfigWithOpts(entryName.Text, "s3", opts); err != nil {
						configState.Set("ERROR:" + err.Error())
					} else {
						configState.Set("DONE:" + entryName.Text)
					}
				}()
			}
		}, w)
		d.Resize(fyne.NewSize(500, 400))
		d.Show()
	}

	cloudList := container.NewVBox(
		widget.NewLabelWithStyle("Populares", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewButtonWithIcon("Mega.nz (Oficial)", theme.UploadIcon(), configureMega),
		widget.NewButtonWithIcon("Google Drive", theme.StorageIcon(), func() { configureOAuth("Google Drive", "drive") }),
		widget.NewButtonWithIcon("Dropbox", theme.ContentAddIcon(), func() { configureOAuth("Dropbox", "dropbox") }),
		widget.NewButtonWithIcon("OneDrive", theme.FolderIcon(), func() { configureOAuth("OneDrive", "onedrive") }),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Avanzado", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewButtonWithIcon("pCloud", theme.StorageIcon(), func() { configureOAuth("pCloud", "pcloud") }),
		widget.NewButtonWithIcon("Box", theme.ContentCopyIcon(), func() { configureOAuth("Box", "box") }),
		widget.NewButtonWithIcon("Nextcloud", theme.ComputerIcon(), func() { configureManual("Nextcloud", "nextcloud") }),
		widget.NewButtonWithIcon("WebDAV", theme.FileIcon(), func() { configureManual("WebDAV", "other") }),
		widget.NewButtonWithIcon("S3 / AWS", theme.SettingsIcon(), configureS3),
		widget.NewSeparator(),
		widget.NewButtonWithIcon("Volver", theme.CancelIcon(), func() { ShowDashboard(w) }),
	)

	w.SetContent(container.NewBorder(nil, nil, nil, nil, container.NewPadded(container.NewVScroll(cloudList))))
}

type myTheme struct{}

var _ fyne.Theme = (*myTheme)(nil)

func (m myTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x18, G: 0x18, B: 0x18, A: 0xFF}
	case theme.ColorNameOverlayBackground, theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x25, G: 0x25, B: 0x25, A: 0xFF}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xFF}
	}
	return theme.DefaultTheme().Color(n, v)
}
func (m myTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }
func (m myTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (m myTheme) Size(n fyne.ThemeSizeName) float32       { return theme.DefaultTheme().Size(n) }

func ShowGlobalSettings() {
	w := fyne.CurrentApp().NewWindow("Preferencias")
	w.Resize(fyne.NewSize(400, 300))

	lblState := widget.NewLabel("Estado: Desconocido")

	isAutostart := system.IsAutostartEnabled()

	checkAuto := widget.NewCheck("Arrancar al iniciar sesion", nil)
	checkAuto.Checked = isAutostart

	checkMin := widget.NewCheck("Iniciar minimizado (silencioso)", nil)
	checkMin.Checked = isAutostart && system.IsAutostartMinimized()
	checkMin.Disable()

	if isAutostart {
		checkMin.Enable()
		lblState.SetText("Estado: Autostart ACTIVO")
	} else {
		lblState.SetText("Estado: Autostart INACTIVO")
	}

	checkAuto.OnChanged = func(checked bool) {
		if checked {
			checkMin.Enable()
		} else {
			checkMin.Disable()
			checkMin.SetChecked(false)
		}
	}

	btnSave := widget.NewButtonWithIcon("Guardar Cambios", theme.DocumentSaveIcon(), func() {
		err := system.SetAutostart(checkAuto.Checked, checkMin.Checked)
		log.Printf("Autoarranque: activo=%v minimizado=%v (error: %v)", checkAuto.Checked, checkMin.Checked, err)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Exito", "Configuracion de inicio actualizada.", w)
			w.Close()
		}
	})

	// Comprobación manual: avisa aunque se haya marcado "no volver a avisar"
	btnUpdate := widget.NewButtonWithIcon("Buscar actualizaciones", theme.ViewRefreshIcon(), nil)
	btnUpdate.OnTapped = func() {
		if version == "dev" {
			dialog.ShowInformation("Actualizaciones", "Esta es una compilación de desarrollo.", w)
			return
		}
		btnUpdate.Disable()
		go func() {
			rel, err := update.CheckLatest(version)
			fyne.Do(func() {
				btnUpdate.Enable()
				switch {
				case err != nil:
					dialog.ShowError(fmt.Errorf("No se pudo comprobar: %v", err), w)
				case rel == nil:
					dialog.ShowInformation("Actualizaciones", "Tienes la última versión ("+version+").", w)
				default:
					showUpdateDialog(w, rel, false)
				}
			})
		}()
	}

	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle("Configuracion del Sistema", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Comportamiento de arranque:"),
		checkAuto,
		checkMin,
		widget.NewSeparator(),
		lblState,
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel("Versión: "+version), layout.NewSpacer(), btnUpdate),
		layout.NewSpacer(),
		btnSave,
	))

	w.Show()
}
