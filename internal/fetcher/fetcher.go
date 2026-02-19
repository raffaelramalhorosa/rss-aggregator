package fetcher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/yourusername/rss-aggregator/internal/models"
	"github.com/yourusername/rss-aggregator/internal/store"
)

// Fetcher periodically pulls every registered feed using concurrent workers
// and pushes parsed articles into the store.
type Fetcher struct {
	store    *store.Store
	parser   *gofeed.Parser
	interval time.Duration
	logger   *slog.Logger
}

// New returns a Fetcher that polls feeds every interval.
func New(s *store.Store, interval time.Duration, logger *slog.Logger) *Fetcher {
	return &Fetcher{
		store:    s,
		parser:   gofeed.NewParser(),
		interval: interval,
		logger:   logger,
	}
}

// Start begins the background polling loop. It blocks until ctx is cancelled.
func (f *Fetcher) Start(ctx context.Context) {
	f.logger.Info("fetcher started", "interval", f.interval)

	// Run immediately on startup, then on every tick.
	f.fetchAll(ctx)

	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("fetcher stopped")
			return
		case <-ticker.C:
			f.fetchAll(ctx)
		}
	}
}

// fetchAll fans-out one goroutine per feed, collects results through a channel,
// and persists them. This is the core concurrency pattern.
func (f *Fetcher) fetchAll(ctx context.Context) {
	feeds := f.store.ListFeeds()
	if len(feeds) == 0 {
		return
	}

	f.logger.Info("fetch cycle starting", "feeds", len(feeds))

	results := make(chan models.FetchResult, len(feeds))

	var wg sync.WaitGroup
	for _, feed := range feeds {
		wg.Add(1)
		go func(feed models.Feed) {
			defer wg.Done()
			articles, err := f.fetchFeed(ctx, feed)
			results <- models.FetchResult{
				FeedID:   feed.ID,
				Articles: articles,
				Err:      err,
			}
		}(feed)
	}

	// Close the channel once every goroutine finishes.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and persist results as they arrive.
	var totalSaved int
	for res := range results {
		if res.Err != nil {
			f.logger.Error("feed fetch failed", "feed_id", res.FeedID, "error", res.Err)
			continue
		}
		saved := f.store.SaveArticles(res.Articles)
		f.store.UpdateLastFetched(res.FeedID, time.Now())
		totalSaved += saved
		f.logger.Info("feed fetched",
			"feed_id", res.FeedID,
			"articles", len(res.Articles),
			"new", saved,
		)
	}

	f.logger.Info("fetch cycle complete", "new_articles", totalSaved)
}

// fetchFeed downloads and parses a single feed, returning article models.
func (f *Fetcher) fetchFeed(ctx context.Context, feed models.Feed) ([]models.Article, error) {
	parsedCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	parsed, err := f.parser.ParseURLWithContext(feed.URL, parsedCtx)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", feed.URL, err)
	}

	articles := make([]models.Article, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		pub := time.Now()
		if item.PublishedParsed != nil {
			pub = *item.PublishedParsed
		}

		articles = append(articles, models.Article{
			ID:          generateID(feed.ID, item.Link),
			FeedID:      feed.ID,
			FeedName:    feed.Name,
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			PublishedAt: pub,
		})
	}
	return articles, nil
}

// generateID creates a deterministic ID so re-fetching the same article
// does not create duplicates.
func generateID(feedID, link string) string {
	h := sha256.Sum256([]byte(feedID + "|" + link))
	return fmt.Sprintf("%x", h[:8])
}
