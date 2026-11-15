package stack

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// prerequisites runs scripts/prerequisites.sh in a copy of the repository files it reads, with env as .env.
// A nil env leaves .env out. It returns the output and the exit code.
func prerequisites(t *testing.T, env map[string]string, args ...string) (string, int) {
	t.Helper()

	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "scripts"), os.DirFS(filepath.Join(root, "scripts"))); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".env.example", "docker-compose.yml"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if env != nil {
		var b strings.Builder
		for key, value := range env {
			b.WriteString(key + "=" + value + "\n")
		}
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(b.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	out, err := exec.Command("sh", append([]string{filepath.Join(dir, "scripts", "prerequisites.sh")}, args...)...).CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func TestPrerequisites(t *testing.T) {
	requireDocker(t)

	const red, yellow, green = "\033[31m", "\033[33m", "\033[32m"
	// filled is a .env with every placeholder of .env.example replaced.
	filled := func(change func(env map[string]string)) map[string]string {
		env := exampleEnv(t)
		env["GRAFANA_ADMIN_USER"] = "someone"
		env["GRAFANA_ADMIN_PASSWORD"] = "prerequisites-test-password"
		env["CLOUDFLARE_TUNNEL_TOKEN"] = "prerequisites-test-token"
		if change != nil {
			change(env)
		}
		return env
	}

	tests := []struct {
		name  string
		env   map[string]string
		args  []string
		code  int
		color string
		want  string
	}{
		{"filled in", filled(nil), nil, 0, green, "[+] GRAFANA_ADMIN_PASSWORD is set"},
		{"filled in with tunnel", filled(nil), []string{"--tunnel"}, 0, green, "[+] CLOUDFLARE_TUNNEL_TOKEN is set"},
		{"placeholders", exampleEnv(t), nil, 0, yellow, "[i] GRAFANA_ADMIN_PASSWORD still has the placeholder from .env.example"},
		{"placeholders with tunnel", exampleEnv(t), []string{"--tunnel"}, 1, red, "[!] GRAFANA_ADMIN_PASSWORD still has the placeholder from .env.example"},
		{"no token with tunnel", filled(func(env map[string]string) { env["CLOUDFLARE_TUNNEL_TOKEN"] = "" }), []string{"--tunnel"}, 1, red, "[!] CLOUDFLARE_TUNNEL_TOKEN is empty"},
		{"no .env", nil, nil, 1, red, "[!] .env is missing"},
		{"missing port", filled(func(env map[string]string) { delete(env, "SHORTENER_PORT") }), nil, 1, red, "[!] SHORTENER_PORT must be a port"},
		{"invalid port", filled(func(env map[string]string) { env["GRAFANA_PORT"] = "70000" }), nil, 1, red, "[!] GRAFANA_PORT must be a port"},
		{"reused port", filled(func(env map[string]string) { env["GRAFANA_PORT"] = env["SHORTENER_PORT"] }), nil, 1, red, "reuses port " + exampleEnv(t)["SHORTENER_PORT"]},
		{"variable required by compose", filled(func(env map[string]string) { env["GRAFANA_ADMIN_PASSWORD"] = "" }), nil, 1, red, "Set GRAFANA_ADMIN_PASSWORD in .env"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, code := prerequisites(t, tt.env, tt.args...)
			if code != tt.code {
				t.Errorf("exit code = %d, want %d", code, tt.code)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("output does not contain %q", tt.want)
			}
			lines := strings.Split(strings.TrimSpace(out), "\n")
			if last := lines[len(lines)-1]; !strings.HasPrefix(last, tt.color) {
				t.Errorf("last line = %q, want it to start with %q", last, tt.color)
			}
			if t.Failed() {
				t.Logf("output:\n%s", out)
			}
		})
	}

	t.Run("unknown flag", func(t *testing.T) {
		if out, code := prerequisites(t, filled(nil), "--tunel"); code != 2 || !strings.Contains(out, "usage:") {
			t.Errorf("exit code = %d, output = %q, want 2 and the usage", code, out)
		}
	})
}
