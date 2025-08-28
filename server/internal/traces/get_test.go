package traces

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/example/otel-stack-demo/internal/sources"
	"github.com/gin-gonic/gin"
)

func TestGet_ReturnsTimelineSpans(t *testing.T) {
	body := "" +
		`{"TraceId":"T","SpanId":"A","ParentSpanId":"","SpanName":"root","SpanKind":"SERVER","ServiceName":"web","start_ns":0,"end_ns":1000000,"SpanAttributes":{},"StatusCode":"OK","StatusMessage":""}` + "\n" +
		`{"TraceId":"T","SpanId":"B","ParentSpanId":"A","SpanName":"db","SpanKind":"CLIENT","ServiceName":"db","start_ns":100000,"end_ns":200000,"SpanAttributes":{"db.system":"mysql"},"StatusCode":"OK","StatusMessage":""}` + "\n"

	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/traces/:traceId", Get(src))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/traces/T", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var out struct {
		TraceID string `json:"traceId"`
		Spans   []Span `json:"spans"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(out.Spans) != 2 {
		t.Fatalf("spans=%d want 2", len(out.Spans))
	}
	if out.Spans[0].Service != "web" || out.Spans[0].Name != "root" {
		t.Fatalf("unexpected root: %+v", out.Spans[0])
	}
	if out.Spans[1].Attributes["db.system"] != "mysql" {
		t.Fatalf("attr not decoded: %+v", out.Spans[1].Attributes)
	}
}
