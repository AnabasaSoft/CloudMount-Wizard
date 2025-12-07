package main

import (
	"fmt"
	"image/color"
	"strings"
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
		)
		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	myWindow.SetCloseIntercept(func() { myWindow.Hide() })

	if system.CheckRclone() {
		ShowDashboard(myWindow)
	} else {
		content := container.NewVBox(
			widget.NewLabelWithStyle("Falta Rclone", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					     widget.NewLabel("Es necesario para conectar las nubes."),
					     widget.NewButtonWithIcon("Instalar Automáticamente", theme.DownloadIcon(), func() {
						     dialog.ShowInformation("Instalando...", "Por favor espera.", myWindow)
						     go func() {
							     err := system.InstallRclone()
							     fyne.Do(func() {
								     if err != nil {
									     dialog.ShowError(err, myWindow)
								     } else {
									     dialog.ShowInformation("Éxito", "Instalado correctamente.", myWindow)
									     ShowDashboard(myWindow)
								     }
							     })
						     }()
					     }),
		)
		myWindow.SetContent(container.NewCenter(content))
	}

	myWindow.ShowAndRun()
}

// ShowDashboard LISTA LAS UNIDADES
func ShowDashboard(w fyne.Window) {
	title := widget.NewLabelWithStyle("Mis Unidades", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	addBtn := widget.NewButtonWithIcon("Nueva Conexión", theme.ContentAddIcon(), func() { ShowCloudSelection(w) })

	listContainer := container.NewVBox()

	remotes, _ := rclone.ListRemotes()
	for _, rName := range remotes {
		name := rName
		mountPath := rclone.GetMountPath(name)
		isMounted := rclone.IsMounted(mountPath)
		opts := settings.GetOptions(name)

		// Detectar si es Mega
		isMega := (name == "Mega")
		displayName := name
		if isMega { displayName = "MEGA (Oficial)" }

		// ESTADO INTELIGENTE
		statusTxt := "OFF"
		statusIcon := theme.ContentClearIcon()

		if isMounted {
			statusTxt = "MONTADO"
			statusIcon = theme.ConfirmIcon()
		} else if isMega && mega.IsLoggedIn() {
			statusTxt = "SESIÓN OK"
			statusIcon = theme.InfoIcon()
			// Reactivación silenciosa del puente
			go mega.GetWebDAVURL()
		}

		// QUOTA
		quotaTxt := binding.NewString(); quotaTxt.Set("...")
		quotaVal := binding.NewFloat()

		// Calculamos espacio si está montado O conectado
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
				// Fallback Rclone
				q, err := rclone.GetQuota(name)
				if err == nil && q.Total > 0 {
					fyne.Do(func() {
						quotaTxt.Set(fmt.Sprintf("%s / %s", rclone.FormatBytes(q.Used), rclone.FormatBytes(q.Total)))
						quotaVal.Set(float64(q.Used)/float64(q.Total))
					})
				} else {
					fyne.Do(func() { quotaTxt.Set("Espacio: Desconocido") })
				}
			}()
		}

		// BOTONES
		btnMount := widget.NewButton("Montar Disco", func() {
			go func() {
				if isMega { mega.GetWebDAVURL() }
				rclone.MountRemote(name)
				fyne.Do(func() { ShowDashboard(w) })
			}()
		})

		btnUnmount := widget.NewButton("Desmontar", func() {
			go func() { rclone.UnmountRemote(name); fyne.Do(func() { ShowDashboard(w) }) }()
		})

		btnOpen := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() { rclone.OpenFileManager(mountPath) })

		if isMounted {
			btnMount.Disable()
		} else {
			btnUnmount.Disable()
			btnOpen.Disable()
		}

		// AJUSTES
		btnSettings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			checkRead := widget.NewCheck("Solo Lectura", nil); checkRead.Checked = opts.ReadOnly
			entryCache := widget.NewEntry(); entryCache.Text = opts.CacheSize; entryCache.PlaceHolder = "Ej: 10G"
			entryBw := widget.NewEntry(); entryBw.Text = opts.BwLimit; entryBw.PlaceHolder = "Ej: 2M"

			items := []*widget.FormItem{
				widget.NewFormItem("Solo Lectura:", checkRead),
							widget.NewFormItem("Límite Caché:", entryCache),
							widget.NewFormItem("Ancho Banda:", entryBw),
			}

			d := dialog.NewForm("Ajustes "+displayName, "Guardar", "Cancelar", items, func(ok bool) {
				if ok {
					settings.SetOptions(name, settings.RemoteOptions{
						ReadOnly:  checkRead.Checked,
						CacheSize: entryCache.Text,
						BwLimit:   entryBw.Text,
					})
					if isMounted { rclone.UnmountRemote(name); rclone.MountRemote(name) }
					ShowDashboard(w)
				}
			}, w)
			d.Resize(fyne.NewSize(400, 300))
			d.Show()
		})

		// BORRAR / LOGOUT
		btnDelete := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			msg := "¿Eliminar "+displayName+"?"
			if isMega { msg = "¿Cerrar sesión y eliminar Mega?" }
			dialog.ShowConfirm("Borrar", msg, func(ok bool) {
				if ok {
					go func() {
						if isMega { mega.Logout() }
						rclone.DeleteRemote(name)
						fyne.Do(func() { ShowDashboard(w) })
					}()
				}
			}, w)
		})

		card := widget.NewCard("", "", container.NewVBox(
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
		))
		listContainer.Add(card)
	}

	if len(listContainer.Objects) == 0 {
		listContainer.Add(widget.NewLabel("No hay unidades configuradas."))
	}

	w.SetContent(container.NewBorder(
		container.NewVBox(container.NewHBox(title, layout.NewSpacer(), addBtn), widget.NewSeparator()),
					 nil, nil, nil,
					 container.NewPadded(container.NewVScroll(listContainer)),
	))
}

// ShowCloudSelection PANTALLA DE SELECCIÓN
func ShowCloudSelection(w fyne.Window) {
	configState := binding.NewString()
	configState.Set("IDLE")

	configState.AddListener(binding.NewDataListener(func() {
		val, _ := configState.Get()
		if strings.HasPrefix(val, "DONE:") {
			remoteName := val[5:]
			dialog.ShowConfirm("Éxito", "Cuenta '"+remoteName+"' guardada.\n¿Montar ahora?", func(ok bool) {
				if ok { go func() { rclone.MountRemote(remoteName); fyne.Do(func() { ShowDashboard(w) }) }() } else { ShowDashboard(w) }
			}, w)
		} else if strings.HasPrefix(val, "ERROR:") {
			fyne.Do(func() { dialog.ShowError(apiError(val[6:]), w) })
		}
	}))

	// 1. MEGA (HÍBRIDO)
	configureMega := func() {
		if !system.CheckMegaCmd() {
			dialog.ShowConfirm("Instalar", "Se necesita MEGAcmd.\n¿Instalar automáticamente?", func(ok bool) {
				if ok {
					w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewLabel("Instalando MEGAcmd..."), widget.NewProgressBarInfinite(), layout.NewSpacer()))
					go func() {
						err := system.InstallMegaCmd()
						fyne.Do(func() {
							if err != nil { ShowCloudSelection(w); dialog.ShowError(err, w) } else { ShowCloudSelection(w); dialog.ShowInformation("Instalado", "Vuelve a conectar.", w) }
						})
					}()
				}
			}, w)
			return
		}

		entryUser := widget.NewEntry(); entryUser.PlaceHolder = "Email"
		entryPass := widget.NewPasswordEntry(); entryPass.PlaceHolder = "Contraseña"
		entry2FA := widget.NewEntry(); entry2FA.PlaceHolder = "Código 2FA"

		d := dialog.NewForm("Conectar Mega", "Login", "Cancelar", []*widget.FormItem{
			widget.NewFormItem("Email:", entryUser), widget.NewFormItem("Pass:", entryPass), widget.NewFormItem("2FA:", entry2FA),
		}, func(ok bool) {
			if ok {
				w.SetContent(container.NewVBox(layout.NewSpacer(), widget.NewLabel("Conectando..."), widget.NewProgressBarInfinite(), layout.NewSpacer()))
				go func() {
					err := mega.Login(strings.TrimSpace(entryUser.Text), strings.TrimSpace(entryPass.Text), strings.TrimSpace(entry2FA.Text))
					if err != nil {
						fyne.Do(func() { ShowCloudSelection(w); dialog.ShowError(fmt.Errorf("Login falló: %v", err), w) })
						return
					}
					webdavURL, errUrl := mega.GetWebDAVURL()
					if errUrl != nil {
						fyne.Do(func() { ShowCloudSelection(w); dialog.ShowError(fmt.Errorf("Error puente: %v", errUrl), w) })
						return
					}
					opts := map[string]string{"url": webdavURL, "vendor": "other", "user": strings.TrimSpace(entryUser.Text), "pass": strings.TrimSpace(entryPass.Text)}
					rclone.CreateConfigWithOpts("Mega", "webdav", opts)

					fyne.Do(func() {
						dialog.ShowInformation("Conectado", "Mega configurado.", w)
						ShowDashboard(w)
					})
				}()
			}
		}, w)
		d.Resize(fyne.NewSize(450, 300)); d.Show()
	}

	// 2. OAUTH
	configureOAuth := func(name, provider string) {
		input := widget.NewEntry(); input.PlaceHolder = "Nombre"
		dialog.ShowCustomConfirm("Configurar "+name, "Ok", "Cancel", input, func(ok bool) {
			if ok && input.Text != "" {
				w.SetContent(widget.NewLabel("Autorizando..."))
				go func() {
					if err := rclone.CreateConfig(input.Text, provider); err != nil { configState.Set("ERROR:"+err.Error()) } else { configState.Set("DONE:"+input.Text) }
				}()
			}
		}, w)
	}

	// 3. MANUAL
	configureManual := func(title, provider string) {
		entryName := widget.NewEntry(); entryURL := widget.NewEntry(); entryURL.PlaceHolder = "https://..."
		entryUser := widget.NewEntry(); entryPass := widget.NewPasswordEntry()
		d := dialog.NewForm(title, "Ok", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Nombre:", entryName), widget.NewFormItem("URL:", entryURL),
				    widget.NewFormItem("User:", entryUser), widget.NewFormItem("Pass:", entryPass),
		}, func(ok bool) {
			if ok {
				opts := map[string]string{"url": entryURL.Text, "user": entryUser.Text, "pass": entryPass.Text, "vendor": "other"}
				if provider == "nextcloud" { opts["vendor"] = "nextcloud" }
				go func() { if err := rclone.CreateConfigWithOpts(entryName.Text, provider, opts); err != nil { configState.Set("ERROR:"+err.Error()) } else { configState.Set("DONE:"+entryName.Text) } }()
			}
		}, w)
		d.Resize(fyne.NewSize(500, 350)); d.Show()
	}

	// 4. S3
	configureS3 := func() {
		entryName := widget.NewEntry(); entryProvider := widget.NewSelect([]string{"AWS", "Minio", "Wasabi", "Other"}, nil)
		entryAccess := widget.NewEntry(); entrySecret := widget.NewPasswordEntry(); entryEndpoint := widget.NewEntry()
		d := dialog.NewForm("Configurar S3", "Ok", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Nombre:", entryName), widget.NewFormItem("Prov:", entryProvider),
				    widget.NewFormItem("Access:", entryAccess), widget.NewFormItem("Secret:", entrySecret),
				    widget.NewFormItem("Endpoint:", entryEndpoint),
		}, func(ok bool) {
			if ok {
				opts := map[string]string{"provider": entryProvider.Selected, "env_auth": "false", "access_key_id": entryAccess.Text, "secret_access_key": entrySecret.Text}
				if entryEndpoint.Text != "" { opts["endpoint"] = entryEndpoint.Text }
				go func() { if err := rclone.CreateConfigWithOpts(entryName.Text, "s3", opts); err != nil { configState.Set("ERROR:"+err.Error()) } else { configState.Set("DONE:"+entryName.Text) } }()
			}
		}, w)
		d.Resize(fyne.NewSize(500, 400)); d.Show()
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
		case theme.ColorNameBackground: return color.NRGBA{R: 0x18, G: 0x18, B: 0x18, A: 0xFF}
		case theme.ColorNameOverlayBackground, theme.ColorNameInputBackground: return color.NRGBA{R: 0x25, G: 0x25, B: 0x25, A: 0xFF}
		case theme.ColorNameButton: return color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xFF}
	}
	return theme.DefaultTheme().Color(n, v)
}
func (m myTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }
func (m myTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }
func (m myTheme) Size(n fyne.ThemeSizeName) float32 { return theme.DefaultTheme().Size(n) }
