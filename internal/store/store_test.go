package store_test

import (
	"testing"
	"time"

	"github.com/raffaelramalhorosa/rss-aggregator/internal/models"
	"github.com/raffaelramalhorosa/rss-aggregator/internal/store"
)

func TestAddAndListFeeds(t *testing.T) {
	s := store.New()

	f1 := s.AddFeed("Go Blog", "https://go.dev/blog/feed.atom")
	f2 := s.AddFeed("Lobsters", "https://lobste.rs/rss")

	feeds := s.ListFeeds()
	if len(feeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(feeds))
	}

	if f1.ID == f2.ID {
		t.Fatal("feed IDs should be unique")
	}
}

func TestRemoveFeed(t *testing.T) {
	s := store.New()
	f := s.AddFeed("Test", "https://example.com/rss")

	if !s.RemoveFeed(f.ID) {
		t.Fatal("expected removal to succeed")
	}

	if s.RemoveFeed(f.ID) {
		t.Fatal("expected removal of non-existent feed to return false")
	}

	if len(s.ListFeeds()) != 0 {
		t.Fatal("expected no feeds after removal")
	}
}

func TestRemoveFeedCascadesArticles(t *testing.T) {
	s := store.New()
	f := s.AddFeed("Test", "https://example.com/rss")

	articles := []models.Article{
		{ID: "a1", FeedID: f.ID, Title: "Post 1"},
		{ID: "a2", FeedID: f.ID, Title: "Post 2"},
	}
	s.SaveArticles(articles)

	s.RemoveFeed(f.ID)

	if len(s.ListArticles("", 0)) != 0 {
		t.Fatal("expected articles to be removed with feed")
	}
}

func TestSaveArticlesDeduplication(t *testing.T) {
	s := store.New()

	articles := []models.Article{
		{ID: "a1", Title: "Post 1"},
		{ID: "a2", Title: "Post 2"},
	}

	saved := s.SaveArticles(articles)
	if saved != 2 {
		t.Fatalf("expected 2 saved, got %d", saved)
	}

	// Save again — duplicates should be skipped.
	saved = s.SaveArticles(articles)
	if saved != 0 {
		t.Fatalf("expected 0 saved on duplicate insert, got %d", saved)
	}
}

func TestListArticlesSortedAndFiltered(t *testing.T) {
	s := store.New()

	now := time.Now()
	articles := []models.Article{
		{ID: "old", FeedID: "f1", Title: "Old", PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "new", FeedID: "f1", Title: "New", PublishedAt: now},
		{ID: "mid", FeedID: "f2", Title: "Mid", PublishedAt: now.Add(-1 * time.Hour)},
	}
	s.SaveArticles(articles)

	// All articles, sorted newest first.
	all := s.ListArticles("", 0)
	if all[0].ID != "new" || all[1].ID != "mid" || all[2].ID != "old" {
		t.Fatal("articles not sorted by published_at descending")
	}

	// Filtered by feed.
	filtered := s.ListArticles("f2", 0)
	if len(filtered) != 1 || filtered[0].ID != "mid" {
		t.Fatal("feed filter not working")
	}

	// Limited.
	limited := s.ListArticles("", 1)
	if len(limited) != 1 {
		t.Fatalf("expected 1 article with limit, got %d", len(limited))
	}
}
