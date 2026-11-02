// Package export escribe los resultados en formatos que se abren fuera de la app.
package export

import (
	"encoding/csv"
	"io"
	"strconv"

	"Sortino_Golang_Wails/internal/ranking"
)

// utf8BOM es la "marca de orden de bytes". Excel la usa para reconocer que el
// archivo está en UTF-8; sin ella muestra mal los acentos.
const utf8BOM = "\xef\xbb\xbf"

// WriteTop escribe el ranking como CSV en w. Recibe un io.Writer en lugar de
// una ruta para poder probarlo con un buffer en memoria, sin tocar el disco.
func WriteTop(w io.Writer, stocks []ranking.Stock) error {
	if _, err := io.WriteString(w, utf8BOM); err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	cw.Write([]string{
		"rank", "symbol", "name", "sector", "last_price", "momentum_12_1", "vol_20d",
		"sortino_3m", "return_3m", "maxdd_3m", "return_maxdd_3m", "score",
	})

	num := func(v float64, decimals int) string { return strconv.FormatFloat(v, 'f', decimals, 64) }
	for i, s := range stocks {
		cw.Write([]string{
			strconv.Itoa(i + 1), s.Symbol, s.Name, s.Sector,
			num(s.Last, 2), num(s.Momentum, 4), num(s.Vol20, 4), num(s.Sortino, 2),
			num(s.Return3M, 4), num(s.MaxDD3M, 4), num(s.Calmar3M, 2), num(s.Score, 1),
		})
	}

	// csv.Writer guarda en un buffer: Flush vacía el buffer y Error informa
	// si alguna escritura anterior falló.
	cw.Flush()
	return cw.Error()
}
