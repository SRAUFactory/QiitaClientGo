package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			items := []QiitaItem{
				{ID: "item1", Title: "Article 1", UpdatedAt: "2026-10-01T00:00:00Z"},
				{ID: "item2", Title: "Article 2", UpdatedAt: "2026-10-02T00:00:00Z"},
			}
			_ = json.NewEncoder(w).Encode(items)
			return
		}

		if page == "2" {
			items := []QiitaItem{
				{ID: "item3", Title: "Article 3", UpdatedAt: "2026-10-03T00:00:00Z"},
			}
			_ = json.NewEncoder(w).Encode(items)
			return
		}

		// Page 3: empty, ends pagination
		_ = json.NewEncoder(w).Encode([]QiitaItem{})
	}))
	defer ts.Close()

	client := NewQiitaClient("test-token", ts.URL, 5*time.Second)
	items, err := client.FetchAllAuthenticatedUserItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].ID != "item1" || items[2].ID != "item3" {
		t.Errorf("unexpected items content: %#v", items)
	}
	if pageCalls != 3 {
		t.Errorf("expected 3 page requests, got %d", pageCalls)
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
