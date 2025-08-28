package traces

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/example/otel-stack-demo/internal/sources"
	"github.com/gin-gonic/gin"
)

func TestList_ParsesTraceRootsAndMapsFields(t *testing.T) {
	body := "" +
		`{"TraceId":"t1","StartTs":"2025-01-01 10:00:00","DurationMs":812.5,"RootService":"web","RootOperation":"GET /checkout","Status":"OK","SpanCount":20,"TopService":"web","TopServiceMs":420.0}` + "\n" +
		`{"TraceId":"t2","StartTs":"2025-01-01 10:05:00","DurationMs":1200.0,"RootService":"api","RootOperation":"POST /charge","Status":"ERROR","SpanCount":33,"TopService":"payments","TopServiceMs":800.0}` + "\n"

	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/traces/list", List(src))

	reqBody := []byte(`{
		"from": 1704103200, "to": 1704106800,
		"filters": { "status": ["ERROR"] },
		"sort": { "by":"duration", "order":"DESC" },
		"page": { "size": 50 }
	}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/traces/list", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("items=%d want 2", len(out.Items))
	}
	if out.Items[1]["traceId"] != "t2" || out.Items[1]["status"] != "ERROR" {
		t.Fatalf("row[1] unexpected: %+v", out.Items[1])
	}
}
