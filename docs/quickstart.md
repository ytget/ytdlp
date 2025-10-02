## Quick Start

### Library
```go
dl := ytdlp.New().WithFormat("", "mp4").WithProgress(func(p ytdlp.Progress) { /* ... */ })
info, err := dl.Download(ctx, "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
```

### CLI
```bash
ytdlp --ext mp4 https://www.youtube.com/watch?v=dQw4w9WgXcQ

# YouTube Shorts
ytdlp https://youtube.com/shorts/brZCOVlyPPo

# playlist
ytdlp --playlist --limit 25 --concurrency 4 'https://www.youtube.com/playlist?list=PLxxxx'
```


