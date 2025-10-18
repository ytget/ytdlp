package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	client := New()

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.HTTPClient == nil {
		t.Fatal("Expected HTTPClient to be initialized")
	}

	if client.HTTPClient.Timeout != defaultTimeout {
		t.Errorf("Expected timeout %v, got %v", defaultTimeout, client.HTTPClient.Timeout)
	}

	if client.Retries != defaultRetries {
		t.Errorf("Expected retries %d, got %d", defaultRetries, client.Retries)
	}

	if client.UserAgent != userAgentValue {
		t.Errorf("Expected user agent '%s', got '%s'", userAgentValue, client.UserAgent)
	}
}

func TestNewWith(t *testing.T) {
	cfg := Config{
		Timeout:   10 * time.Second,
		Retries:   5,
		UserAgent: "Custom Agent",
		ProxyURL:  "http://proxy.example.com:8080",
	}

	client := NewWith(cfg)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.HTTPClient.Timeout != cfg.Timeout {
		t.Errorf("Expected timeout %v, got %v", cfg.Timeout, client.HTTPClient.Timeout)
	}

	if client.Retries != cfg.Retries {
		t.Errorf("Expected retries %d, got %d", cfg.Retries, client.Retries)
	}

	if client.UserAgent != cfg.UserAgent {
		t.Errorf("Expected user agent '%s', got '%s'", cfg.UserAgent, client.UserAgent)
	}
}

func TestNewWithZeroValues(t *testing.T) {
	cfg := Config{}

	client := NewWith(cfg)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.HTTPClient.Timeout != defaultTimeout {
		t.Errorf("Expected timeout %v, got %v", defaultTimeout, client.HTTPClient.Timeout)
	}

	if client.Retries != defaultRetries {
		t.Errorf("Expected retries %d, got %d", defaultRetries, client.Retries)
	}

	if client.UserAgent != userAgentValue {
		t.Errorf("Expected user agent '%s', got '%s'", userAgentValue, client.UserAgent)
	}
}

func TestNewWithNegativeValues(t *testing.T) {
	cfg := Config{
		Timeout: -1 * time.Second,
		Retries: -1,
	}

	client := NewWith(cfg)

	if client.HTTPClient.Timeout != defaultTimeout {
		t.Errorf("Expected timeout %v, got %v", defaultTimeout, client.HTTPClient.Timeout)
	}

	if client.Retries != defaultRetries {
		t.Errorf("Expected retries %d, got %d", defaultRetries, client.Retries)
	}
}

func TestNewWithEmptyUserAgent(t *testing.T) {
	cfg := Config{
		UserAgent: "",
	}

	client := NewWith(cfg)

	if client.UserAgent != userAgentValue {
		t.Errorf("Expected user agent '%s', got '%s'", userAgentValue, client.UserAgent)
	}
}

func TestNewWithInvalidProxy(t *testing.T) {
	cfg := Config{
		ProxyURL: "invalid-proxy-url",
	}

	client := NewWith(cfg)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	// Should still create client even with invalid proxy
	if client.HTTPClient == nil {
		t.Fatal("Expected HTTPClient to be initialized")
	}
}

func TestGetSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := New()
	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	_ = resp.Body.Close()
}

func TestGetWithCustomUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent != userAgentValue {
			t.Errorf("Expected User-Agent '%s', got '%s'", userAgentValue, userAgent)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New()
	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_ = resp.Body.Close()
}

func TestGetWithEmptyUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent != userAgentValue {
			t.Errorf("Expected User-Agent '%s', got '%s'", userAgentValue, userAgent)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Retries:    1,
		UserAgent:  "", // Empty user agent
	}

	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_ = resp.Body.Close()
}

func TestGetWithZeroRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Retries:    0, // Zero retries
		UserAgent:  userAgentValue,
	}

	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_ = resp.Body.Close()
}

func TestGetWithNegativeRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Retries:    -1, // Negative retries
		UserAgent:  userAgentValue,
	}

	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_ = resp.Body.Close()
}

func TestProxyFromURLString(t *testing.T) {
	proxyURL := "http://proxy.example.com:8080"
	proxyFunc, err := proxyFromURLString(proxyURL)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if proxyFunc == nil {
		t.Fatal("Expected proxy function to be non-nil")
	}
}

func TestProxyFromURLStringInvalid(t *testing.T) {
	proxyURL := "://invalid-url"
	_, err := proxyFromURLString(proxyURL)

	if err == nil {
		t.Fatal("Expected error for invalid proxy URL")
	}
}
