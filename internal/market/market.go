// Package market obtiene datos de fuentes externas:
//   - la lista de empresas del S&P 500 (un CSV público en GitHub)
//   - los precios históricos de cada acción (API de gráficos de Yahoo Finance)
//
// Es la única parte del programa que habla con internet. Si mañana quisieras
// cambiar de proveedor de datos, solo tendrías que tocar este paquete.
package market

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Lista de constituyentes del S&P 500 mantenida por la comunidad.
	defaultConstituentsURL = "https://raw.githubusercontent.com/datasets/s-and-p-500-companies/main/data/constituents.csv"

	// Plantilla de la API de Yahoo: %s se reemplaza por el símbolo.
	// range=1y e interval=1d => un año de velas diarias.
	defaultChartURL = "https://query1.finance.yahoo.com/v8/finance/chart/%s?range=1y&interval=1d"

	// Yahoo rechaza peticiones sin un User-Agent que parezca de navegador.
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

	// Cuántas veces se intenta bajar los precios de una acción antes de rendirse.
	maxAttempts = 2
)

// Company identifica una empresa del índice.
type Company struct {
	Symbol string
	Name   string
	Sector string
}

// Client agrupa lo necesario para consultar los datos.
//
// Las URLs son campos (y no constantes sueltas) para poder reemplazarlas por
// un servidor falso en las pruebas; ver market_test.go.
type Client struct {
	http            *http.Client
	constituentsURL string
	chartURL        string
}

// NewClient crea un cliente con valores por defecto sensatos.
func NewClient() *Client {
	return &Client{
		// Siempre se define un timeout: el cliente por defecto de Go no tiene
		// ninguno y una petición colgada bloquearía la app para siempre.
		http:            &http.Client{Timeout: 20 * time.Second},
		constituentsURL: defaultConstituentsURL,
		chartURL:        defaultChartURL,
	}
}

// Constituents descarga la lista de empresas del S&P 500.
func (c *Client) Constituents(ctx context.Context) ([]Company, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.constituentsURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // siempre cerrar el cuerpo para liberar la conexión

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("el CSV no contiene empresas")
	}

	// La primera fila es el encabezado: Symbol,Security,GICS Sector,...
	companies := make([]Company, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) < 3 {
			continue
		}
		companies = append(companies, Company{Symbol: row[0], Name: row[1], Sector: row[2]})
	}
	return companies, nil
}

// Prices devuelve los cierres diarios ajustados del último año, del más
// antiguo al más reciente. Reintenta una vez si la primera petición falla.
func (c *Client) Prices(ctx context.Context, symbol string) ([]float64, error) {
	// Yahoo escribe las clases de acciones con guion: BRK.B -> BRK-B.
	symbol = strings.ReplaceAll(symbol, ".", "-")
	endpoint := fmt.Sprintf(c.chartURL, url.PathEscape(symbol))

	var lastErr error
	for range maxAttempts {
		prices, err := c.fetchPrices(ctx, endpoint)
		if err == nil {
			return prices, nil
		}
		lastErr = err

		// Pausa corta entre intentos, pero cancelable: si el contexto se
		// cancela no tiene sentido seguir esperando.
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

// chartResponse modela solo los campos del JSON de Yahoo que nos interesan.
// Los precios son *float64 porque Yahoo envía null en los días sin datos.
type chartResponse struct {
	Chart struct {
		Result []struct {
			Indicators struct {
				AdjClose []struct {
					AdjClose []*float64 `json:"adjclose"`
				} `json:"adjclose"`
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

// fetchPrices hace una sola petición a Yahoo y extrae la serie de precios.
func (c *Client) fetchPrices(ctx context.Context, endpoint string) ([]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // vaciar el cuerpo permite reutilizar la conexión
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("sin datos")
	}

	// Preferimos el cierre ajustado (descuenta dividendos y splits), que es lo
	// correcto para calcular retornos. Si no existe, usamos el cierre normal.
	ind := data.Chart.Result[0].Indicators
	var raw []*float64
	switch {
	case len(ind.AdjClose) > 0:
		raw = ind.AdjClose[0].AdjClose
	case len(ind.Quote) > 0:
		raw = ind.Quote[0].Close
	}

	prices := make([]float64, 0, len(raw))
	for _, p := range raw {
		if p != nil { // descartar los null
			prices = append(prices, *p)
		}
	}
	return prices, nil
}
