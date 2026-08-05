// Package healthcheck lets a distroless container probe itself.
//
// Six Facile APIs run gcr.io/distroless/static-debian12: no shell, no wget,
// no curl, so a HEALTHCHECK can only work by re-executing the app binary.
// Handle it as the first thing in main:
//
//	func main() {
//		if healthcheck.Handle(os.Args) {
//			return
//		}
//		...
//	}
//
// and declare it in compose as:
//
//	healthcheck:
//	  test: ["CMD", "/app", "healthcheck"]
package healthcheck

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// Timeout bounds the probe request.
const Timeout = 3 * time.Second

// Handle runs the probe and exits when args asks for it, and reports whether
// it did. It returns false for a normal start, so main can continue.
//
// The probe targets 127.0.0.1 rather than localhost: in these containers
// localhost resolves to ::1 first while the server binds 0.0.0.0, so a
// localhost probe fails against a healthy process.
func Handle(args []string) bool {
	if len(args) < 2 || args[1] != "healthcheck" {
		return false
	}

	if err := Probe(port()); err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
	return true
}

// Probe requests /health on the given port and reports whether it answered 2xx.
func Probe(port string) error {
	client := &http.Client{Timeout: Timeout}

	response, err := client.Get("http://" + net.JoinHostPort("127.0.0.1", port) + "/health")
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("/health answered %d", response.StatusCode)
	}
	return nil
}

func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return "8080"
}
