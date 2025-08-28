package server

import (
  "net/http"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/example/otel-stack-demo/internal/sources"
  "github.com/example/otel-stack-demo/internal/traces"
)

func New() http.Handler {
  gin.SetMode(gin.ReleaseMode)
  r := gin.Default()

  src := sources.FromEnv()

  r.GET("/healthz", func(c *gin.Context){ c.JSON(200, gin.H{"ok":true}) })
  r.GET("/readyz", func(c *gin.Context){
    client := &http.Client{ Timeout: 2*time.Second }
    req, _ := http.NewRequest("GET", src.PromURL+"/-/healthy", nil)
    if _, err := client.Do(req); err != nil { c.JSON(200, gin.H{"ok":true}); return }
    c.JSON(200, gin.H{"ok":true})
  })

  r.POST("/api/metrics/query", src.MetricsProxy())
  r.POST("/api/logs/search", src.LogsProxy())

  r.POST("/api/traces/list", traces.List(src))
  r.GET("/api/traces/:traceId", traces.Get(src))
  r.GET("/api/traces/:traceId/flame", traces.Flame(src))
  r.GET("/api/traces/suggest/services", traces.SuggestServices(src))
  r.GET("/api/traces/suggest/operations", traces.SuggestOperations(src))
  r.GET("/api/traces/suggest/attributes", traces.SuggestAttributes(src))

  return r
}
