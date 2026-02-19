package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/yourusername/rss-aggregator/internal/models"
	"github.com/yourusername/rss-aggregator/internal/store"
)

// Server holds dependencies for the HTTP handlers.
type Server struct {
	store  *store.Store
	logger *slog.Logger
	mux    *http.ServeMux
}

// New wires up routes and returns a ready-to-use Server.
func New(s *store.Store, logger *slog.Logger) *Server {
	srv := &Server{store: s, logger: logger, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

// ServeHTTP makes Server satisfy the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ---------- Routes ----------

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	s.mux.HandleFunc("GET /api/feeds", s.handleListFeeds)
	s.mux.HandleFunc("POST /api/feeds", s.handleAddFeed)
	s.mux.HandleFunc("DELETE /api/feeds/{id}", s.handleRemoveFeed)

	s.mux.HandleFunc("GET /api/articles", s.handleListArticles)
}

// ---------- Handlers ----------

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListFeeds(w http.ResponseWriter, _ *http.Request) {
	feeds := s.store.ListFeeds()
	writeJSON(w, http.StatusOK, feeds)
}

func (s *Server) handleAddFeed(w http.ResponseWriter, r *http.Request) {
	var req models.AddFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if req.Name == "" || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and url are required"})
		return
	}

	feed := s.store.AddFeed(req.Name, req.URL)
	s.logger.Info("feed added", "id", feed.ID, "name", feed.Name)
	writeJSON(w, http.StatusCreated, feed)
}

func (s *Server) handleRemoveFeed(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.store.RemoveFeed(id) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "feed not found"})
		return
	}
	s.logger.Info("feed removed", "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "feed removed"})
}

func (s *Server) handleListArticles(w http.ResponseWriter, r *http.Request) {
	feedID := r.URL.Query().Get("feed_id")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	articles := s.store.ListArticles(feedID, limit)
	writeJSON(w, http.StatusOK, articles)
}

// ---------- Helpers ----------

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
