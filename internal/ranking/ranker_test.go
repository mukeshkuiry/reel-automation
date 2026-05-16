package ranking

import (
	"testing"
	"time"

	"github.com/mukeshkuiry/reel-automation/internal/reddit"
)

func TestRankSortsByCompositeScore(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	posts := []reddit.Post{
		{
			ID:          "slow",
			Ups:         100,
			NumComments: 5,
			CreatedUTC:  float64(now.Add(-100 * time.Minute).Unix()),
		},
		{
			ID:          "fast",
			Ups:         100,
			NumComments: 30,
			CreatedUTC:  float64(now.Add(-10 * time.Minute).Unix()),
		},
	}

	ranked := Rank(posts, now)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked posts, got %d", len(ranked))
	}
	if ranked[0].Post.ID != "fast" {
		t.Fatalf("expected highest scored post first, got %q", ranked[0].Post.ID)
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Fatalf("expected descending scores, got %f <= %f", ranked[0].Score, ranked[1].Score)
	}
}
