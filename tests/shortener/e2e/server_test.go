package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

	out, err := exec.Command("go", "build", "-cover", "-o", binary, "github.com/marmag0/just-short-it/services/shortener/cmd").CombinedOutput()
	if err != nil {
		os.RemoveAll(dir)
		log.Fatalf("building shortener: %v\n%s", err, out)
	}

	// The binary is built with -cover. Set GOCOVERDIR to keep its coverage data, otherwise it is discarded with dir.
	if os.Getenv("GOCOVERDIR") == "" {
		os.Setenv("GOCOVERDIR", dir)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// freePort returns a port nothing listens on, so tests do not collide with a running stack.
func freePort(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
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

	port := freePort(t)
	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(), "SHORTENER_PORT="+port)
	return start(t, cmd, port)
}

// start runs cmd, which must make the shortener listen on port, and waits until it accepts connections.
func start(t *testing.T, cmd *exec.Cmd, port string) *server {
	t.Helper()

	addr := net.JoinHostPort("localhost", port)
	s := &server{url: "http://" + addr, cmd: cmd, done: make(chan struct{})}
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
			// An interrupt lets the binary write its coverage data, Kill is the fallback on Windows.
			if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
				s.cmd.Process.Kill()
			}
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

func TestForcedShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signals cannot be sent to processes on Windows")
	}

	s := startServer(t)
	conn, err := net.Dial("tcp", strings.TrimPrefix(s.url, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// The announced body never arrives, so the request stays active past the 5s shutdown timeout.
	fmt.Fprintf(conn, "GET /%s HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1\r\n\r\n", demoLink)
	time.Sleep(200 * time.Millisecond)
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case <-s.done:
		var exitErr *exec.ExitError
		if !errors.As(s.err, &exitErr) || exitErr.ExitCode() != 1 {
			t.Errorf("shortener exited with %v, want exit status 1", s.err)
		}
		if !strings.Contains(s.logs.String(), "server forced to shutdown") {
			t.Errorf("output does not report the forced shutdown:\n%s", s.logs.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("shortener did not stop within 10s")
	}
}

func TestPortFromEnvFile(t *testing.T) {
	port := freePort(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SHORTENER_PORT="+port+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// An empty SHORTENER_PORT counts as unset, so the port has to come from .env in the working directory.
	cmd := exec.Command(binary)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SHORTENER_PORT=")
	s := start(t, cmd, port)

	resp := send(t, http.MethodGet, s.url+"/"+demoLink)
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
}

// failToStart runs the binary in dir with env and checks that it exits with status 1 and logs want.
func failToStart(t *testing.T, dir string, env []string, want string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("shortener exited with %v, want exit status 1\n%s", err, out)
	}
	if !strings.Contains(string(out), want) {
		t.Errorf("output = %q, want it to contain %q", out, want)
	}
}

func TestMissingPort(t *testing.T) {
	tests := []struct {
		name    string
		envFile string
	}{
		{"no .env", ""},
		{".env without SHORTENER_PORT", "GRAFANA_PORT=3000\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.envFile != "" {
				if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(tt.envFile), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			failToStart(t, dir, append(os.Environ(), "SHORTENER_PORT="), "SHORTENER_PORT is not set")
		})
	}
}

func TestInvalidPort(t *testing.T) {
	failToStart(t, t.TempDir(), append(os.Environ(), "SHORTENER_PORT=not-a-port"), "server failed to start")
}
