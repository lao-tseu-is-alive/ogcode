package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SearchResult is one entry returned by the /search endpoint.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// PageContent is the extracted text returned by the /fetch endpoint.
type PageContent struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
}

// BridgeClient speaks to the Node.js Playwright search bridge.
type BridgeClient struct {
	baseURL string
	http    *http.Client
}

// NewBridgeClient creates a client pointed at the given base URL (e.g. "http://127.0.0.1:7331").
func NewBridgeClient(baseURL string) *BridgeClient {
	return &BridgeClient{
		baseURL: baseURL,
		// Generous timeout: the bridge caps concurrency, so a burst of parallel
		// fetches queues server-side while this client's clock runs. 120s leaves
		// room for several queued rounds of slow (up to 25s) page loads.
		http: &http.Client{Timeout: 120 * time.Second},
	}
}

// Health checks whether the bridge is alive.
func (b *BridgeClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bridge health: status %d", resp.StatusCode)
	}
	return nil
}

// Search queries Google via the bridge and returns up to limit results.
func (b *BridgeClient) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 8
	}
	body, _ := json.Marshal(map[string]any{"query": query, "limit": limit})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bridge search: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bridge search: status %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Results []SearchResult `json:"results"`
		Error   string         `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("bridge search decode: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("bridge search: %s", result.Error)
	}
	return result.Results, nil
}

// FetchPage fetches the text content of a URL via the bridge.
func (b *BridgeClient) FetchPage(ctx context.Context, url string) (PageContent, error) {
	body, _ := json.Marshal(map[string]string{"url": url})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/fetch", bytes.NewReader(body))
	if err != nil {
		return PageContent{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		return PageContent{}, fmt.Errorf("bridge fetch: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return PageContent{}, fmt.Errorf("bridge fetch: status %d: %s", resp.StatusCode, raw)
	}

	var content PageContent
	if err := json.Unmarshal(raw, &content); err != nil {
		return PageContent{}, fmt.Errorf("bridge fetch decode: %w", err)
	}
	return content, nil
}
