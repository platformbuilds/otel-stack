package traces

import (
  "fmt"
  "io"
  "net/http"
  "strings"

  "github.com/gin-gonic/gin"
  "github.com/example/otel-stack-demo/internal/sources"
)

func SuggestServices(src *sources.Sources) gin.HandlerFunc {
  return func(c *gin.Context){
    q := c.Query("q")
    where := "1=1"
    if q != "" { where = fmt.Sprintf("ServiceName ILIKE '%%%s%%'", strings.ReplaceAll(q, "'", "''")) }
    sql := fmt.Sprintf(`
      SELECT ServiceName, sum(Cnt) AS c
      FROM %s.service_suggest
      WHERE WindowStart > now() - INTERVAL 24 HOUR AND %s
      GROUP BY ServiceName ORDER BY c DESC LIMIT 20 FORMAT JSONEachRow
    `, src.CHDB, where)
    proxy(c, src, sql)
  }
}
func SuggestOperations(src *sources.Sources) gin.HandlerFunc {
  return func(c *gin.Context){
    q := c.Query("q")
    where := "1=1"
    if q != "" { where = fmt.Sprintf("SpanName ILIKE '%%%s%%'", strings.ReplaceAll(q, "'", "''")) }
    sql := fmt.Sprintf(`
      SELECT SpanName, sum(Cnt) AS c
      FROM %s.operation_suggest
      WHERE WindowStart > now() - INTERVAL 24 HOUR AND %s
      GROUP BY SpanName ORDER BY c DESC LIMIT 20 FORMAT JSONEachRow
    `, src.CHDB, where)
    proxy(c, src, sql)
  }
}
func SuggestAttributes(src *sources.Sources) gin.HandlerFunc {
  return func(c *gin.Context){
    key := c.Query("key"); q := c.Query("q")
    if key == "" { c.JSON(400, gin.H{"error":"key required"}); return }
    where := fmt.Sprintf("Key = '%s'", strings.ReplaceAll(key, "'", "''"))
    if q != "" { where += fmt.Sprintf(" AND Val ILIKE '%%%s%%'", strings.ReplaceAll(q, "'", "''")) }
    sql := fmt.Sprintf(`
      SELECT Val, sum(Cnt) AS c
      FROM %s.attr_values
      WHERE WindowStart > now() - INTERVAL 24 HOUR AND %s
      GROUP BY Val ORDER BY c DESC LIMIT 20 FORMAT JSONEachRow
    `, src.CHDB, where)
    proxy(c, src, sql)
  }
}

func proxy(c *gin.Context, src *sources.Sources, sql string){
  req, _ := http.NewRequest("POST", src.CHURL, strings.NewReader(sql))
  if src.CHUser != "" { req.SetBasicAuth(src.CHUser, src.CHPass) }
  resp, err := src.Client.Do(req)
  if err != nil { c.JSON(502, gin.H{"error": err.Error()}); return }
  defer resp.Body.Close(); b,_ := io.ReadAll(resp.Body)
  c.Data(200, "application/json", b)
}
