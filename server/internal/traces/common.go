package traces

import (
  "encoding/json"
)

func strMap(m map[string]any) map[string]string {
  out := map[string]string{}
  for k, v := range m {
    if s, ok := v.(string); ok { out[k]=s } else { b,_ := json.Marshal(v); out[k]=string(b) }
  }
  return out
}
