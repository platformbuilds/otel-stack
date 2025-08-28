package traces

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/otel-stack-demo/internal/sources"
	"github.com/gin-gonic/gin"
)

type TraceListReq struct {
	From    float64 `json:"from"`
	To      float64 `json:"to"`
	Filters struct {
		Service   []string `json:"service"`
		Operation []string `json:"operation"`
		Status    []string `json:"status"`
		Duration  struct {
			Gte *float64 `json:"gte"`
			Lte *float64 `json:"lte"`
		} `json:"durationMs"`
	} `json:"filters"`
	Sort struct {
		By    string `json:"by"`
		Order string `json:"order"`
	} `json:"sort"`
	Page struct {
		Size int `json:"size"`
	} `json:"page"`
}

func List(src *sources.Sources) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r TraceListReq
		if err := c.BindJSON(&r); err != nil {
			c.JSON(400, gin.H{"error": "bad json"})
			return
		}
		if r.To == 0 {
			r.To = float64(time.Now().Unix())
		}
		if r.From == 0 {
			r.From = r.To - 3600
		}
		if r.Page.Size <= 0 || r.Page.Size > 500 {
			r.Page.Size = 100
		}
		by := strings.ToLower(r.Sort.By)
		if by == "" {
			by = "duration"
		}
		order := strings.ToUpper(r.Sort.Order)
		if order != "ASC" {
			order = "DESC"
		}

		where := []string{
			fmt.Sprintf("StartTs BETWEEN toDateTime(%d) AND toDateTime(%d)", int64(r.From), int64(r.To)),
		}
		if len(r.Filters.Service) > 0 {
			where = append(where, "RootService IN ("+joinQuoted(r.Filters.Service)+")")
		}
		if len(r.Filters.Operation) > 0 {
			where = append(where, "RootOperation IN ("+joinQuoted(r.Filters.Operation)+")")
		}
		if len(r.Filters.Status) > 0 {
			where = append(where, "Status IN ("+joinQuoted(r.Filters.Status)+")")
		}
		if r.Filters.Duration.Gte != nil {
			where = append(where, fmt.Sprintf("DurationMs >= %f", *r.Filters.Duration.Gte))
		}
		if r.Filters.Duration.Lte != nil {
			where = append(where, fmt.Sprintf("DurationMs <= %f", *r.Filters.Duration.Lte))
		}

		orderExpr := "DurationMs"
		switch by {
		case "start":
			orderExpr = "StartTs"
		case "spancount":
			orderExpr = "SpanCount"
		}

		sql := fmt.Sprintf(`
      SELECT TraceId, StartTs, DurationMs, RootService, RootOperation, Status, SpanCount, TopService, TopServiceMs
      FROM %s.trace_roots
      WHERE %s
      ORDER BY %s %s
      LIMIT %d
      FORMAT JSONEachRow
    `, src.CHDB, strings.Join(where, " AND "), orderExpr, order, r.Page.Size)

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
		b, _ := io.ReadAll(resp.Body)

		type Row struct {
			TraceId       string  `json:"TraceId"`
			StartTs       string  `json:"StartTs"`
			DurationMs    float64 `json:"DurationMs"`
			RootService   string  `json:"RootService"`
			RootOperation string  `json:"RootOperation"`
			Status        string  `json:"Status"`
			SpanCount     int     `json:"SpanCount"`
			TopService    string  `json:"TopService"`
			TopServiceMs  float64 `json:"TopServiceMs"`
		}

		dec := json.NewDecoder(strings.NewReader(string(b)))
		items := []map[string]any{}
		for {
			var row Row
			if err := dec.Decode(&row); err != nil {
				break
			}
			items = append(items, map[string]any{
				"traceId":       row.TraceId,
				"startTs":       row.StartTs,
				"durationMs":    row.DurationMs,
				"rootService":   row.RootService,
				"rootOperation": row.RootOperation,
				"status":        row.Status,
				"spanCount":     row.SpanCount,
				"svcBreakdown":  [][2]any{{row.TopService, row.TopServiceMs}},
			})
		}
		c.JSON(200, gin.H{"items": items})
	}
}

// joinQuoted returns "'a', 'b', 'c'" with single quotes safely escaped for SQL.
func joinQuoted(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(vals))
	for _, v := range vals {
		escaped = append(escaped, "'"+strings.ReplaceAll(v, "'", "''")+"'")
	}
	return strings.Join(escaped, ", ")
}
