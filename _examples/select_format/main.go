package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ytget/ytdlp/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <video_url>")
		return
	}
	url := os.Args[1]
	dl := ytdlp.New().WithFormat("height<=480", "mp4")
	info, err := dl.Download(context.Background(), url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Saved:", info.Title)
}
