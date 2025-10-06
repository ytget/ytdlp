package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ytget/ytdlp/v2"
	"github.com/ytget/ytdlp/v2/client"
)

func main() {
	cfg := client.Config{Timeout: 15 * time.Second, Retries: 5, UserAgent: "MyApp/1.0"}
	c := client.NewWith(cfg)
	d := ytdlp.New().WithHTTPClient(c.HTTPClient).WithRateLimit(2 * 1024 * 1024) // 2 MiB/s
	info, err := d.Download(context.Background(), "https://example.com/video/123")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Saved:", info.Title)
}
