package models

import "time"

// Feed represents an RSS/Atom feed source to be monitored.
type Feed struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	LastFetched time.Time `json:"last_fetched"`
}

// Article represents a single item parsed from a feed.
type Article struct {
	ID          string    `json:"id"`
	FeedID      string    `json:"feed_id"`
	FeedName    string    `json:"feed_name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"published_at"`
}

// AddFeedRequest is the payload for registering a new feed.
type AddFeedRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FetchResult carries the outcome of a single feed fetch through a channel.
type FetchResult struct {
	FeedID   string
	Articles []Article
	Err      error
}
