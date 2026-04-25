package mhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("key") != "value" {
			t.Errorf("expected query key=value, got %s", r.URL.Query().Get("key"))
		}
		if r.Header.Get("X-Custom") != "header" {
			t.Errorf("expected X-Custom header, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello"}`))
	}))
	defer ts.Close()

	client := NewClient(WithTimeout(5 * time.Second))
	resp, err := client.Get(ts.URL).
		Query("key", "value").
		SetHeader("X-Custom", "header").
		Do()

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if !resp.IsOK() {
		t.Fatalf("expected status 200, got %d", resp.StatusCode())
	}

	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("expected message=hello, got %s", result["message"])
	}
}

func TestClientPostJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var data map[string]interface{}
		json.Unmarshal(body, &data)

		if data["name"] != "alice" {
			t.Errorf("expected name=alice, got %v", data["name"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))
	defer ts.Close()

	client := NewClient()
	reqBody := map[string]string{"name": "alice"}
	resp, err := client.Post(ts.URL).BodyJSON(reqBody).Do()

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode())
	}

	var result map[string]int
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if result["id"] != 1 {
		t.Errorf("expected id=1, got %d", result["id"])
	}
}

func TestClientDoAndBindString(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text response"))
	}))
	defer ts.Close()

	client := NewClient()
	str, err := client.Get(ts.URL).DoAndBindString()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if str != "plain text response" {
		t.Errorf("expected 'plain text response', got %s", str)
	}
}

func TestClientDoAndBindBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("byte response"))
	}))
	defer ts.Close()

	client := NewClient()
	data, err := client.Get(ts.URL).DoAndBindBytes()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if string(data) != "byte response" {
		t.Errorf("expected 'byte response', got %s", string(data))
	}
}

func TestClientDoAndBindJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"count":42}`))
	}))
	defer ts.Close()

	client := NewClient()
	var result struct {
		Success bool `json:"success"`
		Count   int  `json:"count"`
	}

	err := client.Get(ts.URL).DoAndBindJSON(&result)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success=true")
	}
	if result.Count != 42 {
		t.Errorf("expected count=42, got %d", result.Count)
	}
}

func TestClientAuthorization(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken" {
			t.Errorf("expected Bearer mytoken, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewClient()
	resp, err := client.Get(ts.URL).SetBearerToken("mytoken").Do()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !resp.IsOK() {
		t.Errorf("expected 200, got %d", resp.StatusCode())
	}
}

func TestResponseMethods(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer ts.Close()

	client := NewClient()
	resp, err := client.Get(ts.URL).Do()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.IsSuccess() {
		t.Error("expected IsSuccess() to be false")
	}
	if !resp.IsError() {
		t.Error("expected IsError() to be true")
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode())
	}
	if resp.GetHeader("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header=custom-value, got %s", resp.GetHeader("X-Custom-Header"))
	}
	if resp.ContentType() != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", resp.ContentType())
	}
}

func TestClientBodyString(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test body" {
			t.Errorf("expected body 'test body', got %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewClient()
	resp, err := client.Post(ts.URL).BodyString("test body").Do()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !resp.IsOK() {
		t.Errorf("expected 200, got %d", resp.StatusCode())
	}
}

func TestClientMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		call   func(*Client, string) *Request
	}{
		{"Get", http.MethodGet, func(c *Client, url string) *Request { return c.Get(url) }},
		{"Post", http.MethodPost, func(c *Client, url string) *Request { return c.Post(url) }},
		{"Put", http.MethodPut, func(c *Client, url string) *Request { return c.Put(url) }},
		{"Delete", http.MethodDelete, func(c *Client, url string) *Request { return c.Delete(url) }},
		{"Patch", http.MethodPatch, func(c *Client, url string) *Request { return c.Patch(url) }},
		{"Head", http.MethodHead, func(c *Client, url string) *Request { return c.Head(url) }},
		{"Options", http.MethodOptions, func(c *Client, url string) *Request { return c.Options(url) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("expected %s, got %s", tt.method, r.Method)
				}
				if tt.method != http.MethodHead {
					w.Write([]byte("ok"))
				}
			}))
			defer ts.Close()

			client := NewClient()
			req := tt.call(client, ts.URL)
			resp, err := req.Do()
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode() != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode())
			}
		})
	}
}
