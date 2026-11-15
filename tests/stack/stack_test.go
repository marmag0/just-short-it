package stack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The demo link hardcoded in main.go.
const demoLink = "test"

// freePort returns a host port nothing listens on, so the stack does not collide with a running one.
func freePort(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
}

// eventually retries check every second until it succeeds or a minute passes.
func eventually(t *testing.T, what string, check func() error) {
	t.Helper()

	deadline := time.Now().Add(time.Minute)
	for {
		err := check()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: %v", what, err)
		}
		time.Sleep(time.Second)
	}
}

// body returns the body of a 200 OK response and an error for anything else.
func body(resp *http.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status = %d: %s", resp.StatusCode, b)
	}
	return b, nil
}

func TestStack(t *testing.T) {
	if testing.Short() {
		t.Skip("starting the whole stack is slow")
	}
	requireDocker(t)

	env := exampleEnv(t)
	for _, key := range portKeys(env) {
		env[key] = freePort(t)
	}
	env["GRAFANA_ADMIN_PASSWORD"] = "stack-test-password"
	const project = "just-short-it-test"

	t.Cleanup(func() {
		compose(t, env, "-p", project, "down", "--volumes", "--rmi", "local").Run()
	})
	run(t, compose(t, env, "-p", project, "up", "--detach", "--build"))

	local := func(key string) string {
		return "http://127.0.0.1:" + env[key]
	}
	grafana := func(path string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodGet, local("GRAFANA_PORT")+path, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(env["GRAFANA_ADMIN_USER"], env["GRAFANA_ADMIN_PASSWORD"])
		return http.DefaultClient.Do(req)
	}

	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	eventually(t, "shortener redirect", func() error {
		resp, err := noRedirect.Get(local("SHORTENER_PORT") + "/" + demoLink)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusFound {
			return fmt.Errorf("status = %d, want %d", resp.StatusCode, http.StatusFound)
		}
		return nil
	})

	eventually(t, "Grafana health", func() error {
		_, err := body(grafana("/api/health"))
		return err
	})

	now := time.Now().UnixNano()
	traceID := fmt.Sprintf("%032x", now)
	trace := fmt.Sprintf(`{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"stack-test"}}]},`+
		`"scopeSpans":[{"spans":[{"traceId":%q,"spanId":"0000000000000001","name":"stack-test","kind":2,"startTimeUnixNano":"%d","endTimeUnixNano":"%d"}]}]}]}`,
		traceID, now, now)
	eventually(t, "OTLP/HTTP export to Alloy", func() error {
		_, err := body(http.Post(local("ALLOY_OTLP_HTTP_PORT")+"/v1/traces", "application/json", strings.NewReader(trace)))
		return err
	})

	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	h2c := &http.Client{Transport: &http.Transport{Protocols: protocols}}
	eventually(t, "OTLP/gRPC export to Alloy", func() error {
		// A zero-length gRPC message is a valid, empty ExportTraceServiceRequest.
		req, err := http.NewRequest(http.MethodPost, local("ALLOY_OTLP_GRPC_PORT")+"/opentelemetry.proto.collector.trace.v1.TraceService/Export", bytes.NewReader(make([]byte, 5)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/grpc")
		resp, err := h2c.Do(req)
		if _, err := body(resp, err); err != nil {
			return err
		}
		if status := resp.Trailer.Get("Grpc-Status"); status != "0" {
			return fmt.Errorf("grpc-status = %q, want 0", status)
		}
		return nil
	})

	eventually(t, "trace in Tempo", func() error {
		_, err := body(grafana("/api/datasources/proxy/uid/tempo/api/v2/traces/" + traceID))
		return err
	})

	query := url.Values{"query": {fmt.Sprintf(`{service_name="shortener", project=%q}`, project)}}
	eventually(t, "shortener logs in Loki", func() error {
		b, err := body(grafana("/api/datasources/proxy/uid/loki/loki/api/v1/query_range?" + query.Encode()))
		if err != nil {
			return err
		}
		var r struct {
			Data struct {
				Result []json.RawMessage
			}
		}
		if err := json.Unmarshal(b, &r); err != nil {
			return err
		}
		if len(r.Data.Result) == 0 {
			return errors.New("no shortener log streams yet")
		}
		return nil
	})
}
