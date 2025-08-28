package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/example/otel-stack-demo/internal/server"
)

// ---- getenv() behavior ----

func Test_getenv_DefaultWhenUnset(t *testing.T) {
	const key = "UNIT_TEST_HTTP_ADDR"
	_ = os.Unsetenv(key)

	got := getenv(key, ":8080")
	if got != ":8080" {
		t.Fatalf("getenv()=%q want %q", got, ":8080")
	}
}

func Test_getenv_EnvWins(t *testing.T) {
	const key = "UNIT_TEST_HTTP_ADDR"
	_ = os.Setenv(key, "127.0.0.1:9999")
	defer os.Unsetenv(key)

	got := getenv(key, ":8080")
	if got != "127.0.0.1:9999" {
		t.Fatalf("getenv()=%q want %q", got, "127.0.0.1:9999")
	}
}

// ---- Address syntax / validity table ----
// We don't call main() to bind a port; instead we validate split/Listen feasibility.

// getIPPortHint returns a helpful message to display on failures,
// covering valid IPv4/IPv6 ranges, common examples, and port rules.
func getIPPortHint(addr string) string {
	var family string
	if strings.HasPrefix(addr, "[") || strings.Contains(addr, "::") {
		family = "IPv6"
	} else {
		family = "IPv4"
	}

	return "\n\n" +
		"👉 Valid address tips:\n" +
		"• Port: 1–65535; use :0 for ephemeral. \"\" (empty) is allowed and means :0 (bind any interface, ephemeral port).\n" +
		"• IPv4 examples: 127.0.0.1:8080, 0.0.0.0:9090, localhost:3000\n" +
		"  - Octets are 0–255 (e.g. 192.168.1.10:443). RFC1918 ranges are 10.0.0.0/8, 172.16.0.0–172.31.255.255, 192.168.0.0/16.\n" +
		"• IPv6 examples: [::1]:8080, [::]:0\n" +
		"  - Use square brackets around IPv6 literals when specifying ports.\n" +
		"• Common good values: :8080, 127.0.0.1:0, 0.0.0.0:65535, [::1]:8081\n" +
		"• Common invalid values: 0.0.0.0:65536 (port overflow), :abc (non-numeric), 999.999.999.999:80 (bad IP)\n" +
		"Detected family for \"" + addr + "\": " + family + "\n"
}

func Test_HTTPAddr_SyntaxAndBindability(t *testing.T) {
	type tc struct {
		addr  string
		valid bool // whether net.Listen("tcp", addr) should succeed
	}
	cases := []tc{
		{":8080", true},       // empty host, fixed port
		{"127.0.0.1:0", true}, // ephemeral port
		{"localhost:9090", true},
		{"0.0.0.0:65535", true},
		{"[::1]:8081", true},          // IPv6 loopback
		{"0.0.0.0:65536", false},      // port overflow
		{":abc", false},               // non-numeric port
		{"bad:port", false},           // malformed host+port
		{"999.999.999.999:80", false}, // invalid IPv4
		{"", true},                    // "" == ":0" (bind ephemeral on all interfaces)
	}

	for _, cse := range cases {
		// Optional extra syntax check: SplitHostPort for addresses that contain a colon
		if cse.addr != "" && strings.Contains(cse.addr, ":") && cse.valid {
			if _, _, err := net.SplitHostPort(cse.addr); err != nil {
				t.Fatalf("[%s] unexpected split error: %v%s", cse.addr, err, getIPPortHint(cse.addr))
			}
		}

		ln, err := net.Listen("tcp", cse.addr)
		if cse.valid {
			if err != nil {
				t.Fatalf("[%s] expected to bind, got: %v%s", cse.addr, err, getIPPortHint(cse.addr))
			}
			_ = ln.Close()
		} else {
			if err == nil {
				t.Fatalf("[%s] expected binding error but succeeded%s", cse.addr, getIPPortHint(cse.addr))
			}
		}
	}
}

// ---- Sanity: New() handler serves /healthz ----

func Test_ServerHealthz_FromMainWiring(t *testing.T) {
	h := server.New() // what main() would serve via http.ListenAndServe
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d want 200", resp.StatusCode)
	}
}
