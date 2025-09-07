package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func testInnerTubeAPI() {
	fmt.Println("\n4️⃣ Testing InnerTube API with HTTP/1.1...")

	// Create HTTP client with our fixes
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false, // Our fix: disable HTTP/2
			MaxIdleConns:      100,
			IdleConnTimeout:   90 * time.Second,
		},
		Timeout: 30 * time.Second,
	}

	// Test URLs that should trigger InnerTube API calls
	testURLs := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=jNQXAC9IVRw",
	}

	for i, url := range testURLs {
		fmt.Printf("   Testing InnerTube API for URL %d: %s\n", i+1, url)

		// Test 1: Basic page access (should trigger API key extraction)
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("      ❌ Failed to access YouTube page: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("      ✅ YouTube page accessible (status: %d)\n", resp.StatusCode)

		// Test 2: Check if page contains InnerTube API key patterns
		if resp.StatusCode == 200 {
			// Read a small portion to check for API key patterns
			buffer := make([]byte, 1024*10) // 10KB should be enough
			n, err := resp.Body.Read(buffer)
			if err != nil && n == 0 {
				fmt.Printf("      ❌ Failed to read response body: %v\n", err)
				continue
			}

			content := string(buffer[:n])

			// Check for InnerTube API key patterns
			if strings.Contains(content, "INNERTUBE_API_KEY") {
				fmt.Printf("      ✅ InnerTube API key pattern found in page\n")
			} else {
				fmt.Printf("      ⚠️  InnerTube API key pattern not found (may be in JavaScript)\n")
			}

			// Check for player.js references
			if strings.Contains(content, "jsUrl") || strings.Contains(content, "player.js") {
				fmt.Printf("      ✅ Player.js references found in page\n")
			} else {
				fmt.Printf("      ⚠️  Player.js references not found\n")
			}
		}
	}
}

func testHTTPHeaders() {
	fmt.Println("\n5️⃣ Testing HTTP Headers and User-Agent...")

	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
		},
		Timeout: 30 * time.Second,
	}

	// Test with enhanced headers (similar to our innertube client)
	req, err := http.NewRequest("GET", "https://www.youtube.com", nil)
	if err != nil {
		fmt.Printf("   ❌ Failed to create request: %v\n", err)
		return
	}

	// Set headers similar to our innertube client
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   ❌ Failed to make request with enhanced headers: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Enhanced headers request successful (status: %d)\n", resp.StatusCode)
	fmt.Printf("   ✅ User-Agent: %s\n", req.Header.Get("User-Agent"))
}
