package export

import (
	"bytes"
	"strings"
	"testing"

	"Sortino_Golang_Wails/internal/ranking"
)

func TestWriteTop(t *testing.T) {
	var buf bytes.Buffer // un buffer en memoria hace de "archivo"
	stocks := []ranking.Stock{{Symbol: "AAPL", Name: "Apple", Sector: "Tech", Last: 150.5, Score: 91.234}}

	if err := WriteTop(&buf, stocks); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, utf8BOM) {
		t.Error("falta el BOM UTF-8 al inicio")
	}

	lines := strings.Split(strings.TrimSpace(strings.TrimPrefix(out, utf8BOM)), "\n")
	if len(lines) != 2 {
		t.Fatalf("esperaba encabezado + 1 fila, obtuve %d líneas", len(lines))
	}
	if !strings.HasPrefix(lines[1], "1,AAPL,Apple,Tech,150.50,") {
		t.Errorf("fila inesperada: %q", lines[1])
	}
	if !strings.HasSuffix(lines[1], ",91.2") {
		t.Errorf("el score debería terminar en 91.2: %q", lines[1])
	}
}
