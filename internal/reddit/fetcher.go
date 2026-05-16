package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Post struct {
	ID          string
	Title       string
	Subreddit   string
	Ups         int
	NumComments int
	CreatedUTC  float64
	VideoURL    string
	Permalink   string
}

type Fetcher interface {
	FetchTrending(ctx context.Context, subreddit string, limit int) ([]Post, error)
}

type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	username     string
	password     string
	userAgent    string

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewClient(httpClient *http.Client, clientID, clientSecret, username, password, userAgent string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		httpClient:   httpClient,
		clientID:     clientID,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
		userAgent:    userAgent,
	}
}

func (c *Client) FetchTrending(ctx context.Context, subreddit string, limit int) ([]Post, error) {
	if limit <= 0 {
		limit = 25
	}
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("https://oauth.reddit.com/r/%s/top?t=hour&limit=%d", url.PathEscape(subreddit), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reddit top request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("reddit top returned status %d", resp.StatusCode)
	}

	var payload listingResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode reddit response: %w", err)
	}

	posts := make([]Post, 0, len(payload.Data.Children))
	now := time.Now().UTC()
	for _, child := range payload.Data.Children {
		d := child.Data
		if !d.IsVideo || d.Over18 || d.Locked || d.RemovedByCategory != "" {
			continue
		}
		postAge := now.Sub(time.Unix(int64(d.CreatedUTC), 0))
		if postAge > 2*time.Hour {
			continue
		}
		videoURL := d.URLOverriddenByDest
		if videoURL == "" && d.Media.RedditVideo.FallbackURL != "" {
			videoURL = d.Media.RedditVideo.FallbackURL
		}
		if videoURL == "" {
			continue
		}
		posts = append(posts, Post{
			ID:          d.ID,
			Title:       d.Title,
			Subreddit:   d.Subreddit,
			Ups:         d.Ups,
			NumComments: d.NumComments,
			CreatedUTC:  d.CreatedUTC,
			VideoURL:    videoURL,
			Permalink:   "https://reddit.com" + d.Permalink,
		})
	}
	return posts, nil
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	if c.clientID == "" || c.clientSecret == "" || c.username == "" || c.password == "" {
		return "", errors.New("reddit oauth credentials are not configured")
	}

	form := url.Values{
		"grant_type": {"password"},
		"username":   {c.username},
		"password":   {c.password},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.reddit.com/api/v1/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("reddit oauth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("reddit oauth returned status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode oauth response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", errors.New("missing oauth access token")
	}

	c.mu.Lock()
	c.token = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.mu.Unlock()
	return tokenResp.AccessToken, nil
}

type listingResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID                  string  `json:"id"`
				Title               string  `json:"title"`
				Subreddit           string  `json:"subreddit"`
				Ups                 int     `json:"ups"`
				NumComments         int     `json:"num_comments"`
				CreatedUTC          float64 `json:"created_utc"`
				IsVideo             bool    `json:"is_video"`
				Over18              bool    `json:"over_18"`
				Locked              bool    `json:"locked"`
				Permalink           string  `json:"permalink"`
				RemovedByCategory   string  `json:"removed_by_category"`
				URLOverriddenByDest string  `json:"url_overridden_by_dest"`
				Media               struct {
					RedditVideo struct {
						FallbackURL string `json:"fallback_url"`
					} `json:"reddit_video"`
				} `json:"media"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func AgeInMinutes(createdUTC float64, now time.Time) float64 {
	age := now.Sub(time.Unix(int64(createdUTC), 0)).Minutes()
	if age <= 0 {
		return 1
	}
	return age
}

func ParseCreatedUTC(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse created_utc: %w", err)
	}
	return parsed, nil
}
