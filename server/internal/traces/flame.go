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

type FlameNode struct {
	Name     string      `json:"name"`
	Value    int64       `json:"value"` // microseconds
	Children []FlameNode `json:"children,omitempty"`
}

// local (internal) span shape for flame building to avoid colliding with traces.Span from get.go
type fSpan struct {
	ID       string
	Parent   string
	Name     string
	Service  string
	Start    int64 // ns
	End      int64 // ns
	Children []string
}

func Flame(src *sources.Sources) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.Param("traceId")
		groupBy := c.DefaultQuery("groupBy", "service_operation")
		mode := c.DefaultQuery("mode", "total")

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
		if src.CHUser != "" {
			req.SetBasicAuth(src.CHUser, src.CHPass)
		}
		resp, err := src.Client.Do(req)
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		type row struct {
			SpanId       string `json:"SpanId"`
			ParentSpanId string `json:"ParentSpanId"`
			SpanName     string `json:"SpanName"`
			ServiceName  string `json:"ServiceName"`
			StartNS      int64  `json:"start_ns"`
			EndNS        int64  `json:"end_ns"`
		}

		spans := map[string]*fSpan{}
		var roots []string

		rdr := bufio.NewReader(resp.Body)
		for {
			line, err := rdr.ReadBytes('\n')
			if len(line) > 0 {
				var r row
				if json.Unmarshal(line, &r) == nil {
					spans[r.SpanId] = &fSpan{
						ID:      r.SpanId,
						Parent:  r.ParentSpanId,
						Name:    r.SpanName,
						Service: r.ServiceName,
						Start:   r.StartNS,
						End:     r.EndNS,
					}
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
		}

		// wire up tree
		for _, s := range spans {
			if s.Parent != "" {
				if p, ok := spans[s.Parent]; ok {
					p.Children = append(p.Children, s.ID)
				} else {
					roots = append(roots, s.ID) // orphan
				}
			} else {
				roots = append(roots, s.ID)
			}
		}
		for _, s := range spans {
			sort.Slice(s.Children, func(i, j int) bool {
				return spans[s.Children[i]].Start < spans[s.Children[j]].Start
			})
		}

		// build flame tree
		var root FlameNode
		if len(roots) == 1 {
			root = buildFlame(spans[roots[0]], spans, groupBy, mode)
		} else {
			root = FlameNode{Name: "trace:" + traceID}
			for _, rid := range roots {
				ch := buildFlame(spans[rid], spans, groupBy, mode)
				root.Children = append(root.Children, ch)
				root.Value += ch.Value
			}
		}

		c.JSON(200, root)
	}
}

func buildFlame(s *fSpan, spans map[string]*fSpan, groupBy, mode string) FlameNode {
	totalUS := (s.End - s.Start) / 1000
	node := FlameNode{Name: label(s, groupBy), Value: totalUS}

	for _, cid := range s.Children {
		node.Children = append(node.Children, buildFlame(spans[cid], spans, groupBy, mode))
	}

	if mode == "self" {
		var childSum int64
		for _, ch := range node.Children {
			childSum += ch.Value
		}
		if childSum < node.Value {
			node.Value -= childSum
		} else {
			node.Value = 0
		}
	}

	return node
}

func label(s *fSpan, group string) string {
	switch group {
	case "service":
		return s.Service
	case "operation", "name":
		return s.Name
	default:
		if s.Service == "" {
			return s.Name
		}
		return s.Service + ":" + s.Name
	}
}
