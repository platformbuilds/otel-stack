package traces

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/example/otel-stack-demo/internal/sources"
)

func TestFlame_TotalAndSelf(t *testing.T) {
	// Trace: root A (0..1000µs), children: B (100..300) = 200µs, C (400..700) = 300µs.
	// total(root)=1000, self(root)=1000-(200+300)=500
	body := "" +
		`{"SpanId":"A","ParentSpanId":"","SpanName":"A","ServiceName":"web","start_ns":0,"end_ns":1000000}` + "\n" +
		`{"SpanId":"B","ParentSpanId":"A","SpanName":"B","ServiceName":"cart","start_ns":100000,"end_ns":300000}` + "\n" +
		`{"SpanId":"C","ParentSpanId":"A","SpanName":"C","ServiceName":"pay","start_ns":400000,"end_ns":700000}` + "\n"

	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}
	r := newRouter("/api/traces/:traceId/flame", Flame(src))

	// mode=total
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/traces/t1/flame?groupBy=service_operation&mode=total", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("status(total)=%d body=%s", w1.Code, w1.Body.String())
	}
	var total FlameNode
	if err := json.Unmarshal(w1.Body.Bytes(), &total); err != nil {
		t.Fatalf("json(total): %v", err)
	}
	if total.Name != "web:A" || total.Value != 1000 {
		t.Fatalf("root(total) got name=%q value=%d", total.Name, total.Value)
	}

	// mode=self
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/traces/t1/flame?groupBy=service_operation&mode=self", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("status(self)=%d body=%s", w2.Code, w2.Body.String())
	}
	var self FlameNode
	if err := json.Unmarshal(w2.Body.Bytes(), &self); err != nil {
		t.Fatalf("json(self): %v", err)
	}
	if self.Value != 500 {
		t.Fatalf("root(self) value=%d want 500", self.Value)
	}
}

func TestFlame_GroupByVariants(t *testing.T) {
	body := `{"SpanId":"X","ParentSpanId":"","SpanName":"GET /foo","ServiceName":"api","start_ns":0,"end_ns":200000}` + "\n"
	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}
	r := newRouter("/api/traces/:traceId/flame", Flame(src))

	cases := []struct {
		q    string
		want string
	}{
		{"service_operation", "api:GET /foo"},
		{"service", "api"},
		{"operation", "GET /foo"},
		{"name", "GET /foo"},
	}
	for _, cse := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/traces/T/flame?groupBy="+cse.q, nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("[%s] status=%d body=%s", cse.q, w.Code, w.Body.String())
		}
		var root FlameNode
		if err := json.Unmarshal(w.Body.Bytes(), &root); err != nil {
			t.Fatalf("[%s] json: %v", cse.q, err)
		}
		if root.Name != cse.want {
			t.Fatalf("[%s] root.Name=%q want %q", cse.q, root.Name, cse.want)
		}
	}
}

func TestFlame_MultiRootAggregates(t *testing.T) {
	// Two independent roots (R1, R2) should be wrapped under "trace:<id>" with Value=sum(children)
	body := "" +
		`{"SpanId":"R1","ParentSpanId":"","SpanName":"r1","ServiceName":"s1","start_ns":0,"end_ns":100000}` + "\n" +
		`{"SpanId":"R2","ParentSpanId":"","SpanName":"r2","ServiceName":"s2","start_ns":0,"end_ns":300000}` + "\n"
	ts := fakeCH(t, body)
	defer ts.Close()

	src := &sources.Sources{CHURL: ts.URL, CHDB: "default", Client: ts.Client()}
	r := newRouter("/api/traces/:traceId/flame", Flame(src))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/traces/ZZ/flame", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var root FlameNode
	_ = json.Unmarshal(w.Body.Bytes(), &root)
	if len(root.Children) != 2 || root.Value != (100000/1000+300000/1000) {
		t.Fatalf("aggregate: %+v", root)
	}
}
