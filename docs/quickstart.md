## Quick Start

### Library
```go
dl := ytdlp.New().WithFormat("", "mp4").WithProgress(func(p ytdlp.Progress) { /* ... */ })
info, err := dl.Download(ctx, "https://example.com/video/123")
```

### CLI
```bash
ytdlp --ext mp4 https://example.com/video/123

# Short-form videos
ytdlp https://example.com/shorts/abc123

# playlist
ytdlp --playlist --limit 25 --concurrency 4 'https://example.com/playlist/PLxxxx'
```


