## package client

Constructors:
- `New() *Client`
- `NewWith(Config) *Client`

Types:
- `type Config` — Timeout, Retries, UserAgent, ProxyURL
- `type Client` — HTTPClient, Retries, UserAgent

Methods:
- `(*Client) Get(url string) (*http.Response, error)` — GET with retries and UA


