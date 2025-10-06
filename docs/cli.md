## CLI Usage

```bash
ytdlp [flags] <video_url>
```

### Flags
- `--format string` — Format selector (e.g., `itag=22`, `best`, `height<=480`)
- `--ext string` — Desired extension (e.g., `mp4`, `webm`)
- `--output string` — Output path (file or directory)
- `--no-progress` — Disable progress output
- `--rate-limit string` — Download rate limit (e.g., `2MiB/s`)
- `--http-timeout duration` — HTTP timeout (default `30s`)
- `--retries int` — HTTP retries (default `3`)
- `--ua string` — Override User-Agent
- `--proxy string` — Proxy URL

### Flags Reference

| Flag | Type | Default | Description | Maps to |
|------|------|---------|-------------|---------|
| `--format` | string | empty | Format selector: `itag=NN`, `best`, `worst`, `height<=N`, `height>=N` | `ytdlp.WithFormat(quality, ext)` (quality) |
| `--ext` | string | empty | Desired extension (case-insensitive). Examples: `mp4`, `webm` | `ytdlp.WithFormat(quality, ext)` (ext) |
| `--output` | string | empty | Output file or directory. When empty, derives `Title + ext` | `ytdlp.WithOutputPath(path)` |
| `--no-progress` | bool | false | Disable progress output | omit `ytdlp.WithProgress` |
| `--rate-limit` | string | empty | Limit download rate. Supports `KiB/MiB/GiB` or `KB/MB/GB`, optional `/s`. Examples: `2MiB/s`, `500KiB/s`, `5MB/s` | `ytdlp.WithRateLimit(bps)` |
| `--http-timeout` | duration | `30s` | HTTP client timeout. Go duration format (`300ms`, `10s`, `1m`) | `client.Config.Timeout` |
| `--retries` | int | `3` | Max retry attempts for transient errors (5xx, network) | `client.Config.Retries` |
| `--ua` | string | default desktop UA | Override User-Agent header | `client.Config.UserAgent` |
| `--proxy` | string | empty | HTTP/HTTPS/SOCKS proxy URL | `client.Config.ProxyURL` |
| `--playlist` | bool | false | Treat input as playlist URL or ID (`list=...`) | `(*ytdlp.Downloader).GetPlaylistItemsAll` |
| `--limit` | int | `0` (all) | Limit number of playlist items to process | `GetPlaylistItemsAll(limit)` |
| `--concurrency` | int | `1` | Parallel downloads for playlist items | CLI worker pool |

Notes:
- Precedence: `--format` defines candidate set; `--ext` further filters by extension.
- Rate limit parser accepts binary (`KiB/MiB/GiB`) and decimal (`KB/MB/GB`) units.

Planned flags:
- `--progress string` — `bar|plain|none`
- `--quiet` / `--verbose` — Logging verbosity
- `--version` — Print version

### Examples
```bash
# Best mp4
ytdlp https://example.com/video/123

# itag
ytdlp --format itag=22 <url>

# height constraint
ytdlp --format 'height<=480' <url>

# playlist
ytdlp --playlist --limit 25 --concurrency 4 'https://example.com/playlist/PLxxxx'
```


