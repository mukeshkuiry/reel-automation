package caption

import (
	"fmt"
	"strings"

	"github.com/mukeshkuiry/reel-automation/internal/reddit"
)

func Generate(post reddit.Post) string {
	base := strings.TrimSpace(post.Title)
	if base == "" {
		base = "Trending video from Reddit."
	}
	return fmt.Sprintf("%s\n\nFollow for more viral internet videos.\n\n#reddit #viral #%s", base, strings.ToLower(post.Subreddit))
}
