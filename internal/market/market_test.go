package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient apunta el cliente a un servidor falso local, así las pruebas
// no dependen de internet ni de que Yahoo esté disponible.
func newTestClient(handler http.HandlerFunc) (*Client, func()) {
	srv := httptest.NewServer(handler)
	c := NewClient()
	c.chartURL = srv.URL + "/chart/%s"
	c.constituentsURL = srv.URL + "/constituents.csv"
	return c, srv.Close
}

func TestPricesSkipsNullsAndPrefersAdjClose(t *testing.T) {
	c, closeSrv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		// Un día sin datos (null) debe descartarse.
		w.Write([]byte(`{"chart":{"result":[{"indicators":{
			"quote":[{"close":[1,2,3]}],
			"adjclose":[{"adjclose":[10,null,30]}]}}]}}`))
	})
	defer closeSrv()

	prices, err := c.Prices(context.Background(), "AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) != 2 || prices[0] != 10 || prices[1] != 30 {
		t.Fatalf("precios = %v, esperaba [10 30]", prices)
	}
}

func TestPricesConvertsDotToDash(t *testing.T) {
	var path string
	c, closeSrv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"chart":{"result":[{"indicators":{"quote":[{"close":[1]}]}}]}}`))
	})
	defer closeSrv()

	if _, err := c.Prices(context.Background(), "BRK.B"); err != nil {
		t.Fatal(err)
	}
	if path != "/chart/BRK-B" {
		t.Fatalf("ruta = %q, esperaba /chart/BRK-B", path)
	}
}

func TestConstituents(t *testing.T) {
	c, closeSrv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Symbol,Security,GICS Sector\nMMM,3M,Industrials\nAOS,A. O. Smith,Industrials\n"))
	})
	defer closeSrv()

	companies, err := c.Constituents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(companies) != 2 || companies[0].Symbol != "MMM" || companies[1].Name != "A. O. Smith" {
		t.Fatalf("empresas inesperadas: %+v", companies)
	}
}
