package client

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	defaultRetries = 3

	userAgentValue   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	initialBackoff   = 200 * time.Millisecond
	maxBackoff       = 3 * time.Second
	successMinCode   = http.StatusOK                  // 200
	retryableMinCode = http.StatusInternalServerError // 500
)

// defaultTransport is a tuned HTTP transport reused across clients.
var defaultTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   10,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	// Enable HTTP/2
	ForceAttemptHTTP2: true,
	// Disable compression for regex parsing
	DisableCompression: true,
	// Add timeouts for read/write operations
	ReadBufferSize:  16 * 1024,
	WriteBufferSize: 16 * 1024,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
}

// Config holds optional client parameters. Zero values use defaults.
type Config struct {
	Timeout   time.Duration
	Retries   int
	UserAgent string
	ProxyURL  string
}

// Client wraps http.Client with retry/backoff and default headers.
type Client struct {
	HTTPClient *http.Client
	Retries    int
	UserAgent  string
}

// New creates a new Client with a tuned Transport, default timeout, and retries.
func New() *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout:   defaultTimeout,
			Transport: defaultTransport,
		},
		Retries:   defaultRetries,
		UserAgent: userAgentValue,
	}
}

// NewWith creates a new client with provided config. Zero values use defaults.
func NewWith(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	retries := cfg.Retries
	if retries <= 0 {
		retries = defaultRetries
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = userAgentValue
	}

	tr := defaultTransport.Clone()
	if cfg.ProxyURL != "" {
		if proxyFunc, err := proxyFromURLString(cfg.ProxyURL); err == nil {
			tr.Proxy = proxyFunc
		}
	}

	return &Client{
		HTTPClient: &http.Client{
			Timeout:   timeout,
			Transport: tr,
		},
		Retries:   retries,
		UserAgent: ua,
	}
}

// Get performs a GET request with a simple retry policy for transient errors
// (HTTP 5xx or network failures). It sets a desktop-like User-Agent header.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	ua := c.UserAgent
	if ua == "" {
		ua = userAgentValue
	}
	req.Header.Set("User-Agent", ua)

	retries := c.Retries
	if retries < 1 {
		retries = 1
	}
	var resp *http.Response
	backoff := initialBackoff
	for attempt := 0; attempt < retries; attempt++ {
		resp, err = c.HTTPClient.Do(req)
		if err == nil && resp != nil && resp.StatusCode >= successMinCode && resp.StatusCode < retryableMinCode {
			return resp, err
		}
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
	return resp, err
}

// proxyFromURLString parses a proxy URL and returns a Proxy function.
func proxyFromURLString(raw string) (func(*http.Request) (*url.URL, error), error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return http.ProxyURL(u), nil
}
