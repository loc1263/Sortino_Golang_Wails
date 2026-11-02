// Aplicación de escritorio con Wails v3: backend en Go y frontend en Svelte.
//
// Estructura del proyecto:
//
//	main.go              arranque de la app y ventana (solo cableado)
//	stocks_service.go    servicio expuesto al frontend (puente Go <-> JavaScript)
//	internal/market      datos externos: lista del S&P 500 y precios de Yahoo
//	internal/ranking     indicadores y puntaje (funciones puras, con pruebas)
//	internal/export      escritura del CSV
//	frontend/            interfaz Svelte (ver frontend/src)
//
// Ver README.md para los comandos de desarrollo y compilación.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// go:embed incluye la carpeta del frontend compilado dentro del .exe, por eso
// el programa final es un único archivo sin dependencias de archivos externos.
// Esa carpeta (frontend/dist) la genera `wails3 build` al compilar el frontend.
//
//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Registrar el tipo de cada evento permite que Wails genere el API
	// JavaScript con tipos para escucharlo.
	application.RegisterEvent[Progress](progressEvent)
}

func main() {
	app := application.New(application.Options{
		Name:        "Sortino_Golang_Wails",
		Description: "Ranking riesgo/retorno del S&P 500",

		// Cada servicio registrado expone sus métodos públicos al frontend.
		Services: []application.Service{
			application.NewService(NewStockService()),
		},

		// De dónde se sirve la interfaz web: los archivos embebidos de arriba.
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Sortino_Golang_Wails",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGB(27, 38, 54), // evita el parpadeo blanco al abrir
		URL:              "/",
	})

	// Run bloquea hasta que se cierra la app.
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
