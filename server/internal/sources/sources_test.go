package sources

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// --- Helpers ---

// restoreEnv resets given keys after the test finishes.
func restoreEnv(t *testing.T, keys ...string) {
	t.Helper()
	prev := map[string]string{}
	for _, k := range keys {
		prev[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, v := range prev {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	})
}

// route registers a single route and returns a gin.Engine for testing.
func route(method, path string, h gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	switch strings.ToUpper(method) {
	case http.MethodGet:
		r.GET(path, h)
	case http.MethodPost:
		r.POST(path, h)
	default:
		r.Any(path, h)
	}
	return r
}

// readAll reads the body and returns string.
func readAll(t *testing.T, r *http.Request) string {
	t.Helper()
	b, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	return string(b)
}

// --- Tests ---

func TestFromEnv_DefaultsAndOverrides(t *testing.T) {
	restoreEnv(t,
		"PROM_URL", "VLOGS_URL", "CH_HTTP_URL",
		"CH_USER", "CH_PASS", "CH_DATABASE",
	)

	// Clear all -> defaults kick in
	_ = os.Unsetenv("PROM_URL")
	_ = os.Unsetenv("VLOGS_URL")
	_ = os.Unsetenv("CH_HTTP_URL")
	_ = os.Unsetenv("CH_USER")
	_ = os.Unsetenv("CH_PASS")
	_ = os.Unsetenv("CH_DATABASE")

	s := FromEnv()
	if s.PromURL != "http://localhost:9090" {
		t.Fatalf("PromURL default = %q", s.PromURL)
	}
	if s.VLogsURL != "http://localhost:9428" {
		t.Fatalf("VLogsURL default = %q", s.VLogsURL)
	}
	if s.CHURL != "http://localhost:8123" || s.CHUser != "default" || s.CHDB != "default" {
		t.Fatalf("CH defaults = %+v", *s)
	}
	if s.Client == nil || s.Client.Timeout <= 0 {
		t.Fatalf("Client should be initialized with timeout")
	}

	// Set overrides
	_ = os.Setenv("PROM_URL", "http://prom.test:9090")
	_ = os.Setenv("VLOGS_URL", "http://logs.test:9428")
	_ = os.Setenv("CH_HTTP_URL", "http://ch.test:8123")
	_ = os.Setenv("CH_USER", "alice")
	_ = os.Setenv("CH_PASS", "secret")
	_ = os.Setenv("CH_DATABASE", "observability")

	s2 := FromEnv()
	if s2.PromURL != "http://prom.test:9090" || s2.VLogsURL != "http://logs.test:9428" {
		t.Fatalf("env overrides not applied: %+v", *s2)
	}
	if s2.CHURL != "http://ch.test:8123" || s2.CHUser != "alice" || s2.CHPass != "secret" || s2.CHDB != "observability" {
		t.Fatalf("CH env overrides not applied: %+v", *s2)
	}
}

func TestMetricsProxy_ForwardsQueryRangeAndEchoesUpstream(t *testing.T) {
	// Fake Prometheus that validates query params and returns a JSON body
	var captured url.Values
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v1/query_range") {
			t.Fatalf("path=%s", r.URL.Path)
		}
		captured = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
	defer up.Close()

	s := &Sources{
		PromURL: up.URL,
		Client:  &http.Client{Timeout: 3 * time.Second},
	}

	r := route("POST", "/api/metrics/query", s.MetricsProxy())

	// Client request
	body := map[string]any{
		"query": "rate(http_requests_total[5m])",
		"start": float64(1700000000),
		"end":   float64(1700000600),
		"step":  float64(60),
	}
	bs, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/metrics/query", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	// ensure upstream received expected params
	if captured.Get("query") != "rate(http_requests_total[5m])" {
		t.Fatalf("captured query=%q", captured.Get("query"))
	}
	if captured.Get("start") == "" || captured.Get("end") == "" || captured.Get("step") == "" {
		t.Fatalf("missing time params: %v", captured)
	}
	// response should be the upstream JSON
	if !strings.Contains(w.Body.String(), `"status":"success"`) {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestMetricsProxy_SetsDefaultsWhenMissing(t *testing.T) {
	// Upstream echoes back whatever so we just check it was called.
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// When start/end/step omitted, handler should fill them.
		if q.Get("query") != "up" || q.Get("start") == "" || q.Get("end") == "" || q.Get("step") == "" {
			t.Fatalf("defaults not applied: %v", q)
		}
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer up.Close()

	s := &Sources{PromURL: up.URL, Client: up.Client()}
	r := route("POST", "/api/metrics/query", s.MetricsProxy())

	req := httptest.NewRequest("POST", "/api/metrics/query", bytes.NewBufferString(`{"query":"up"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestMetricsProxy_UpstreamErrorReturns502(t *testing.T) {
	// Point to an unroutable address to trigger dial error.
	s := &Sources{PromURL: "http://127.0.0.1:0", Client: &http.Client{Timeout: 100 * time.Millisecond}}
	r := route("POST", "/api/metrics/query", s.MetricsProxy())

	req := httptest.NewRequest("POST", "/api/metrics/query", bytes.NewBufferString(`{"query":"up"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 502 {
		t.Fatalf("want 502 got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLogsProxy_ForwardsFormAndEchoesUpstream(t *testing.T) {
	// Fake VictoriaLogs that checks method and body (form-urlencoded)
	var gotCT string
	var gotBody string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST got %s", r.Method)
		}
		if r.URL.Path != "/select/logsql/query" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		gotCT = r.Header.Get("Content-Type")
		gotBody = readAll(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hits":42}`))
	}))
	defer up.Close()

	s := &Sources{VLogsURL: up.URL, Client: up.Client()}
	r := route("POST", "/api/logs/search", s.LogsProxy())

	req := httptest.NewRequest("POST", "/api/logs/search", bytes.NewBufferString(`{"query":"error _time:5m"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("upstream content-type=%q", gotCT)
	}
	// Body should be URL-encoded form with "query=..."
	if !strings.Contains(gotBody, "query=error+_time%3A5m") && !strings.Contains(gotBody, "query=error%20_time%3A5m") {
		t.Fatalf("form body not encoded: %q", gotBody)
	}
	if !strings.Contains(w.Body.String(), `"hits":42`) {
		t.Fatalf("unexpected proxy body: %s", w.Body.String())
	}
}

func TestLogsProxy_UpstreamErrorReturns502(t *testing.T) {
	s := &Sources{VLogsURL: "http://127.0.0.1:0", Client: &http.Client{Timeout: 100 * time.Millisecond}}
	r := route("POST", "/api/logs/search", s.LogsProxy())

	req := httptest.NewRequest("POST", "/api/logs/search", bytes.NewBufferString(`{"query":"info"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 502 {
		t.Fatalf("want 502 got %d body=%s", w.Code, w.Body.String())
	}
}
