package traces

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/example/otel-stack-demo/internal/sources"
	"github.com/gin-gonic/gin"
)

// Span is defined elsewhere in your package:
// type Span struct {
//   SpanID string `json:"spanId"`
//   ParentSpanID string `json:"parentSpanId,omitempty"`
//   Name string `json:"name"`
//   Kind string `json:"kind"`
//   Service string `json:"service"`
//   StartUnixNanos int64 `json:"startUnixNanos"`
//   EndUnixNanos int64 `json:"endUnixNanos"`
//   Attributes map[string]string `json:"attributes,omitempty"`
//   StatusCode string `json:"statusCode,omitempty"`
//   StatusMessage string `json:"statusMessage,omitempty"`
// }

// JSONEachRow coming from ClickHouse for flame
type flameRow struct {
	SpanId       string `json:"SpanId"`
	ParentSpanId string `json:"ParentSpanId"`
	SpanName     string `json:"SpanName"`
	ServiceName  string `json:"ServiceName"`
	StartNS      int64  `json:"start_ns"`
	EndNS        int64  `json:"end_ns"`
}

// Output node for d3-flame-graph
type FlameNode struct {
	Name     string      `json:"name"`
	Value    int64       `json:"value"` // microseconds
	Children []FlameNode `json:"children,omitempty"`
}

// NOTE on units:
//   - Most OTEL→ClickHouse pipelines store Duration in *milliseconds*.
//     If yours stores *nanoseconds*, change "+ (Duration * 1000000)" below to "+ Duration".
const flameSQL = `
SELECT
  SpanId,
  ifNull(ParentSpanId, '') AS ParentSpanId,
  SpanName AS SpanName,
  ServiceName AS ServiceName,
  toInt64(toUnixTimestamp64Nano(Timestamp)) AS start_ns,
  toInt64(toUnixTimestamp64Nano(Timestamp) + (Duration * 1000000)) AS end_ns
FROM {db:Identifier}.otel_traces
WHERE lower(TraceId) = lower({traceId:String})
ORDER BY start_ns ASC
FORMAT JSONEachRow
`

// Flame returns a flamegraph-compatible tree for a trace
func Flame(src *sources.Sources) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := strings.ToLower(c.Param("traceId"))
		groupBy := c.DefaultQuery("groupBy", "service_operation") // service|operation|name|service_operation
		mode := c.DefaultQuery("mode", "total")                   // total|self

		// Build ClickHouse HTTP request.
		// Bind both the default database (for HTTP context) and the query param {db:Identifier}.
		chURL := fmt.Sprintf("%s/?database=%s&default_format=JSONEachRow&param_traceId=%s&param_db=%s",
			strings.TrimRight(src.CHURL, "/"),
			src.CHDB, traceID, src.CHDB,
		)

		req, err := http.NewRequest(http.MethodPost, chURL, strings.NewReader(flameSQL))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "build CH request: " + err.Error()})
			return
		}
		if src.CHUser != "" || src.CHPass != "" {
			req.SetBasicAuth(src.CHUser, src.CHPass)
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")

		resp, err := src.Client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "query CH: " + err.Error()})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusMultipleChoices {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("CH %d: %s", resp.StatusCode, string(b))})
			return
		}

		// Decode JSONEachRow
		spans := make(map[string]*Span, 128)
		dec := json.NewDecoder(bufio.NewReader(resp.Body))
		for {
			var r flameRow
			if err := dec.Decode(&r); err != nil {
				if err == io.EOF {
					break
				}
				c.JSON(http.StatusBadGateway, gin.H{"error": "decode CH rows: " + err.Error()})
				return
			}
			s := &Span{
				SpanID:         r.SpanId,
				ParentSpanID:   r.ParentSpanId,
				Name:           r.SpanName,
				Service:        r.ServiceName,
				StartUnixNanos: r.StartNS,
				EndUnixNanos:   r.EndNS,
			}
			spans[s.SpanID] = s
		}

		if len(spans) == 0 {
			c.JSON(http.StatusOK, FlameNode{Name: "trace:" + traceID, Value: 0, Children: nil})
			return
		}

		// Build adjacency (parent -> children)
		children := map[string][]string{}
		roots := make([]string, 0, 4)
		for _, s := range spans {
			if s.ParentSpanID == "" || spans[s.ParentSpanID] == nil {
				roots = append(roots, s.SpanID)
			} else {
				children[s.ParentSpanID] = append(children[s.ParentSpanID], s.SpanID)
			}
		}
		sort.Strings(roots)

		// Assemble tree
		var root FlameNode
		if len(roots) == 1 {
			root = buildFlame(roots[0], spans, children, groupBy, mode)
		} else {
			root = FlameNode{Name: "trace:" + traceID}
			for _, rid := range roots {
				ch := buildFlame(rid, spans, children, groupBy, mode)
				root.Children = append(root.Children, ch)
				root.Value += ch.Value
			}
		}
		c.JSON(http.StatusOK, root)
	}
}

func buildFlame(id string, spans map[string]*Span, children map[string][]string, groupBy, mode string) FlameNode {
	s := spans[id]
	totalUS := (s.EndUnixNanos - s.StartUnixNanos) / 1000 // ns -> µs for d3-flame-graph
	node := FlameNode{Name: label(s, groupBy), Value: totalUS}

	// Recurse
	for _, cid := range children[id] {
		ch := buildFlame(cid, spans, children, groupBy, mode)
		node.Children = append(node.Children, ch)
	}

	if mode == "self" {
		var sum int64
		for _, ch := range node.Children {
			sum += ch.Value
		}
		if sum < node.Value {
			node.Value -= sum
		} else {
			node.Value = 0
		}
	}
	return node
}

func label(s *Span, group string) string {
	switch group {
	case "service":
		return s.Service
	case "operation", "name":
		return s.Name
	case "service_operation":
		if s.Service == "" {
			return s.Name
		}
		return s.Service + ":" + s.Name
	default:
		if s.Service == "" {
			return s.Name
		}
		return s.Service + ":" + s.Name
	}
}
