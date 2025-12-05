/*
 * CloudMount Wizard - GUI for Rclone on Linux
 * Copyright (C) 2024 Anabasa Software
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package main

import (
	"fmt"
	"image/color" // <--- NUEVO IMPORT NECESARIO PARA EL TEMA
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/anabasasoft/cloudmount-wizard/internal/rclone"
	"github.com/anabasasoft/cloudmount-wizard/internal/settings"
	"github.com/anabasasoft/cloudmount-wizard/internal/system"
)

// Caché de espacio
var (
	quotaCache = make(map[string]*rclone.Quota)
	quotaMutex sync.RWMutex
)

func main() {
	myApp := app.NewWithID("com.anabasasoft.cloudmount")
	myApp.SetIcon(resourceIconPng)

	// --- APLICAMOS EL TEMA GRIS ---
	myApp.Settings().SetTheme(&myTheme{})

	myWindow := myApp.NewWindow("CloudMount Wizard")
	myWindow.Resize(fyne.NewSize(850, 650))

	// SYSTEM TRAY
	if desk, ok := myApp.(desktop.App); ok {
		m := fyne.NewMenu("CloudMount",
				  fyne.NewMenuItem("Mostrar Panel", func() {
					  myWindow.Show()
					  myWindow.RequestFocus()
				  }),
		)
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})

	// BINDINGS
	statusText := binding.NewString()
	statusText.Set("Verificando sistema...")
	detailText := binding.NewString()
	detailText.Set("Por favor, espera un momento.")

	// UI HEADER
	title := widget.NewLabelWithStyle("CloudMount", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewVBox(
		container.NewHBox(widget.NewIcon(theme.StorageIcon()), title),
				    widget.NewSeparator(),
	)

	statusLabel := widget.NewLabelWithData(statusText)
	statusLabel.Alignment = fyne.TextAlignCenter
	detailLabel := widget.NewLabelWithData(detailText)
	detailLabel.Alignment = fyne.TextAlignCenter
	statusIcon := widget.NewIcon(theme.HelpIcon())
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()
	actionBtn := widget.NewButton("Acción", nil)
	actionBtn.Hide()

	goToDashboard := func() {
		ShowDashboard(myWindow)
	}

	statusText.AddListener(binding.NewDataListener(func() {
		val, _ := statusText.Get()
		if val == "¡Instalación completada!" {
			actionBtn.SetText("Ir a mis unidades >>")
			actionBtn.SetIcon(theme.HomeIcon())
			actionBtn.OnTapped = goToDashboard
			actionBtn.Enable()
			progressBar.Hide()
			statusIcon.SetResource(theme.ConfirmIcon())
		}
	}))

	cardContent := container.NewVBox(
		container.NewCenter(statusIcon),
					 statusLabel,
				  widget.NewSeparator(),
					 detailLabel,
				  progressBar,
				  layout.NewSpacer(),
					 actionBtn,
	)
	card := widget.NewCard("Estado del Sistema", "", cardContent)

	if system.CheckRclone() {
		ShowDashboard(myWindow)
	} else {
		statusIcon.SetResource(theme.WarningIcon())
		statusText.Set("Falta componente Rclone")
		detailText.Set("Es necesario instalar el motor de conexión.")

		actionBtn.SetText("Instalar automáticamente")
		actionBtn.SetIcon(theme.DownloadIcon())

		actionBtn.OnTapped = func() {
			actionBtn.Disable()
			progressBar.Show()
			statusText.Set("Instalando...")

			go func() {
				err := system.InstallRclone()
				if err != nil {
					statusText.Set("Error en la instalación")
					detailText.Set(err.Error())
				} else {
					statusText.Set("¡Instalación completada!")
				}
			}()
		}
		actionBtn.Show()

		myWindow.SetContent(container.NewPadded(container.NewBorder(
			header, nil, nil, nil, container.NewPadded(card),
		)))
	}

	myWindow.ShowAndRun()
}

// ShowDashboard LISTA LAS UNIDADES
func ShowDashboard(w fyne.Window) {
	title := widget.NewLabelWithStyle("Mis Unidades", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	controlState := binding.NewString()
	controlState.Set("IDLE")
	errorState := binding.NewString()
	errorState.Set("")

	controlState.AddListener(binding.NewDataListener(func() {
		val, _ := controlState.Get()
		if val == "REFRESH" {
			ShowDashboard(w)
		}
	}))
	errorState.AddListener(binding.NewDataListener(func() {
		val, _ := errorState.Get()
		if val != "" && val != "CLEAN" {
			dialog.ShowError(apiError(val), w)
			errorState.Set("CLEAN")
		}
	}))

	addBtn := widget.NewButtonWithIcon("Nueva Conexión", theme.ContentAddIcon(), func() {
		ShowCloudSelection(w)
	})

	listContainer := container.NewVBox()
	remotes, err := rclone.ListRemotes()

	if err != nil {
		listContainer.Add(widget.NewLabel("Error: " + err.Error()))
	} else if len(remotes) == 0 {
		listContainer.Add(widget.NewLabel("No tienes nubes configuradas."))
	} else {
		for _, remote := range remotes {
			rName := remote
			mountPath := rclone.GetMountPath(rName)
			isMounted := rclone.IsMounted(mountPath)
			isAutomount := rclone.IsAutomountEnabled(rName)

			// Cargar opciones
			opts := settings.GetOptions(rName)

			var statusIcon fyne.Resource
			var statusTxt string
			if isMounted {
				statusIcon = theme.ConfirmIcon()
				statusTxt = "ACTIVO"
				if opts.ReadOnly {
					statusTxt += " (Lectura)"
				}
			} else {
				statusIcon = theme.ContentClearIcon()
				statusTxt = "OFF"
			}

			// --- QUOTA ---
			quotaProgressBind := binding.NewFloat()
			quotaTextBind := binding.NewString()

			quotaMutex.RLock()
			cachedQuota, hasData := quotaCache[rName]
			quotaMutex.RUnlock()

			if hasData {
				if cachedQuota != nil && cachedQuota.Total > 0 {
					percent := float64(cachedQuota.Used) / float64(cachedQuota.Total)
					quotaProgressBind.Set(percent)
					quotaTextBind.Set(fmt.Sprintf("%s / %s", rclone.FormatBytes(cachedQuota.Used), rclone.FormatBytes(cachedQuota.Total)))
				} else {
					quotaTextBind.Set("Espacio ilimitado")
				}
			} else {
				quotaTextBind.Set("Calculando...")
				go func(remote string, qVal binding.Float, qTxt binding.String) {
					q, err := rclone.GetQuota(remote)
					quotaMutex.Lock()
					quotaCache[remote] = q
					quotaMutex.Unlock()
					if err != nil {
						qTxt.Set("Info no disponible")
					} else {
						if q.Total > 0 {
							percent := float64(q.Used) / float64(q.Total)
							qVal.Set(percent)
							qTxt.Set(fmt.Sprintf("%s / %s", rclone.FormatBytes(q.Used), rclone.FormatBytes(q.Total)))
						} else {
							qTxt.Set("Espacio ilimitado")
						}
					}
				}(rName, quotaProgressBind, quotaTextBind)
			}

			quotaBar := widget.NewProgressBarWithData(quotaProgressBind)
			quotaLabel := widget.NewLabelWithData(quotaTextBind)
			quotaLabel.TextStyle = fyne.TextStyle{Italic: true}

			// --- CONTROLES ---
			btnMount := widget.NewButton("Conectar", nil)
			btnUnmount := widget.NewButton("Desconectar", nil)
			btnOpen := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), nil)
			btnDelete := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			btnRename := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)

			// --- BOTÓN DE AJUSTES (NUEVO) ---
			btnSettings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
				// Formulario de opciones
				checkRead := widget.NewCheck("Modo Solo Lectura", nil)
				checkRead.SetChecked(opts.ReadOnly)

				entryCache := widget.NewEntry()
				entryCache.SetPlaceHolder("Ej: 10G (vacío = ilimitado)")
				entryCache.SetText(opts.CacheSize)

				entryBw := widget.NewEntry()
				entryBw.SetPlaceHolder("Ej: 2M (vacío = ilimitado)")
				entryBw.SetText(opts.BwLimit)

				items := []*widget.FormItem{
					widget.NewFormItem("", checkRead),
								widget.NewFormItem("Límite Caché Disco:", entryCache),
								widget.NewFormItem("Límite Ancho Banda:", entryBw),
				}

				d := dialog.NewForm("Ajustes de "+rName, "Guardar", "Cancelar", items, func(confirm bool) {
					if confirm {
						newOpts := settings.RemoteOptions{
							ReadOnly:  checkRead.Checked,
							CacheSize: entryCache.Text,
							BwLimit:   entryBw.Text,
						}
						settings.SetOptions(rName, newOpts)

						// Si estaba en automontaje, regeneramos servicio para aplicar cambios
						if isAutomount {
							go func() {
								rclone.EnableAutomount(rName)
								controlState.Set("REFRESH")
							}()
						} else {
							// Si es manual y está montado, solo refrescamos UI (usuario debe reconectar)
							controlState.Set("REFRESH")
						}
					}
				}, w)
				d.Resize(fyne.NewSize(400, 300))
				d.Show()
			})

			checkAuto := widget.NewCheck("Automontar", nil)
			checkAuto.SetChecked(isAutomount)

			checkAuto.OnChanged = func(checked bool) {
				checkAuto.Disable()
				go func() {
					var err error
					if checked {
						err = rclone.EnableAutomount(rName)
					} else {
						err = rclone.DisableAutomount(rName)
					}
					if err != nil {
						errorState.Set("Error auto: " + err.Error())
					}
					controlState.Set("REFRESH")
				}()
			}

			if isMounted {
				btnMount.Disable()
				btnDelete.Disable()
				btnRename.Disable()

				btnUnmount.OnTapped = func() {
					btnUnmount.Disable()
					btnUnmount.SetText("...")
					go func() {
						err := rclone.UnmountRemote(rName)
						if err != nil {
							errorState.Set(err.Error())
						}
						controlState.Set("REFRESH")
					}()
				}
				btnOpen.OnTapped = func() { rclone.OpenFileManager(mountPath) }
			} else {
				btnUnmount.Disable()
				btnOpen.Disable()
				btnMount.OnTapped = func() {
					btnMount.Disable()
					btnMount.SetText("...")
					go func() {
						_, err := rclone.MountRemote(rName)
						if err != nil {
							errorState.Set("Error: " + err.Error())
						}
						controlState.Set("REFRESH")
					}()
				}
				btnDelete.OnTapped = func() {
					dialog.ShowConfirm("Borrar", "¿Eliminar '"+rName+"'?", func(confirm bool) {
						if confirm {
							go func() {
								quotaMutex.Lock()
								delete(quotaCache, rName)
								quotaMutex.Unlock()
								rclone.DeleteRemote(rName)
								controlState.Set("REFRESH")
							}()
						}
					}, w)
				}
				btnRename.OnTapped = func() {
					input := widget.NewEntry()
					input.SetText(rName)
					content := container.NewVBox(widget.NewLabel("Nuevo nombre:"), input)
					d := dialog.NewCustomConfirm("Renombrar", "Guardar", "Cancelar", content, func(confirm bool) {
						if confirm && input.Text != "" {
							newName := input.Text
							go func() {
								err := rclone.RenameRemote(rName, newName)
								quotaMutex.Lock()
								if val, ok := quotaCache[rName]; ok {
									quotaCache[newName] = val
									delete(quotaCache, rName)
								}
								quotaMutex.Unlock()
								if err != nil {
									errorState.Set(err.Error())
								} else {
									controlState.Set("REFRESH")
								}
							}()
						}
					}, w)
					d.Resize(fyne.NewSize(400, 200))
					d.Show()
				}
			}

			// DISEÑO
			topRow := container.NewHBox(
				widget.NewIcon(statusIcon),
						    widget.NewLabelWithStyle(rName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
						    layout.NewSpacer(),
						    widget.NewLabel(statusTxt),
			)

			// Fila cuota y automontar
			quotaRow := container.NewBorder(nil, nil, quotaLabel, checkAuto, quotaBar)

			// Fila inferior de botones
			// Agrupamos gestión a la derecha: Renombrar, Ajustes, Borrar
			manageBtns := container.NewHBox(btnRename, btnSettings, btnDelete)

			botRow := container.NewBorder(nil, nil, nil, manageBtns, container.NewHBox(btnMount, btnUnmount, btnOpen))

			cardContent := container.NewVBox(
				topRow,
				widget.NewSeparator(),
							 quotaRow,
				    widget.NewSeparator(),
							 botRow,
			)

			listContainer.Add(widget.NewCard("", "", cardContent))
		}
	}

	content := container.NewBorder(
		container.NewVBox(container.NewHBox(title, layout.NewSpacer(), addBtn), widget.NewSeparator()),
				       nil, nil, nil,
				container.NewPadded(container.NewVScroll(listContainer)),
	)
	w.SetContent(content)
}

// ShowCloudSelection PANTALLA DE AÑADIR
func ShowCloudSelection(w fyne.Window) {
	title := widget.NewLabelWithStyle("Añadir Nueva Nube", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	configState := binding.NewString()
	configState.Set("IDLE")

	// Listener de finalización
	configState.AddListener(binding.NewDataListener(func() {
		val, _ := configState.Get()
		if len(val) > 5 && val[:5] == "DONE:" {
			remoteName := val[5:]
			dialog.ShowConfirm("Éxito", "Cuenta '"+remoteName+"' guardada.\n¿Montar ahora?", func(mountNow bool) {
				if mountNow {
					_, err := rclone.MountRemote(remoteName)
					if err != nil {
						dialog.ShowError(err, w)
					} else {
						dialog.ShowInformation("Listo", "Unidad montada.", w)
						ShowDashboard(w)
					}
				} else {
					ShowDashboard(w)
				}
			}, w)
		} else if len(val) > 6 && val[:6] == "ERROR:" {
			// --- FIX: Usamos NewError + SetOnClosed para restaurar la UI ---
			d := dialog.NewError(apiError(val[6:]), w)
			d.SetOnClosed(func() {
				// Volvemos a mostrar la selección al cerrar el error
				ShowCloudSelection(w)
			})
			d.Show()
		}
	}))

	// 1. CONFIGURADOR OAUTH (Navegador) - Este no necesita cambio de tamaño
	configureOAuth := func(displayName, rcloneType string) {
		input := widget.NewEntry()
		input.SetPlaceHolder("Ej: Mi" + displayName)

		dialog.ShowCustomConfirm("Configurar "+displayName, "Continuar", "Cancelar", input, func(accepted bool) {
			if !accepted || input.Text == "" {
				return
			}
			name := input.Text
			w.SetContent(container.NewVBox(
				layout.NewSpacer(),
						       widget.NewLabelWithStyle("Configurando "+displayName, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
						       widget.NewLabel("Autoriza en el navegador..."),
						       widget.NewProgressBarInfinite(),
						       layout.NewSpacer(),
			))
			go func() {
				err := rclone.CreateConfig(name, rcloneType)
				if err != nil {
					configState.Set("ERROR:" + err.Error())
				} else {
					configState.Set("DONE:" + name)
				}
			}()
		}, w)
	}

	// 2. CONFIGURADOR MANUAL (WebDAV, Nextcloud) - REDIMENSIONADO
	configureManual := func(displayName, rcloneType string) {
		entryName := widget.NewEntry()
		entryURL := widget.NewEntry()
		entryURL.SetPlaceHolder("https://...")
		entryUser := widget.NewEntry()
		entryPass := widget.NewPasswordEntry()

		items := []*widget.FormItem{
			widget.NewFormItem("Nombre:", entryName),
			widget.NewFormItem("URL:", entryURL),
			widget.NewFormItem("Usuario:", entryUser),
			widget.NewFormItem("Contraseña:", entryPass),
		}

		// Usamos NewForm para poder redimensionar
		d := dialog.NewForm("Configurar "+displayName, "Guardar", "Cancelar", items, func(confirm bool) {
			if confirm {
				if entryName.Text == "" || entryURL.Text == "" {
					dialog.ShowError(apiError("Faltan datos"), w)
					return
				}
				opts := map[string]string{
					"url": entryURL.Text, "user": entryUser.Text, "pass": entryPass.Text, "vendor": "nextcloud",
				}
				if rcloneType == "webdav" {
					opts["vendor"] = "other"
				}

				w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewProgressBarInfinite(), layout.NewSpacer()))
				go func() {
					err := rclone.CreateConfigWithOpts(entryName.Text, rcloneType, opts)
					if err != nil {
						configState.Set("ERROR:" + err.Error())
					} else {
						configState.Set("DONE:" + entryName.Text)
					}
				}()
			}
		}, w)

		// Hacemos el diálogo más ancho (500px)
		d.Resize(fyne.NewSize(500, 350))
		d.Show()
	}

	// 3. CONFIGURADOR MEGA - CON SOPORTE 2FA
	configureMega := func() {
		entryName := widget.NewEntry()
		entryUser := widget.NewEntry()
		entryUser.SetPlaceHolder("email@mega.nz")
		entryPass := widget.NewPasswordEntry()

		// Nuevo campo para el código 2FA
		entry2FA := widget.NewEntry()
		entry2FA.SetPlaceHolder("Opcional: Solo si tienes 2FA activado")

		items := []*widget.FormItem{
			widget.NewFormItem("Nombre Conexión:", entryName),
			widget.NewFormItem("Email Mega:", entryUser),
			widget.NewFormItem("Contraseña:", entryPass),
			widget.NewFormItem("Código 2FA (6 dígitos):", entry2FA), // Añadimos el campo al formulario
		}

		// Usamos NewForm
		d := dialog.NewForm("Configurar Mega", "Guardar", "Cancelar", items, func(confirm bool) {
			if confirm {
				if entryName.Text == "" || entryUser.Text == "" || entryPass.Text == "" {
					dialog.ShowError(apiError("Nombre, usuario y contraseña son obligatorios"), w)
					return
				}

				// Preparamos los parámetros para Rclone
				opts := map[string]string{
					"user": entryUser.Text,
					"pass": entryPass.Text,
				}

				// SI el usuario escribió un código 2FA, lo añadimos a los parámetros.
				// Rclone usa la clave "2fa" durante la configuración para generar el token.
				if entry2FA.Text != "" {
					opts["2fa"] = entry2FA.Text
				}

				w.SetContent(container.NewVBox(
					layout.NewSpacer(),
							       widget.NewLabel("Conectando con Mega..."),
							       widget.NewProgressBarInfinite(),
							       layout.NewSpacer(),
				))

				go func() {
					// Esto ejecutará: rclone config create NOMBRE mega user=... pass=... 2fa=CODIGO
					err := rclone.CreateConfigWithOpts(entryName.Text, "mega", opts)
					if err != nil {
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

	// 4. CONFIGURADOR S3 - REDIMENSIONADO
	configureS3 := func() {
		entryName := widget.NewEntry()
		entryProvider := widget.NewSelect([]string{"AWS", "Minio", "Wasabi", "DigitalOcean", "Other"}, nil)
		entryProvider.SetSelected("Minio")
		entryAccess := widget.NewEntry()
		entrySecret := widget.NewPasswordEntry()
		entryEndpoint := widget.NewEntry()

		items := []*widget.FormItem{
			widget.NewFormItem("Nombre:", entryName),
			widget.NewFormItem("Proveedor:", entryProvider),
			widget.NewFormItem("Access Key:", entryAccess),
			widget.NewFormItem("Secret Key:", entrySecret),
			widget.NewFormItem("Endpoint:", entryEndpoint),
		}

		// Usamos NewForm
		d := dialog.NewForm("Configurar S3", "Guardar", "Cancelar", items, func(confirm bool) {
			if confirm {
				if entryName.Text == "" || entryAccess.Text == "" {
					return
				}
				opts := map[string]string{
					"provider": entryProvider.Selected, "env_auth": "false",
					"access_key_id": entryAccess.Text, "secret_access_key": entrySecret.Text,
				}
				if entryEndpoint.Text != "" {
					opts["endpoint"] = entryEndpoint.Text
				}

				w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewProgressBarInfinite(), layout.NewSpacer()))
				go func() {
					err := rclone.CreateConfigWithOpts(entryName.Text, "s3", opts)
					if err != nil {
						configState.Set("ERROR:" + err.Error())
					} else {
						configState.Set("DONE:" + entryName.Text)
					}
				}()
			}
		}, w)

		// Hacemos el diálogo más ancho (550px porque tiene más campos)
		d.Resize(fyne.NewSize(550, 400))
		d.Show()
	}

	// --- LISTA VISUAL ---
	cloudList := container.NewVBox(
		widget.NewLabelWithStyle("Nubes Personales (Navegador)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				       widget.NewButtonWithIcon("Google Drive", theme.StorageIcon(), func() { configureOAuth("Google Drive", "drive") }),
				       widget.NewButtonWithIcon("Dropbox", theme.ContentAddIcon(), func() { configureOAuth("Dropbox", "dropbox") }),
				       widget.NewButtonWithIcon("OneDrive", theme.FolderIcon(), func() { configureOAuth("OneDrive", "onedrive") }),
				       widget.NewButtonWithIcon("pCloud", theme.StorageIcon(), func() { configureOAuth("pCloud", "pcloud") }),
				       widget.NewButtonWithIcon("Box", theme.ContentCopyIcon(), func() { configureOAuth("Box", "box") }),
				       widget.NewButtonWithIcon("Yandex Disk", theme.FileIcon(), func() { configureOAuth("Yandex", "yandex") }),

				       widget.NewSeparator(),
				       widget.NewLabelWithStyle("Autohospedado / Otros", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),

				       widget.NewButtonWithIcon("Mega.nz", theme.UploadIcon(), func() { configureMega() }),
				       widget.NewButtonWithIcon("Nextcloud / Owncloud", theme.ComputerIcon(), func() { configureManual("Nextcloud", "webdav") }),
				       widget.NewButtonWithIcon("WebDAV Genérico", theme.FileIcon(), func() { configureManual("WebDAV", "webdav") }),
				       widget.NewButtonWithIcon("S3 / MinIO / AWS", theme.SettingsIcon(), func() { configureS3() }),
	)

	cancelBtn := widget.NewButtonWithIcon("Cancelar", theme.CancelIcon(), func() { ShowDashboard(w) })
	content := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
				       cancelBtn, nil, nil,
				container.NewPadded(container.NewVScroll(cloudList)),
	)
	w.SetContent(content)
}

func apiError(msg string) error { return &stringError{msg} }

type stringError struct{ msg string }

func (e *stringError) Error() string { return e.msg }

// --- TEMA PERSONALIZADO (Gris Oscuro) ---
type myTheme struct{}

var _ fyne.Theme = (*myTheme)(nil)

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 0x18, G: 0x18, B: 0x18, A: 0xFF}
		case theme.ColorNameOverlayBackground, theme.ColorNameInputBackground:
			return color.NRGBA{R: 0x25, G: 0x25, B: 0x25, A: 0xFF}
		case theme.ColorNameButton:
			return color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xFF}
	}
	return theme.DefaultTheme().Color(name, variant)
}
func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource     { return theme.DefaultTheme().Icon(name) }
func (m myTheme) Font(style fyne.TextStyle) fyne.Resource        { return theme.DefaultTheme().Font(style) }
func (m myTheme) Size(name fyne.ThemeSizeName) float32           { return theme.DefaultTheme().Size(name) }
