package storage

import (
	"context"
	"sync"
	"time"
)

type UploadRecord struct {
	RedditPostID     string
	Subreddit        string
	Title            string
	Permalink        string
	Score            float64
	VideoURL         string
	LocalVideoPath   string
	InstagramMediaID string
	UploadStatus     string
	CreatedAt        time.Time
	UploadedAt       *time.Time
}

type Store interface {
	IsUploaded(ctx context.Context, redditPostID string) (bool, error)
	Save(ctx context.Context, record UploadRecord) error
}

type InMemoryStore struct {
	mu      sync.RWMutex
	records map[string]UploadRecord
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{records: make(map[string]UploadRecord)}
}

func (s *InMemoryStore) IsUploaded(_ context.Context, redditPostID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.records[redditPostID]
	return ok, nil
}

func (s *InMemoryStore) Save(_ context.Context, record UploadRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.RedditPostID] = record
	return nil
}
