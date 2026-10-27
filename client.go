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

// FetchAllAuthenticatedUserItems retrieves all articles of the authenticated user with pagination.
func (c *QiitaClient) FetchAllAuthenticatedUserItems() ([]QiitaItem, error) {
	var allItems []QiitaItem
	page := 1

	for {
		url := fmt.Sprintf("%s/authenticated_user/items?page=%d&per_page=100", c.baseURL, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for page %d: %w", page, err)
		}
		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for page %d: %w", page, err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("Qiita API returned status %d on page %d: %s", resp.StatusCode, page, string(bodyBytes))
		}

		var items []QiitaItem
		if err := json.Unmarshal(bodyBytes, &items); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response for page %d: %w", page, err)
		}

		if len(items) == 0 {
			break
		}

		allItems = append(allItems, items...)
		page++
	}

	return allItems, nil
}

// CreateItem creates a new Qiita article (POST /items).
func (c *QiitaClient) CreateItem(reqData QiitaItemRequest) (*QiitaItem, error) {
	payload, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	url := fmt.Sprintf("%s/items", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read POST response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to create item (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var item QiitaItem
	if err := json.Unmarshal(bodyBytes, &item); err != nil {
		return nil, fmt.Errorf("failed to parse created item JSON: %w", err)
	}

	return &item, nil
}

// UpdateItem updates an existing Qiita article (PATCH /items/{id}).
func (c *QiitaClient) UpdateItem(id string, reqData QiitaItemRequest) (*QiitaItem, error) {
	payload, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	url := fmt.Sprintf("%s/items/%s", c.baseURL, id)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create PATCH request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PATCH request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PATCH response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to update item (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var item QiitaItem
	if err := json.Unmarshal(bodyBytes, &item); err != nil {
		return nil, fmt.Errorf("failed to parse updated item JSON: %w", err)
	}

	return &item, nil
}

func (c *QiitaClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}
