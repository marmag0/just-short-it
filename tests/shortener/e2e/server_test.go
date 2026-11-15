package e2e

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The demo link hardcoded in main.go.
const (
	demoLink   = "test"
	demoTarget = "https://akamai.com"
)

var binary string

// Redirects are not followed, so tests see the shortener's own response.
var client = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "shortener-e2e")
	if err != nil {
		log.Fatal(err)
	}

	binary = filepath.Join(dir, "shortener")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	out, err := exec.Command("go", "build", "-o", binary, "github.com/marmag0/just-short-it/services/shortener/cmd").CombinedOutput()
	if err != nil {
		os.RemoveAll(dir)
		log.Fatalf("building shortener: %v\n%s", err, out)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// TODO: main.go still hardcodes 8888, so other values work only once it reads SHORTENER_PORT.
func port() string {
	if p := os.Getenv("SHORTENER_PORT"); p != "" {
		return p
	}
	return "8888"
}

type server struct {
	url  string
	cmd  *exec.Cmd
	logs bytes.Buffer
	done chan struct{}
	err  error
}

// startServer runs the built binary and waits until it accepts connections.
func startServer(t *testing.T) *server {
	t.Helper()

	addr := net.JoinHostPort("localhost", port())
	if conn, err := net.Dial("tcp", addr); err == nil {
		conn.Close()
		t.Fatalf("%s is already in use, stop the process listening on it or set SHORTENER_PORT", addr)
	}

	s := &server{url: "http://" + addr, done: make(chan struct{})}
	s.cmd = exec.Command(binary)
	s.cmd.Env = append(os.Environ(), "SHORTENER_PORT="+port())
	s.cmd.Stdout = &s.logs
	s.cmd.Stderr = &s.logs
	if err := s.cmd.Start(); err != nil {
		t.Fatalf("starting shortener: %v", err)
	}
	go func() {
		s.err = s.cmd.Wait()
		close(s.done)
	}()

	t.Cleanup(func() {
		select {
		case <-s.done:
		default:
			s.cmd.Process.Kill()
			<-s.done
		}
		if t.Failed() {
			t.Logf("shortener output:\n%s", s.logs.String())
		}
	})

	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case <-s.done:
			t.Fatalf("shortener exited before accepting connections: %v", s.err)
		default:
		}
		if conn, err := net.Dial("tcp", addr); err == nil {
			conn.Close()
			return s
		}
		if time.Now().After(deadline) {
			t.Fatalf("shortener is not accepting connections on %s", addr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func send(t *testing.T, method, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	resp.Body.Close()
	return resp
}

func TestRedirect(t *testing.T) {
	s := startServer(t)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			resp := send(t, method, s.url+"/"+demoLink)

			// Permanent redirects get cached by clients, so later clicks would skip telemetry.
			if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusTemporaryRedirect {
				t.Errorf("status = %d, want a temporary redirect (302 or 307)", resp.StatusCode)
			}
			if got := resp.Header.Get("Location"); got != demoTarget {
				t.Errorf("Location = %q, want %q", got, demoTarget)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	s := startServer(t)

	tests := []struct {
		name string
		path string
	}{
		{"unknown link", "/unknown"},
		{"too short", "/ab"},
		{"too long", "/" + strings.Repeat("a", 33)},
		{"disallowed character", "/te-st"},
		{"encoded slash", "/te%2Fst"},
		{"root", "/"},
		{"nested path", "/" + demoLink + "/extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := send(t, http.MethodGet, s.url+tt.path)
			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("GET %s status = %d, want %d", tt.path, resp.StatusCode, http.StatusNotFound)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	s := startServer(t)

	resp := send(t, http.MethodPost, s.url+"/"+demoLink)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
	if allow := resp.Header.Get("Allow"); !strings.Contains(allow, http.MethodGet) {
		t.Errorf("Allow = %q, want it to include GET", allow)
	}
}

func TestGracefulShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signals cannot be sent to processes on Windows")
	}

	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			s := startServer(t)
			if err := s.cmd.Process.Signal(sig); err != nil {
				t.Fatalf("sending %v: %v", sig, err)
			}

			select {
			case <-s.done:
				if s.err != nil {
					t.Errorf("shortener exited with %v after %v, want a clean exit", s.err, sig)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("shortener did not stop within 10s after %v", sig)
			}
		})
	}
}
