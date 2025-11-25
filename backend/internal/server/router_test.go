package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIHappyPaths(t *testing.T) {
	router := NewRouter()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list_conversations", http.MethodGet, "/api/v1/conversations", ""},
		{"get_conversation", http.MethodGet, "/api/v1/conversations/conv-1", ""},
		{"list_conversation_messages", http.MethodGet, "/api/v1/conversations/conv-1/messages", ""},
		{"get_message", http.MethodGet, "/api/v1/messages/msg-1", ""},
		{"get_message_context", http.MethodGet, "/api/v1/messages/msg-1/context", ""},
		{"search", http.MethodPost, "/api/v1/search", `{"keyword":"demo","page":1,"page_size":10}`},
		{"list_trees", http.MethodGet, "/api/v1/trees", ""},
		{"update_tree_create", http.MethodPost, "/api/v1/tree/update", `{"conversation_uuids":["conv-1","conv-2"]}`},
		{"update_tree_update", http.MethodPost, "/api/v1/tree/update", `{"tree_id":"tree-1","conversation_uuids":["conv-1"]}`},
		{"get_tree", http.MethodGet, "/api/v1/trees/tree-1", ""},
		{"delete_tree", http.MethodDelete, "/api/v1/trees/tree-1", ""},
		{"create_favorite", http.MethodPost, "/api/v1/favorites", `{"target_type":"message","target_id":"msg-1","category":"default","notes":"demo"}`},
		{"list_favorites", http.MethodGet, "/api/v1/favorites", ""},
		{"delete_favorite", http.MethodDelete, "/api/v1/favorites/fav-1", ""},
		{"list_tags", http.MethodGet, "/api/v1/tags", ""},
		{"create_tag", http.MethodPost, "/api/v1/tags", `{"name":"监控","color":"#3B82F6"}`},
		{"add_conversation_tag", http.MethodPost, "/api/v1/conversation-tags", `{"tag_id":1,"conversation_uuid":"conv-1"}`},
		{"batch_add_conversation_tags", http.MethodPost, "/api/v1/conversation-tags/batch-add", `{"conversation_uuid":"conv-1","tag_ids":[1,2]}`},
		{"batch_remove_conversation_tags", http.MethodPost, "/api/v1/conversation-tags/batch-remove", `{"conversation_uuid":"conv-1","tag_ids":[2]}`},
		{"delete_conversation_tag", http.MethodDelete, "/api/v1/conversation-tags/1", ""},
		{"list_tag_conversations", http.MethodGet, "/api/v1/tags/1/conversations", ""},
		{"stats_overview", http.MethodGet, "/api/v1/stats/overview", ""},
		{"stats_by_date", http.MethodGet, "/api/v1/stats/by-date", ""},
		{"sync_batch", http.MethodPost, "/internal/v1/sync/batch", `{"source_type":"gpt","conversations":[{"uuid":"conv-1"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
			}

			var resp APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			if resp.Code != 0 {
				t.Fatalf("expected code 0, got %d, msg: %s", resp.Code, resp.Message)
			}
		})
	}
}
