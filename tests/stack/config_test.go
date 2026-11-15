package stack

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// root is the repository root with docker-compose.yml and .env.example.
const root = "../.."

// requireDocker skips the test when Docker is not available.
func requireDocker(t *testing.T) {
	t.Helper()

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("Docker is not available: %v", err)
	}
}

// exampleEnv returns the variables from .env.example, the template of every .env.
func exampleEnv(t *testing.T) map[string]string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	for line := range strings.Lines(string(data)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		env[key] = value
	}
	return env
}

// portKeys returns the sorted names of the port variables in env.
func portKeys(env map[string]string) []string {
	var keys []string
	for key := range env {
		if strings.HasSuffix(key, "_PORT") {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

// compose returns a docker compose command run from the repository root with env in place of .env.
func compose(t *testing.T, env map[string]string, args ...string) *exec.Cmd {
	t.Helper()

	// os.DevNull keeps compose away from the root .env, and inherited variables could override env.
	cmd := exec.Command("docker", append([]string{"compose", "--env-file", os.DevNull}, args...)...)
	cmd.Dir = root
	example := exampleEnv(t)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if _, ok := example[key]; !ok {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	return cmd
}

// run runs cmd and returns its stdout, failing the test with its stderr on error.
func run(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			out = exitErr.Stderr
		}
		t.Fatalf("%s: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}
	return out
}

type service struct {
	Image string
	Ports []struct {
		HostIP    string `json:"host_ip"`
		Target    int
		Published string
	}
	Environment map[string]string
	Command     []string
	Build       *struct {
		Args map[string]string
	}
}

// config returns the services of docker compose config, including the tunnel profile.
func config(t *testing.T, env map[string]string) map[string]service {
	t.Helper()

	var c struct {
		Services map[string]service
	}
	if err := json.Unmarshal(run(t, compose(t, env, "--profile", "tunnel", "config", "--format", "json")), &c); err != nil {
		t.Fatal(err)
	}
	return c.Services
}

func TestComposePortsFromEnv(t *testing.T) {
	requireDocker(t)

	// Distinct values trace every published port back to its variable.
	env := exampleEnv(t)
	keys := map[string]string{}
	for i, key := range portKeys(env) {
		env[key] = strconv.Itoa(20001 + i)
		keys[env[key]] = key
	}

	published := map[string]bool{}
	for name, s := range config(t, env) {
		for _, p := range s.Ports {
			key, ok := keys[p.Published]
			switch {
			case !ok:
				t.Errorf("%s publishes port %s, which no *_PORT variable in .env.example sets", name, p.Published)
			case p.HostIP != "127.0.0.1":
				t.Errorf("%s publishes %s on %q, want 127.0.0.1", name, key, p.HostIP)
			case strconv.Itoa(p.Target) != p.Published:
				t.Errorf("%s maps %s to container port %d, want the same port inside the container", name, key, p.Target)
			case !slices.Contains(slices.Collect(maps.Values(s.Environment)), p.Published) && !slices.ContainsFunc(s.Command, func(arg string) bool { return strings.HasSuffix(arg, ":"+p.Published) }):
				t.Errorf("%s does not pass %s to its container, so it cannot listen on it", name, key)
			case s.Build != nil && !slices.Contains(slices.Collect(maps.Values(s.Build.Args)), p.Published):
				t.Errorf("%s does not pass %s to its image build, so the image cannot expose it", name, key)
			}
			published[key] = true
		}
	}
	for _, key := range portKeys(env) {
		if !published[key] {
			t.Errorf("%s is set in .env.example, but no service publishes it", key)
		}
	}
}

func TestComposeRequiresPorts(t *testing.T) {
	requireDocker(t)

	for _, key := range portKeys(exampleEnv(t)) {
		t.Run(key, func(t *testing.T) {
			env := exampleEnv(t)
			delete(env, key)
			out, err := compose(t, env, "--profile", "tunnel", "config", "--quiet").CombinedOutput()
			if err == nil {
				t.Fatalf("docker compose config accepted a missing %s", key)
			}
			if want := "Set " + key + " in .env"; !strings.Contains(string(out), want) {
				t.Errorf("output = %q, want the hint %q", out, want)
			}
		})
	}
}

func TestServiceConfigs(t *testing.T) {
	requireDocker(t)

	services := config(t, exampleEnv(t))
	tests := []struct {
		service string
		file    string
		args    []string
	}{
		{"loki", "services/loki/config.yml", []string{"-config.file=/config", "-verify-config"}},
		{"tempo", "services/tempo/config.yml", []string{"-config.file=/config", "-config.verify=true"}},
		{"alloy", "services/alloy/config.alloy", []string{"validate", "/config"}},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			file, err := filepath.Abs(filepath.Join(root, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			// The image from compose validates the config against the version the stack runs.
			args := append([]string{"run", "--rm", "--volume", file + ":/config:ro", services[tt.service].Image}, tt.args...)
			if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
				t.Errorf("%s rejected %s: %v\n%s", tt.service, tt.file, err, out)
			}
		})
	}
}

func TestShortenerImage(t *testing.T) {
	requireDocker(t)

	const port, tag = "20080", "just-short-it-test-shortener"
	build := exec.Command("docker", "build", "--quiet", "--file", "services/shortener/deploy/Dockerfile", "--build-arg", "SHORTENER_PORT="+port, "--tag", tag, ".")
	build.Dir = root
	run(t, build)
	t.Cleanup(func() {
		exec.Command("docker", "image", "rm", tag).Run()
	})

	var image struct {
		ExposedPorts map[string]struct{}
		Env          []string
	}
	if err := json.Unmarshal(run(t, exec.Command("docker", "image", "inspect", "--format", "{{json .Config}}", tag)), &image); err != nil {
		t.Fatal(err)
	}
	if _, ok := image.ExposedPorts[port+"/tcp"]; !ok {
		t.Errorf("image exposes %v, want %s/tcp", slices.Collect(maps.Keys(image.ExposedPorts)), port)
	}
	if !slices.Contains(image.Env, "SHORTENER_PORT="+port) {
		t.Errorf("image environment = %v, want SHORTENER_PORT=%s", image.Env, port)
	}
}
