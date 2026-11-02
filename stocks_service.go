package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"Sortino_Golang_Wails/internal/export"
	"Sortino_Golang_Wails/internal/market"
	"Sortino_Golang_Wails/internal/ranking"
)

const (
	workers = 10 // descargas simultáneas; más de esto arriesga que Yahoo nos limite
	topN    = 10 // cuántas acciones muestra el ranking

	// Tiempo máximo de una descarga completa (~500 acciones).
	downloadTimeout = 5 * time.Minute

	// Nombre del evento que el backend emite y el frontend escucha.
	progressEvent = "progress"
)

// Report es el resultado de una descarga completa. Se serializa a JSON y llega
// al frontend; Wails genera su clase JavaScript a partir de estas etiquetas.
type Report struct {
	Top    []ranking.Stock `json:"top"`    // las mejores acciones, ya ordenadas
	Total  int             `json:"total"`  // acciones que tiene el índice
	Failed int             `json:"failed"` // acciones que no se pudieron analizar
}

// Progress es el dato del evento "progress": cuántas acciones se han procesado.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// StockService es un "servicio" de Wails: todos sus métodos públicos quedan
// disponibles para el frontend como funciones JavaScript (en
// frontend/bindings/Sortino_Golang_Wails). Por eso solo expone lo que la interfaz necesita
// y deja la lógica real en los paquetes de internal/.
type StockService struct {
	market *market.Client

	// Wails puede invocar métodos desde distintas goroutines, así que el
	// estado compartido se protege con un mutex.
	mu   sync.Mutex
	last Report // último resultado; lo usa ExportCSV
}

// NewStockService crea el servicio con sus dependencias.
func NewStockService() *StockService {
	return &StockService{market: market.NewClient()}
}

// outcome es el resultado de analizar una empresa: o trae la acción o el error.
type outcome struct {
	stock ranking.Stock
	err   error
}

// TopStocks descarga un año de precios de todas las acciones del S&P 500,
// calcula sus indicadores y devuelve las mejores. Mientras trabaja emite el
// evento "progress".
func (s *StockService) TopStocks() (Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	companies, err := s.market.Constituents(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("no se pudo obtener la lista del S&P 500: %w", err)
	}

	// Patrón "fan-out / fan-in":
	//   - un productor envía empresas al canal `jobs`
	//   - varios workers las toman, las analizan y envían el resultado a `results`
	//   - este método (el consumidor) junta todo y emite el progreso
	// Como solo el consumidor modifica los totales, no hace falta un mutex.
	jobs := make(chan market.Company)
	results := make(chan outcome)

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for company := range jobs {
				results <- s.analyze(ctx, company)
			}
		})
	}

	go func() { // productor
		defer close(jobs)
		for _, c := range companies {
			jobs <- c
		}
	}()

	go func() { // cierra `results` cuando todos los workers terminaron
		wg.Wait()
		close(results)
	}()

	var stocks []ranking.Stock
	failed, done := 0, 0
	for r := range results {
		done++
		if r.err != nil {
			failed++
		} else {
			stocks = append(stocks, r.stock)
		}
		// Se emite cada 10 acciones para no inundar al frontend de eventos.
		if done%10 == 0 || done == len(companies) {
			application.Get().Event.Emit(progressEvent, Progress{Done: done, Total: len(companies)})
		}
	}

	report := Report{Top: ranking.Rank(stocks, topN), Total: len(companies), Failed: failed}

	s.mu.Lock()
	s.last = report
	s.mu.Unlock()

	return report, nil
}

// analyze baja los precios de una empresa y calcula sus indicadores.
func (s *StockService) analyze(ctx context.Context, c market.Company) outcome {
	prices, err := s.market.Prices(ctx, c.Symbol)
	if err != nil {
		return outcome{err: err}
	}

	stock, err := ranking.Analyze(prices)
	if err != nil {
		return outcome{err: err}
	}

	stock.Symbol, stock.Name, stock.Sector = c.Symbol, c.Name, c.Sector
	return outcome{stock: stock}
}

// ExportCSV pide al usuario dónde guardar y escribe el último ranking como CSV.
// Devuelve la ruta guardada, o "" si el usuario canceló el diálogo.
func (s *StockService) ExportCSV() (string, error) {
	s.mu.Lock()
	top := s.last.Top
	s.mu.Unlock()

	if len(top) == 0 {
		return "", fmt.Errorf("primero descarga los precios")
	}

	// Diálogo nativo de Windows para elegir dónde guardar.
	path, err := application.Get().Dialog.SaveFile().
		SetMessage("Exportar top 10").
		SetFilename("sp500_top10_"+time.Now().Format("2006-01-02")+".csv").
		AddFilter("CSV (*.csv)", "*.csv").
		PromptForSingleSelection()
	if err != nil || path == "" { // error, o el usuario canceló
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}

	if err := export.WriteTop(f, top); err != nil {
		f.Close()
		return "", err
	}
	// Close también puede fallar (por ejemplo, disco lleno): hay que revisarlo.
	if err := f.Close(); err != nil {
		return "", err
	}
	return path, nil
}
