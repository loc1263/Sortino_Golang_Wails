package ranking

import (
	"errors"
	"math"
	"testing"
)

// series construye `n` precios que parten de `start` y varían `step` por día.
func series(n int, start, step float64) []float64 {
	prices := make([]float64, n)
	for i := range prices {
		prices[i] = start + step*float64(i)
	}
	return prices
}

func TestAnalyzeNotEnoughData(t *testing.T) {
	_, err := Analyze(series(minPrices-1, 100, 1))
	if !errors.Is(err, ErrNotEnoughData) {
		t.Fatalf("esperaba ErrNotEnoughData, obtuve %v", err)
	}
}

func TestAnalyzeNoDownside(t *testing.T) {
	// Una serie que solo sube no tiene días negativos: el Sortino no se puede calcular.
	_, err := Analyze(series(250, 100, 1))
	if !errors.Is(err, ErrNoDownside) {
		t.Fatalf("esperaba ErrNoDownside, obtuve %v", err)
	}
}

func TestMaxDrawdown(t *testing.T) {
	// Sube a 100, cae a 80 (-20%) y se recupera a 90: la caída máxima es 20%.
	got := maxDrawdown([]float64{50, 100, 80, 90})
	if math.Abs(got-0.20) > 1e-9 {
		t.Fatalf("maxDrawdown = %v, esperaba 0.20", got)
	}
}

func TestSortinoIgnoresUpside(t *testing.T) {
	// Con la misma caída, más subidas deben dar un Sortino mayor.
	annual := math.Sqrt(tradingDays)
	low, _ := sortino([]float64{0.01, -0.01, 0.01, -0.01}, annual)
	high, _ := sortino([]float64{0.05, -0.01, 0.05, -0.01}, annual)
	if high <= low {
		t.Fatalf("Sortino con más subidas (%v) debería superar a %v", high, low)
	}
}

func TestRankOrdersBestFirstAndLimits(t *testing.T) {
	stocks := []Stock{
		{Symbol: "MALA", Momentum: -0.10, Vol20: 0.60, Sortino: -1},
		{Symbol: "BUENA", Momentum: 0.40, Vol20: 0.15, Sortino: 5},
		{Symbol: "MEDIA", Momentum: 0.10, Vol20: 0.30, Sortino: 2},
	}

	top := Rank(stocks, 2)

	if len(top) != 2 {
		t.Fatalf("esperaba 2 acciones, obtuve %d", len(top))
	}
	if top[0].Symbol != "BUENA" || top[1].Symbol != "MEDIA" {
		t.Fatalf("orden inesperado: %s, %s", top[0].Symbol, top[1].Symbol)
	}
	// La mejor en los tres indicadores recibe el puntaje máximo.
	if top[0].Score != 100 {
		t.Fatalf("Score de la mejor = %v, esperaba 100", top[0].Score)
	}
}
