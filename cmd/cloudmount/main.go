package main

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"
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
)

var (
	quotaCache = make(map[string]*rclone.Quota)
	quotaMutex sync.RWMutex
)

func main() {
	minimizedFlag := flag.Bool("minimized", false, "Iniciar minimizado")
	flag.Parse()

	myApp := app.NewWithID("com.anabasasoft.cloudmount")
	myApp.SetIcon(resourceIconPng)
	myApp.Settings().SetTheme(&myTheme{})

	// Persistencia Mega
	go mega.EnsureDaemon()

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

			// Automontaje en paralelo
			for _, rName := range remotes {
				opts := settings.GetOptions(rName)
				if opts.MountOnStart {
					// Lanzar cada montaje en su propia goroutine
					go func(name string) {
						// Preparacion especial para Mega
						if name == "Mega" {
							_ = mega.EnsureDaemon()
							time.Sleep(300 * time.Millisecond)
							_, _ = mega.GetWebDAVURL()
						}

						// Montar
						_, _ = rclone.MountRemote(name)
					}(rName)

					// Pequeña pausa entre inicios
					time.Sleep(150 * time.Millisecond)
				}
			}

			// Refrescar UI después de 2 segundos para mostrar estados actualizados
			time.Sleep(2 * time.Second)
			fyne.Do(func() {
				ShowDashboard(myWindow)
			})
		}()
	} else {
		// Rclone no instalado
		content := container.NewVBox(
			widget.NewLabelWithStyle("Rclone no encontrado", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					     widget.NewLabel("Se requiere Rclone para usar esta aplicacion."),
					     widget.NewButton("Instalar Rclone", func() {
						     // Aqui iria la logica de instalacion
						     dialog.ShowInformation("Info", "Funcion de instalacion pendiente", myWindow)
					     }),
		)
		myWindow.SetContent(container.NewCenter(content))
	}

	if *minimizedFlag {
		myApp.Run()
	} else {
		myWindow.ShowAndRun()
	}
}

// ShowLogViewer muestra la ventana de logs
func ShowLogViewer(w fyne.Window) {
	logContent := widget.NewMultiLineEntry()
	logContent.Wrapping = fyne.TextWrapOff
	logContent.TextStyle = fyne.TextStyle{Monospace: true}
	logContent.SetMinRowsVisible(20)

	logPath := rclone.GetLogFilePath()

	var timer *time.Timer
	var readAndShowLogs func()

	readAndShowLogs = func() {
		defer func() {
			timer = time.AfterFunc(1000*time.Millisecond, readAndShowLogs)
		}()

		content, err := os.ReadFile(logPath)
		if err != nil {
			msg := "Esperando logs..."
			if !os.IsNotExist(err) {
				msg = fmt.Sprintf("Error leyendo logs: %v", err)
			}
			fyne.Do(func() { logContent.SetText(msg) })
			return
		}

		lines := strings.Split(string(content), "\n")
		start := 0
		if len(lines) > 300 {
			start = len(lines) - 300
		}
		display := strings.Join(lines[start:], "\n")

		fyne.Do(func() {
			currentText := logContent.Text
			if display != currentText {
				logContent.SetText(display)
				logContent.Refresh()
				logContent.CursorRow = len(lines)
			}
		})
	}

	readAndShowLogs()

	logWindow := fyne.CurrentApp().NewWindow("Visor de Logs")
	logWindow.SetContent(container.NewBorder(
		widget.NewLabelWithStyle("Ruta: "+logPath, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
						 nil, nil, nil,
					  container.NewPadded(container.NewVScroll(logContent)),
	))
	logWindow.Resize(fyne.NewSize(800, 600))

	logWindow.SetOnClosed(func() {
		if timer != nil {
			timer.Stop()
		}
	})

	logWindow.Show()
}

// ShowDashboard muestra la lista de unidades y herramientas
func ShowDashboard(w fyne.Window) {
	// Cabecera y herramientas globales
	title := widget.NewLabelWithStyle("Mis Unidades", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	addBtn := widget.NewButtonWithIcon("Nueva", theme.ContentAddIcon(), func() { ShowCloudSelection(w) })
	logBtn := widget.NewButtonWithIcon("Logs", theme.VisibilityIcon(), func() { ShowLogViewer(w) })
	configBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() { ShowGlobalSettings(w) })

	listContainer := container.NewVBox()

	// Obtener lista de nubes
	remotes, _ := rclone.ListRemotes()

	// Generar tarjetas para cada nube
	for _, rName := range remotes {
		name := rName
		mountPath := rclone.GetMountPath(name)
		isMounted := rclone.IsMounted(mountPath)
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
		} else if isMega && mega.IsLoggedIn() {
			statusTxt = "SESION OK"
			statusIcon = theme.InfoIcon()
			go mega.GetWebDAVURL()
		}

		// Calculo de espacio (asincrono)
		quotaTxt := binding.NewString()
		quotaTxt.Set("...")
		quotaVal := binding.NewFloat()

		if isMounted || (isMega && mega.IsLoggedIn()) {
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
					fyne.Do(func() { quotaTxt.Set("Calculando...") })
				}
			}()
		}

		// Botones de accion
		btnMount := widget.NewButton("Montar Disco", func() {
			go func() {
				if isMega {
					mega.EnsureDaemon()
					_, _ = mega.GetWebDAVURL()
				}
				rclone.MountRemote(name)
				fyne.Do(func() { ShowDashboard(w) })
			}()
		})

		btnUnmount := widget.NewButton("Desmontar", func() {
			go func() {
				rclone.UnmountRemote(name)
				fyne.Do(func() { ShowDashboard(w) })
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

			checkAutoInfo := widget.NewCheck("Automontar al inicio", nil)
			checkAutoInfo.Checked = opts.MountOnStart
			checkAutoInfo.Disable()

			items := []*widget.FormItem{
				widget.NewFormItem("Solo Lectura:", checkRead),
							widget.NewFormItem("Limite Cache:", entryCache),
							widget.NewFormItem("Ancho Banda:", entryBw),
							widget.NewFormItem("Estado:", checkAutoInfo),
			}

			d := dialog.NewForm("Ajustes "+displayName, "Guardar", "Cancelar", items, func(ok bool) {
				if ok {
					currentOpts := settings.GetOptions(name)
					settings.SetOptions(name, settings.RemoteOptions{
						ReadOnly:     checkRead.Checked,
						CacheSize:    entryCache.Text,
						BwLimit:      entryBw.Text,
						MountOnStart: currentOpts.MountOnStart,
					})

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
						if isMounted {
							rclone.UnmountRemote(name)
						}
						if isMega {
							mega.Logout()
						}
						rclone.DeleteRemote(name)
						fyne.Do(func() { ShowDashboard(w) })
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
			dialog.ShowConfirm("Exito", "Cuenta '"+remoteName+"' guardada.\nMontar ahora?", func(ok bool) {
				if ok {
					go func() {
						rclone.MountRemote(remoteName)
						fyne.Do(func() { ShowDashboard(w) })
					}()
				} else {
					ShowDashboard(w)
				}
			}, w)
		} else if strings.HasPrefix(val, "ERROR:") {
			fyne.Do(func() { dialog.ShowError(apiError(val[6:]), w) })
		}
	}))

	configureMega := func() {
		if !system.CheckMegaCmd() {
			dialog.ShowConfirm("Instalar", "Se necesita MEGAcmd.\nInstalar automaticamente?", func(ok bool) {
				if ok {
					w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewLabel("Instalando MEGAcmd..."), widget.NewProgressBarInfinite(), layout.NewSpacer()))
					go func() {
						err := system.InstallMegaCmd()
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
						fyne.Do(func() {
							ShowCloudSelection(w)
							dialog.ShowError(fmt.Errorf("Login fallo: %v", err), w)
						})
						return
					}
					webdavURL, errUrl := mega.GetWebDAVURL()
					if errUrl != nil {
						fyne.Do(func() {
							ShowCloudSelection(w)
							dialog.ShowError(fmt.Errorf("Error puente: %v", errUrl), w)
						})
						return
					}
					opts := map[string]string{
						"url":    webdavURL,
						"vendor": "other",
						"user":   strings.TrimSpace(entryUser.Text),
				    "pass":   strings.TrimSpace(entryPass.Text),
					}
					rclone.CreateConfigWithOpts("Mega", "webdav", opts)

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

	configureManual := func(title, provider string) {
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
					"vendor": "other",
				}
				if provider == "nextcloud" {
					opts["vendor"] = "nextcloud"
				}
				go func() {
					if err := rclone.CreateConfigWithOpts(entryName.Text, provider, opts); err != nil {
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
				       widget.NewButtonWithIcon("Nextcloud", theme.ComputerIcon(), func() { configureManual("Nextcloud", "webdav") }),
				       widget.NewButtonWithIcon("WebDAV", theme.FileIcon(), func() { configureManual("WebDAV", "webdav") }),
				       widget.NewButtonWithIcon("S3 / AWS", theme.SettingsIcon(), configureS3),
				       widget.NewSeparator(),
				       widget.NewButtonWithIcon("Volver", theme.CancelIcon(), func() { ShowDashboard(w) }),
	)

	w.SetContent(container.NewBorder(nil, nil, nil, nil, container.NewPadded(container.NewVScroll(cloudList))))
}

func apiError(msg string) error { return fmt.Errorf(msg) }

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
func (m myTheme) Font(s fyne.TextStyle) fyne.Resource    { return theme.DefaultTheme().Font(s) }
func (m myTheme) Size(n fyne.ThemeSizeName) float32      { return theme.DefaultTheme().Size(n) }

func ShowGlobalSettings(parent fyne.Window) {
	w := fyne.CurrentApp().NewWindow("Preferencias")
	w.Resize(fyne.NewSize(400, 300))

	lblState := widget.NewLabel("Estado: Desconocido")

	isAutostart := system.IsAutostartEnabled()

	checkAuto := widget.NewCheck("Arrancar al iniciar sesion", nil)
	checkAuto.Checked = isAutostart

	checkMin := widget.NewCheck("Iniciar minimizado (silencioso)", nil)
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
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Exito", "Configuracion de inicio actualizada.", w)
			w.Close()
		}
	})

	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle("Configuracion del Sistema", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				       widget.NewSeparator(),
				       widget.NewLabel("Comportamiento de arranque:"),
				       checkAuto,
				checkMin,
				widget.NewSeparator(),
				       lblState,
				layout.NewSpacer(),
				       btnSave,
	))

	w.Show()
}
