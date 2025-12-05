package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Run inicia la aplicación
func Run() {
	// 1. Crear la instancia de la aplicación
	// Fyne detecta automáticamente si tu ID es único para guardar preferencias
	myApp := app.NewWithID("com.cloudmount.wizard")

	// 2. Crear la ventana principal
	w := myApp.NewWindow("CloudMount Wizard")
	w.Resize(fyne.NewSize(600, 400)) // Tamaño inicial decente

	// 3. Crear elementos (Widgets)
	title := widget.NewLabelWithStyle(
		"Configurador de Nube",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	subtitle := widget.NewLabel("Bienvenido. Vamos a montar tu Google Drive o OneDrive como un disco local.")
	subtitle.Wrapping = fyne.TextWrapWord // Para que el texto no se corte

	// Un contenedor para el contenido principal
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
				     subtitle,
			      layoutSpacer(), // Función auxiliar (ver abajo)
	widget.NewButton("Comenzar Configuración", func() {
		// Aquí irá la lógica más adelante
		subtitle.SetText("¡Botón presionado! El sistema funciona.")
	}),
	)

	// 4. Asignar contenido a la ventana y ejecutar
	w.SetContent(container.NewCenter(content))
	w.ShowAndRun()
}

// layoutSpacer crea un espacio vertical flexible (hack visual simple)
func layoutSpacer() fyne.CanvasObject {
	return widget.NewLabel("")
}
