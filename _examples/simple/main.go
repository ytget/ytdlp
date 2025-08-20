package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ytget/ytdlp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <video_url>")
		return
	}
	videoURL := os.Args[1]

	dl := ytdlp.New().WithProgress(func(p ytdlp.Progress) {
		fmt.Printf("Downloaded %.2f%% \r", p.Percent)
	})

	videoInfo, err := dl.Download(context.Background(), videoURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nDownload complete: %s\n", videoInfo.Title)
}
