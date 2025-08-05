package kratos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteIdentifierFirstUpdateLoginRequest(t *testing.T) {
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", req.Method)
		}
		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer server.Close()

	client := NewClient(server.URL, false)
	ctx := context.Background()

	resp, err := client.ExecuteIdentifierFirstUpdateLoginRequest(ctx, "flow123", "csrf_token_1234", "test@example.com", cookies)

	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", resp.StatusCode)
	}
}
