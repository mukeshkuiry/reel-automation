package ranking

import (
	"sort"
	"time"

	"github.com/mukeshkuiry/reel-automation/internal/reddit"
)

type RankedPost struct {
	Post  reddit.Post
	Score float64
}

func Rank(posts []reddit.Post, now time.Time) []RankedPost {
	ranked := make([]RankedPost, 0, len(posts))
	for _, post := range posts {
		ageMinutes := reddit.AgeInMinutes(post.CreatedUTC, now)
		velocity := float64(post.Ups) / ageMinutes
		score := velocity*0.7 + float64(post.NumComments)*0.3
		ranked = append(ranked, RankedPost{Post: post, Score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}
