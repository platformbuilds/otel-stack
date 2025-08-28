package traces

import (
  "bufio"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "sort"
  "strings"

  "github.com/gin-gonic/gin"
  "github.com/example/otel-stack-demo/internal/sources"
)

type FlameNode struct { Name string `json:"name"`; Value int64 `json:"value"`; Children []FlameNode `json:"children,omitempty"` }

func Flame(src *sources.Sources) gin.HandlerFunc {
  return func(c *gin.Context){
    traceID := c.Param("traceId")
    groupBy := c.DefaultQuery("groupBy","service_operation")
    mode := c.DefaultQuery("mode","total")

    sql := fmt.Sprintf(`
      SELECT SpanId, ParentSpanId, SpanName, ServiceName,
             toUnixTimestamp64Nano(Timestamp) AS start_ns,
             toUnixTimestamp64Nano(Timestamp) + (Duration * 1000000) AS end_ns
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

    type Row struct { SpanId, ParentSpanId, SpanName, ServiceName string; StartNS, EndNS int64 }
    type Span struct{ id,parent,name,service string; start,end int64; children []string }
    spans := map[string]*Span{}; roots := []string{}
    rdr := bufio.NewReader(resp.Body)
    for {
      line, err := rdr.ReadBytes('\n')
      if len(line)>0 {
        var r Row; if json.Unmarshal(line, &r)==nil {
          spans[r.SpanId] = &Span{id:r.SpanId, parent:r.ParentSpanId, name:r.SpanName, service:r.ServiceName, start:r.StartNS, end:r.EndNS}
        }
      }
      if err==io.EOF { break }
      if err!=nil { break }
    }
    for _, s := range spans {
      if s.parent != "" {
        if p, ok := spans[s.parent]; ok { p.children = append(p.children, s.id) } else { roots = append(roots, s.id) }
      } else { roots = append(roots, s.id) }
    }
    for _, s := range spans { sort.Slice(s.children, func(i,j int)bool{ return spans[s.children[i]].start < spans[s.children[j]].start }) }

    var root FlameNode
    if len(roots)==1 { root = buildFlame(spans[roots[0]], spans, groupBy, mode) } else {
      root = FlameNode{Name:"trace:"+traceID}
      for _, rid := range roots {
        ch := buildFlame(spans[rid], spans, groupBy, mode); root.Children = append(root.Children, ch); root.Value += ch.Value
      }
    }
    c.JSON(200, root)
  }
}

func buildFlame(s *Span, spans map[string]*Span, groupBy, mode string) FlameNode {
  totalUS := (s.end - s.start)/1000
  node := FlameNode{Name: label(s, groupBy), Value: totalUS}
  for _, cid := range s.children { node.Children = append(node.Children, buildFlame(spans[cid], spans, groupBy, mode)) }
  if mode=="self" {
    var sum int64; for _, ch := range node.Children { sum += ch.Value }
    if sum < node.Value { node.Value -= sum } else { node.Value = 0 }
  }
  return node
}
func label(s *Span, group string) string {
  switch group {
  case "service": return s.service
  case "operation","name": return s.name
  default: if s.service=="" { return s.name }; return s.service+":"+s.name
  }
}
