// Package ranking calcula indicadores de riesgo/retorno a partir de una serie
// de precios y ordena las acciones con un puntaje compuesto.
//
// Son funciones puras: reciben números y devuelven números. No hacen red ni
// tocan la interfaz, así que se pueden probar fácilmente (ver ranking_test.go).
package ranking

import (
	"errors"
	"math"
)

// Parámetros del análisis. Se expresan en "días de mercado" (ruedas).
const (
	tradingDays = 252 // días hábiles por año; sirve para anualizar
	minPrices   = 200 // mínimo de datos; descarta salidas a bolsa recientes
	skipDays    = 21  // el último mes se excluye del momentum (ver abajo)
	window3M    = 63  // ~3 meses de ruedas
	window20D   = 20  // ~1 mes de ruedas

	// Piso de la caída máxima al calcular retorno/caída. Evita dividir por un
	// número casi cero cuando una acción no cayó nunca en el período.
	drawdownFloor = 0.01
)

// Errores esperables: no son fallas del programa, solo acciones que no se
// pueden analizar. El llamador decide si las cuenta como "excluidas".
var (
	ErrNotEnoughData = errors.New("pocos datos")
	ErrNoDownside    = errors.New("sin días a la baja")
)

// Stock es una acción con sus indicadores. Las etiquetas `json` definen los
// nombres de los campos que verá el frontend (y que Wails usa para generar
// los modelos de JavaScript).
type Stock struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Sector string `json:"sector"`

	Last     float64 `json:"last"`     // último precio de cierre
	Momentum float64 `json:"momentum"` // retorno de 12 meses sin contar el último mes
	Vol20    float64 `json:"vol20"`    // volatilidad anualizada de los últimos 20 días
	Sortino  float64 `json:"sortino"`  // Sortino anualizado de los últimos 3 meses
	Return3M float64 `json:"return3m"` // retorno de los últimos 3 meses
	MaxDD3M  float64 `json:"maxdd3m"`  // caída máxima en 3 meses (fracción positiva)
	Calmar3M float64 `json:"calmar3m"` // Return3M / MaxDD3M
	Score    float64 `json:"score"`    // 0-100, lo asigna Rank
}

// Analyze calcula los indicadores de una acción a partir de sus precios
// (del más antiguo al más reciente). Deja vacíos Symbol, Name, Sector y
// Score: los completa quien llama.
func Analyze(prices []float64) (Stock, error) {
	n := len(prices)
	if n < minPrices {
		return Stock{}, ErrNotEnoughData
	}

	returns := dailyReturns(prices)
	annual := math.Sqrt(tradingDays) // factor para anualizar una desviación diaria

	// --- Momentum 12-1 -------------------------------------------------
	// Retorno desde hace ~12 meses hasta hace 1 mes. Se salta el último mes
	// porque a ese plazo las acciones tienden a revertir (las que más
	// subieron suelen devolver parte del alza).
	momentum := prices[n-1-skipDays]/prices[0] - 1

	// --- Volatilidad de 20 días ---------------------------------------
	// Es la parte más "predecible" del riesgo: una acción agitada hoy
	// probablemente lo siga siendo el próximo mes.
	vol20 := stdDev(returns[len(returns)-window20D:]) * annual

	// --- Últimos 3 meses ---------------------------------------------
	r3 := returns[len(returns)-window3M:]
	p3 := prices[n-1-window3M:] // un precio más que retornos

	sortino, err := sortino(r3, annual)
	if err != nil {
		return Stock{}, err
	}

	maxDD := maxDrawdown(p3)
	ret3 := p3[len(p3)-1]/p3[0] - 1

	return Stock{
		Last:     prices[n-1],
		Momentum: momentum,
		Vol20:    vol20,
		Sortino:  sortino,
		Return3M: ret3,
		MaxDD3M:  maxDD,
		Calmar3M: ret3 / math.Max(maxDD, drawdownFloor),
	}, nil
}

// dailyReturns convierte precios en retornos diarios: p[i]/p[i-1] - 1.
func dailyReturns(prices []float64) []float64 {
	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns = append(returns, prices[i]/prices[i-1]-1)
	}
	return returns
}

// sortino = media / desviación a la baja, anualizado.
//
// Es como el Sharpe, pero en el denominador solo cuentan los días negativos:
// una subida fuerte no se considera "riesgo". La tasa mínima aceptable es 0.
func sortino(returns []float64, annual float64) (float64, error) {
	sumSq := 0.0
	for _, r := range returns {
		if r < 0 {
			sumSq += r * r
		}
	}
	downside := math.Sqrt(sumSq / float64(len(returns)))
	if downside == 0 {
		return 0, ErrNoDownside
	}
	return mean(returns) / downside * annual, nil
}

// maxDrawdown es la mayor caída porcentual desde un máximo previo hasta un
// mínimo posterior, como fracción positiva (0.10 = cayó 10%).
func maxDrawdown(prices []float64) float64 {
	peak, worst := prices[0], 0.0
	for _, p := range prices {
		peak = math.Max(peak, p)
		worst = math.Max(worst, 1-p/peak)
	}
	return worst
}

func mean(xs []float64) float64 {
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// stdDev es la desviación estándar muestral (divide por n-1).
func stdDev(xs []float64) float64 {
	m := mean(xs)
	sum := 0.0
	for _, x := range xs {
		sum += (x - m) * (x - m)
	}
	return math.Sqrt(sum / float64(len(xs)-1))
}
