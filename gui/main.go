package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.New()
	w := a.NewWindow("Distancizer")
	w.Resize(fyne.NewSize(800, 600))

	da := NewDistancizerApp(w)
	w.SetContent(da.buildUI())
	w.ShowAndRun()
}
