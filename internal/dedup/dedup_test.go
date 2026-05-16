package dedup

import (
	"context"
	"testing"

	"github.com/mukeshkuiry/reel-automation/internal/storage"
)

func TestAlreadyPostedReflectsStoredRecord(t *testing.T) {
	ctx := context.Background()
	store := storage.NewInMemoryStore()
	svc := New(store)

	posted, err := svc.AlreadyPosted(ctx, "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if posted {
		t.Fatal("expected post to be unseen")
	}

	if err := store.Save(ctx, storage.UploadRecord{RedditPostID: "abc123"}); err != nil {
		t.Fatalf("save record: %v", err)
	}
	posted, err = svc.AlreadyPosted(ctx, "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !posted {
		t.Fatal("expected post to be seen")
	}
}
