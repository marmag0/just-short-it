package link_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marmag0/just-short-it/services/shortener/link"
)

const target = "https://example.com/target"

func serve(suffix string) *httptest.ResponseRecorder {
	srv := &link.Server{
		Base:  "short.test",
		Links: map[string]string{"known": target},
	}

	req := httptest.NewRequest(http.MethodGet, "/"+suffix, nil)
	req.SetPathValue("link", suffix)
	rec := httptest.NewRecorder()
	srv.HandleLink(rec, req)
	return rec
}

func TestHandleLinkRedirect(t *testing.T) {
	rec := serve("known")

	// Permanent redirects get cached by clients, so later clicks would skip telemetry.
	if rec.Code != http.StatusFound && rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want a temporary redirect (302 or 307)", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != target {
		t.Errorf("Location = %q, want %q", got, target)
	}
}

func TestHandleLinkNotFound(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
	}{
		{"unknown link", "unknown"},
		{"too short", "ab"},
		{"too long", strings.Repeat("a", 33)},
		{"disallowed character", "kno-wn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(tt.suffix)
			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
			if got := rec.Header().Get("Location"); got != "" {
				t.Errorf("Location = %q, want none", got)
			}
		})
	}
}
