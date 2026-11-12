package link

import (
	"log/slog"
	"net/http"
)

type Server struct {
	Base  string
	Links map[string]string
	// further args in future
}

func (s *Server) HandleLink(w http.ResponseWriter, r *http.Request) {
	link := r.PathValue("link")
	fullLink := s.Base + "/" + link

	slog.Info("received short link request", "link", fullLink)

	// 1. Initial short link format validation
	ok, msg := InitialLinkValid(link)
	if !ok {
		slog.Error("short link not allowed", "link", fullLink, "msg", msg)
		http.NotFound(w, r)
		return
	}

	// DEMO: checking if short link exists (in mock database)
	var target string = s.Links[link]
	if target == "" {
		slog.Error("short link doesn't exist", "link", fullLink)
		http.NotFound(w, r)
		return
	}

	// 2. Checking if short link existis in Redis cache
	// ...

	// 3. Checking if short link exists in PostgreSQL database
	// ...

	// 4. Updating Redis cache entry
	// ...

	// 5. Saving telemetry data
	// ...

	http.Redirect(w, r, target, http.StatusFound)
	slog.Info("redirect successful", "link", fullLink, "target", target)
}
