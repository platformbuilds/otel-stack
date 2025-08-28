package main

import (
  "log"
  "net/http"
  "os"

  "github.com/example/otel-stack-demo/internal/server"
)

func main() {
  addr := getenv("HTTP_ADDR", ":8080")
  s := server.New()
  log.Printf("listening on %s", addr)
  if err := http.ListenAndServe(addr, s); err != nil { log.Fatal(err) }
}

func getenv(k,d string) string { if v:=os.Getenv(k); v!="" { return v }; return d }
