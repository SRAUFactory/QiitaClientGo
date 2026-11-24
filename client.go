package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://qiita.com/api/v2"
	DefaultTimeout = 30 * time.Second
)

// QiitaTag represents a tag attached to a Qiita article.
type QiitaTag struct {
	Name string `json:"name"`
}

// TagNames converts a slice of QiitaTag into a slice of tag name strings.
func TagNames(tags []QiitaTag) []string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names
}

// ToQiitaTags converts a slice of tag name strings into a slice of QiitaTag.
func ToQiitaTags(names []string) []QiitaTag {
	tags := make([]QiitaTag, len(names))
	for i, name := range names {
		tags[i] = QiitaTag{Name: name}
	}
	return tags
}

// QiitaItem represents a Qiita article retrieved from the API.
type QiitaItem struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Private   bool       `json:"private"`
	Tags      []QiitaTag `json:"tags"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	URL       string     `json:"url"`
}

// QiitaItemRequest represents the request payload for creating or updating an article.
type QiitaItemRequest struct {
	Title   string     `json:"title"`
	Body    string     `json:"body"`
	Private bool       `json:"private"`
	Tags    []QiitaTag `json:"tags"`
}

// QiitaClient provides methods to interact with Qiita API v2.
type QiitaClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewQiitaClient creates a new QiitaClient instance.
func NewQiitaClient(token string, baseURL string, timeout time.Duration) *QiitaClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &QiitaClient{
		token:   token,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// do executes an HTTP request, handles headers, status checking, and JSON encoding/decoding.
func (c *QiitaClient) do(method, path string, in, out any) error {
	var bodyReader io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("failed to marshal request body for %s %s: %w", method, path, err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request for %s %s: %w", method, path, err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request for %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body for %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request %s %s failed with status %d: %s", method, path, resp.StatusCode, string(bodyBytes))
	}

	if out != nil {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return fmt.Errorf("failed to parse JSON response for %s %s: %w", method, path, err)
		}
	}

	return nil
}

// FetchAllAuthenticatedUserItems retrieves all articles of the authenticated user with pagination.
func (c *QiitaClient) FetchAllAuthenticatedUserItems() ([]QiitaItem, error) {
	var allItems []QiitaItem
	page := 1
	const perPage = 100

	for {
		path := fmt.Sprintf("/authenticated_user/items?page=%d&per_page=%d", page, perPage)
		var items []QiitaItem
		if err := c.do(http.MethodGet, path, nil, &items); err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		if len(items) == 0 {
			break
		}

		allItems = append(allItems, items...)

		if len(items) < perPage {
			break
		}

		page++
	}

	return allItems, nil
}

// CreateItem creates a new Qiita article (POST /items).
func (c *QiitaClient) CreateItem(reqData QiitaItemRequest) (*QiitaItem, error) {
	var item QiitaItem
	if err := c.do(http.MethodPost, "/items", reqData, &item); err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}
	return &item, nil
}

// UpdateItem updates an existing Qiita article (PATCH /items/{id}).
func (c *QiitaClient) UpdateItem(id string, reqData QiitaItemRequest) (*QiitaItem, error) {
	path := fmt.Sprintf("/items/%s", id)
	var item QiitaItem
	if err := c.do(http.MethodPatch, path, reqData, &item); err != nil {
		return nil, fmt.Errorf("failed to update item %s: %w", id, err)
	}
	return &item, nil
}

func (c *QiitaClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}
