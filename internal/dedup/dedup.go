package dedup

import (
	"context"

	"github.com/mukeshkuiry/reel-automation/internal/storage"
)

type Service struct {
	store storage.Store
}

func New(store storage.Store) *Service {
	return &Service{store: store}
}

func (s *Service) AlreadyPosted(ctx context.Context, redditPostID string) (bool, error) {
	return s.store.IsUploaded(ctx, redditPostID)
}
