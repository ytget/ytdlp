package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ytget/ytdlp/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <playlist_id>")
		return
	}
	playlistID := os.Args[1]
	d := ytdlp.New()
	items, err := d.GetPlaylistItemsAll(context.Background(), playlistID, 50)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for i, it := range items {
		fmt.Printf("%02d. %s (%s)\n", i+1, it.Title, it.VideoID)
	}
}
