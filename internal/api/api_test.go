package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/raffaelramalhorosa/rss-aggregator/internal/api"
	"github.com/raffaelramalhorosa/rss-aggregator/internal/models"
	"github.com/raffaelramalhorosa/rss-aggregator/internal/store"
)

func setup() (*api.Server, *store.Store) {
	s := store.New()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return api.New(s, logger), s
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := setup()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAddFeedEndpoint(t *testing.T) {
	srv, _ := setup()

	body, _ := json.Marshal(models.AddFeedRequest{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom"})
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var feed models.Feed
	json.NewDecoder(rec.Body).Decode(&feed)

	if feed.Name != "Go Blog" {
		t.Fatalf("expected name 'Go Blog', got '%s'", feed.Name)
	}
}

func TestAddFeedValidation(t *testing.T) {
	srv, _ := setup()

	body, _ := json.Marshal(models.AddFeedRequest{Name: "", URL: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/feeds", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty fields, got %d", rec.Code)
	}
}

func TestListFeedsEndpoint(t *testing.T) {
	srv, s := setup()
	s.AddFeed("Feed 1", "https://example.com/1")
	s.AddFeed("Feed 2", "https://example.com/2")

	req := httptest.NewRequest(http.MethodGet, "/api/feeds", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	var feeds []models.Feed
	json.NewDecoder(rec.Body).Decode(&feeds)

	if len(feeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(feeds))
	}
}

func TestRemoveFeedEndpoint(t *testing.T) {
	srv, s := setup()
	f := s.AddFeed("To Remove", "https://example.com/rss")

	req := httptest.NewRequest(http.MethodDelete, "/api/feeds/"+f.ID, nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Removing again should 404.
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/feeds/"+f.ID, nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", rec.Code)
	}
}

func TestListArticlesEndpoint(t *testing.T) {
	srv, s := setup()
	s.SaveArticles([]models.Article{
		{ID: "a1", FeedID: "f1", Title: "Article 1"},
		{ID: "a2", FeedID: "f1", Title: "Article 2"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/articles?feed_id=f1&limit=1", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	var articles []models.Article
	json.NewDecoder(rec.Body).Decode(&articles)

	if len(articles) != 1 {
		t.Fatalf("expected 1 article with limit=1, got %d", len(articles))
	}
}
