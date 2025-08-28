package traces

import (
	"net/http/httptest"
	"testing"

	"github.com/example/otel-stack-demo/internal/sources"
	"github.com/gin-gonic/gin"
)

func TestSuggest_Services(t *testing.T) {
	body := "" +
		`{"ServiceName":"checkout","c":120}` + "\n" +
		`{"ServiceName":"api","c":90}` + "\n"
	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/traces/suggest/services", SuggestServices(src))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/traces/suggest/services?q=ch", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 || len(w.Body.Bytes()) == 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSuggest_Operations(t *testing.T) {
	body := "" +
		`{"SpanName":"GET /cart","c":50}` + "\n" +
		`{"SpanName":"POST /charge","c":40}` + "\n"
	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}
	r := gin.New()
	r.GET("/api/traces/suggest/operations", SuggestOperations(src))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/traces/suggest/operations?q=ge", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 || len(w.Body.Bytes()) == 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSuggest_Attributes(t *testing.T) {
	body := "" +
		`{"Val":"GET"}` + "\n" +
		`{"Val":"POST"}` + "\n"
	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}
	r := gin.New()
	r.GET("/api/traces/suggest/attributes", SuggestAttributes(src))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/traces/suggest/attributes?key=http.method&q=G", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 || len(w.Body.Bytes()) == 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
