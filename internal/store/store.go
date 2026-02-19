package store

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/raffaelramalhorosa/rss-aggregator/internal/models"
)

// Store provides thread-safe, in-memory storage for feeds and articles.
// All public methods are safe for concurrent use.
type Store struct {
	mu       sync.RWMutex
	feeds    map[string]models.Feed
	articles map[string]models.Article // keyed by article ID
}

// New creates an empty Store ready for use.
func New() *Store {
	return &Store{
		feeds:    make(map[string]models.Feed),
		articles: make(map[string]models.Article),
	}
}

// ---------- Feeds ----------

// AddFeed registers a new feed and returns its generated ID.
func (s *Store) AddFeed(name, url string) models.Feed {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("feed_%d", time.Now().UnixNano())
	feed := models.Feed{
		ID:   id,
		Name: name,
		URL:  url,
	}
	s.feeds[id] = feed
	return feed
}

// RemoveFeed deletes a feed and all of its articles.
func (s *Store) RemoveFeed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.feeds[id]; !ok {
		return false
	}

	delete(s.feeds, id)

	for key, art := range s.articles {
		if art.FeedID == id {
			delete(s.articles, key)
		}
	}
	return true
}

// ListFeeds returns every registered feed.
func (s *Store) ListFeeds() []models.Feed {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feeds := make([]models.Feed, 0, len(s.feeds))
	for _, f := range s.feeds {
		feeds = append(feeds, f)
	}
	return feeds
}

// UpdateLastFetched records when a feed was last successfully fetched.
func (s *Store) UpdateLastFetched(feedID string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if f, ok := s.feeds[feedID]; ok {
		f.LastFetched = t
		s.feeds[feedID] = f
	}
}

// ---------- Articles ----------

// SaveArticles persists a batch of articles, skipping duplicates by link.
func (s *Store) SaveArticles(articles []models.Article) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	saved := 0
	for _, a := range articles {
		if _, exists := s.articles[a.ID]; !exists {
			s.articles[a.ID] = a
			saved++
		}
	}
	return saved
}

// ListArticles returns articles sorted newest-first.
// If feedID is non-empty only articles from that feed are returned.
// limit <= 0 means no limit.
func (s *Store) ListArticles(feedID string, limit int) []models.Article {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Article, 0, len(s.articles))
	for _, a := range s.articles {
		if feedID != "" && a.FeedID != feedID {
			continue
		}
		result = append(result, a)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].PublishedAt.After(result[j].PublishedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}
