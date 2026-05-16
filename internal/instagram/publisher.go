package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Publisher interface {
	Publish(ctx context.Context, videoURL string, caption string) (string, error)
}

type GraphPublisher struct {
	httpClient     *http.Client
	businessID     string
	accessToken    string
	graphBase      string
	pollInterval   time.Duration
	maxPollRetries int
}

func NewGraphPublisher(httpClient *http.Client, businessID, accessToken string) *GraphPublisher {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &GraphPublisher{
		httpClient:     httpClient,
		businessID:     businessID,
		accessToken:    accessToken,
		graphBase:      "https://graph.facebook.com/v20.0",
		pollInterval:   5 * time.Second,
		maxPollRetries: 12,
	}
}

func (p *GraphPublisher) Publish(ctx context.Context, videoURL string, caption string) (string, error) {
	if p.businessID == "" || p.accessToken == "" {
		return "", fmt.Errorf("instagram credentials are not configured")
	}
	containerID, err := p.createMediaContainer(ctx, videoURL, caption)
	if err != nil {
		return "", err
	}
	return p.publishContainer(ctx, containerID)
}

func (p *GraphPublisher) createMediaContainer(ctx context.Context, videoURL string, caption string) (string, error) {
	values := url.Values{}
	values.Set("media_type", "REELS")
	values.Set("video_url", videoURL)
	values.Set("caption", caption)
	values.Set("access_token", p.accessToken)

	endpoint := fmt.Sprintf("%s/%s/media", p.graphBase, p.businessID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("create media request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("media create failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("media create returned status %d", resp.StatusCode)
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode media create response: %w", err)
	}
	if payload.ID == "" {
		return "", fmt.Errorf("instagram media container id missing")
	}
	return payload.ID, nil
}

func (p *GraphPublisher) publishContainer(ctx context.Context, creationID string) (string, error) {
	values := url.Values{}
	values.Set("creation_id", creationID)
	values.Set("access_token", p.accessToken)

	endpoint := fmt.Sprintf("%s/%s/media_publish", p.graphBase, p.businessID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return "", fmt.Errorf("create publish request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("media publish failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("media publish returned status %d", resp.StatusCode)
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode media publish response: %w", err)
	}
	if payload.ID == "" {
		return "", fmt.Errorf("instagram media id missing")
	}
	return payload.ID, nil
}
