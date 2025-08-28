package traces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeCH starts a test HTTP server that pretends to be ClickHouse and returns the given body.
func fakeCH(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// emulate CH JSONEachRow: newline-delimited JSON objects
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
}

// newRouter wires only the routes a test needs.
func newRouter(path string, h gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(path, h)
	return r
}
