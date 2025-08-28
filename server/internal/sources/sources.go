package sources

import (
  "fmt"
  "io"
  "net/http"
  "net/url"
  "os"
  "strings"
  "time"

  "github.com/gin-gonic/gin"
)

type Sources struct {
  PromURL  string
  VLogsURL string
  CHURL    string
  CHUser   string
  CHPass   string
  CHDB     string
  Client   *http.Client
}

func FromEnv() *Sources {
  return &Sources{
    PromURL: getenv("PROM_URL","http://localhost:9090"),
    VLogsURL: getenv("VLOGS_URL","http://localhost:9428"),
    CHURL: getenv("CH_HTTP_URL","http://localhost:8123"),
    CHUser: getenv("CH_USER","default"),
    CHPass: getenv("CH_PASS",""),
    CHDB: getenv("CH_DATABASE","default"),
    Client: &http.Client{ Timeout: 20 * time.Second },
  }
}

func getenv(k,d string) string { if v:=os.Getenv(k); v!="" { return v }; return d }

// ---- Proxies ----
type metricsReq struct{ Query string `json:"query"`; Start float64 `json:"start"`; End float64 `json:"end"`; Step float64 `json:"step"` }

func (s *Sources) MetricsProxy() gin.HandlerFunc {
  return func(c *gin.Context){
    var r metricsReq
    if err := c.BindJSON(&r); err != nil { c.JSON(400, gin.H{"error":"bad json"}); return }
    if r.End==0 { r.End = float64(time.Now().Unix()) }
    if r.Start==0 { r.Start = r.End - 3600 }
    if r.Step==0 { r.Step = 60 }
    v := url.Values{}
    v.Set("query", r.Query); v.Set("start", fmt.Sprintf("%f", r.Start))
    v.Set("end", fmt.Sprintf("%f", r.End)); v.Set("step", fmt.Sprintf("%f", r.Step))
    resp, err := s.Client.Get(s.PromURL + "/api/v1/query_range?" + v.Encode())
    if err != nil { c.JSON(502, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close(); b,_ := io.ReadAll(resp.Body)
    c.Data(resp.StatusCode, "application/json", b)
  }
}

type logsReq struct{ Query string `json:"query"` }
func (s *Sources) LogsProxy() gin.HandlerFunc {
  return func(c *gin.Context){
    var r logsReq
    if err := c.BindJSON(&r); err != nil { c.JSON(400, gin.H{"error":"bad json"}); return }
    form := url.Values{}; form.Set("query", r.Query)
    req, _ := http.NewRequest("POST", s.VLogsURL+"/select/logsql/query", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type","application/x-www-form-urlencoded")
    resp, err := s.Client.Do(req)
    if err != nil { c.JSON(502, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close(); b,_ := io.ReadAll(resp.Body)
    c.Data(resp.StatusCode, "application/json", b)
  }
}
