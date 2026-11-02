package ranking

import "sort"

// Rank asigna un puntaje a cada acción y devuelve las n mejores, de mayor a
// menor puntaje.
//
// El puntaje (0-100) es el promedio del percentil de tres indicadores:
//   - Momentum: cuanto más alto, mejor
//   - Vol20:    cuanto más baja, mejor
//   - Sortino:  cuanto más alto, mejor
//
// Usar percentiles (posición relativa) en vez de valores absolutos permite
// combinar indicadores con escalas distintas: un 40% de momentum y un
// Sortino de 5 no se pueden sumar directamente, pero "está en el percentil
// 90 de cada uno" sí.
//
// Rank reordena y modifica la slice recibida.
func Rank(stocks []Stock, n int) []Stock {
	assignScores(stocks)

	sort.Slice(stocks, func(i, j int) bool { return stocks[i].Score > stocks[j].Score })
	if len(stocks) > n {
		stocks = stocks[:n]
	}
	return stocks
}

func assignScores(stocks []Stock) {
	count := len(stocks)
	if count < 2 {
		return // con una sola acción no hay con qué comparar
	}

	sums := make([]float64, count)

	// addPercentile ordena las acciones de peor a mejor según `worse` y suma
	// a cada una su percentil: la peor recibe 0 y la mejor 100.
	addPercentile := func(worse func(a, b Stock) bool) {
		order := make([]int, count)
		for i := range order {
			order[i] = i
		}
		sort.Slice(order, func(x, y int) bool { return worse(stocks[order[x]], stocks[order[y]]) })
		for position, idx := range order {
			sums[idx] += float64(position) / float64(count-1) * 100
		}
	}

	addPercentile(func(a, b Stock) bool { return a.Momentum < b.Momentum }) // más momentum = mejor
	addPercentile(func(a, b Stock) bool { return a.Vol20 > b.Vol20 })       // menos volatilidad = mejor
	addPercentile(func(a, b Stock) bool { return a.Sortino < b.Sortino })   // más Sortino = mejor

	const indicators = 3
	for i := range stocks {
		stocks[i].Score = sums[i] / indicators
	}
}
