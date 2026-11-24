package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQiitaClient_FetchAllAuthenticatedUserItems(t *testing.T) {
	pageCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		page := r.URL.Query().Get("page")
		pageCalls++

		if page == "1" {
			// Exactly 100 items on page 1
			items := make([]QiitaItem, 100)
			for i := 0; i < 100; i++ {
				items[i] = QiitaItem{ID: fmt.Sprintf("item-%d", i+1)}
			}
			_ = json.NewEncoder(w).Encode(items)
			return
		}

		if page == "2" {
			// 1 item on page 2 (< 100, terminates pagination)
			items := []QiitaItem{
				{ID: "item-101", Title: "Article 101"},
			}
			_ = json.NewEncoder(w).Encode(items)
			return
		}

		// Page 3: should not be reached
		_ = json.NewEncoder(w).Encode([]QiitaItem{})
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	items, err := client.FetchAllAuthenticatedUserItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 101 {
		t.Fatalf("expected 101 items, got %d", len(items))
	}
	if items[0].ID != "item-1" || items[100].ID != "item-101" {
		t.Errorf("unexpected items content")
	}
	if pageCalls != 2 {
		t.Errorf("expected 2 page requests, got %d", pageCalls)
	}
}

func TestQiitaClient_FetchAllPaginationBoundary(t *testing.T) {
	pageCalls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pageCalls++

		if page == "1" {
			// Exactly 100 items
			items := make([]QiitaItem, 100)
			for i := 0; i < 100; i++ {
				items[i] = QiitaItem{ID: fmt.Sprintf("item-%d", i)}
			}
			_ = json.NewEncoder(w).Encode(items)
			return
		}

		if page == "2" {
			// Empty page to terminate
			_ = json.NewEncoder(w).Encode([]QiitaItem{})
			return
		}

		t.Errorf("unexpected page %s requested", page)
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	items, err := client.FetchAllAuthenticatedUserItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 100 {
		t.Fatalf("expected 100 items, got %d", len(items))
	}
	if pageCalls != 2 {
		t.Errorf("expected 2 page requests, got %d", pageCalls)
	}
}

func TestQiitaClient_CreateItem(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/items" {
			t.Errorf("expected path /items, got %s", r.URL.Path)
		}

		var req QiitaItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		respItem := QiitaItem{
			ID:      "new-item-123",
			Title:   req.Title,
			Body:    req.Body,
			Private: req.Private,
			Tags:    req.Tags,
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(respItem)
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	item, err := client.CreateItem(QiitaItemRequest{
		Title:   "新規記事",
		Body:    "テスト本文",
		Private: false,
		Tags:    []QiitaTag{{Name: "Go"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item.ID != "new-item-123" {
		t.Errorf("expected ID 'new-item-123', got %s", item.ID)
	}
}

func TestQiitaClient_UpdateItem(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/items/existing-id" {
			t.Errorf("expected path /items/existing-id, got %s", r.URL.Path)
		}

		var req QiitaItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		respItem := QiitaItem{
			ID:    "existing-id",
			Title: req.Title,
			Body:  req.Body,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respItem)
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	item, err := client.UpdateItem("existing-id", QiitaItemRequest{
		Title: "更新記事",
		Body:  "更新された本文",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item.Title != "更新記事" {
		t.Errorf("expected Title '更新記事', got %s", item.Title)
	}
}

func TestQiitaClient_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"scope required"}`))
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	_, err := client.CreateItem(QiitaItemRequest{Title: "Forbidden test"})
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}

	expectedParts := []string{"POST /items", "403", "scope required"}
	for _, part := range expectedParts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("expected error message to contain %q, got %q", part, err.Error())
		}
	}
}

func TestQiitaClient_InvalidJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not a valid json`))
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	_, err := client.CreateItem(QiitaItemRequest{Title: "Invalid JSON test"})
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	expectedParts := []string{"POST /items", "failed to parse JSON response"}
	for _, part := range expectedParts {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("expected error message to contain %q, got %q", part, err.Error())
		}
	}
}
