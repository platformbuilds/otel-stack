package traces

import (
  "bufio"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "strings"

  "github.com/gin-gonic/gin"
  "github.com/example/otel-stack-demo/internal/sources"
)

type Span struct {
  SpanID string `json:"spanId"`; ParentSpanID string `json:"parentSpanId,omitempty"`
  Name string `json:"name"`; Kind string `json:"kind"`; Service string `json:"service"`
  StartUnixNanos int64 `json:"startUnixNanos"`; EndUnixNanos int64 `json:"endUnixNanos"`
  Attributes map[string]string `json:"attributes,omitempty"`
  StatusCode string `json:"statusCode,omitempty"`; StatusMessage string `json:"statusMessage,omitempty"`
}

func Get(src *sources.Sources) gin.HandlerFunc {
  return func(c *gin.Context){
    traceID := c.Param("traceId")
    sql := fmt.Sprintf(`
      SELECT TraceId, SpanId, ParentSpanId, SpanName, SpanKind, ServiceName,
             toUnixTimestamp64Nano(Timestamp) AS start_ns,
             toUnixTimestamp64Nano(Timestamp) + (Duration * 1000000) AS end_ns,
             SpanAttributes, StatusCode, StatusMessage
      FROM %s.otel_traces
      WHERE TraceId = '%s'
      ORDER BY start_ns ASC
      FORMAT JSONEachRow
    `, src.CHDB, traceID)

    req, _ := http.NewRequest("POST", src.CHURL, strings.NewReader(sql))
    if src.CHUser != "" { req.SetBasicAuth(src.CHUser, src.CHPass) }
    resp, err := src.Client.Do(req)
    if err != nil { c.JSON(502, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close()

    type Row struct {
      SpanId string `json:"SpanId"`; ParentSpanId string `json:"ParentSpanId"`
      SpanName string `json:"SpanName"`; SpanKind string `json:"SpanKind"`; ServiceName string `json:"ServiceName"`
      StartNS int64 `json:"start_ns"`; EndNS int64 `json:"end_ns"`
      SpanAttributes map[string]any `json:"SpanAttributes"`; StatusCode string `json:"StatusCode"`; StatusMessage string `json:"StatusMessage"`
    }
    out := []Span{}
    rdr := bufio.NewReader(resp.Body)
    for {
      line, err := rdr.ReadBytes('\n')
      if len(line)>0 {
        var r Row
        if json.Unmarshal(line, &r)==nil {
          out = append(out, Span{
            SpanID: r.SpanId, ParentSpanID: r.ParentSpanId, Name: r.SpanName, Kind: r.SpanKind, Service: r.ServiceName,
            StartUnixNanos: r.StartNS, EndUnixNanos: r.EndNS, Attributes: strMap(r.SpanAttributes),
            StatusCode: r.StatusCode, StatusMessage: r.StatusMessage,
          })
        }
      }
      if err==io.EOF { break }
      if err!=nil { break }
    }
    c.JSON(200, gin.H{"traceId": traceID, "spans": out})
  }
}
