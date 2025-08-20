# ytdlp

Native Go library and CLI to download YouTube videos — no external binaries, Android-friendly. MVP focuses on progressive formats (video+audio) like MP4 (itag 22/18). No HLS/DASH or muxing on first stage.

## Status
- MVP in progress: YouTube only, progressive formats only.
- Signature deciphering implemented (regex fast-path, JS fallback via otto), `n`-throttling supported.
- No ffmpeg, no merging adaptive streams yet.

## Install
```bash
go get github.com/ytget/ytdlp
```

CLI binary:
```bash
go install github.com/ytget/ytdlp/cmd/ytdlp@latest
```

## Quick Start (library)
```go
package main

import (
	"context"
	"fmt"
	"github.com/ytget/ytdlp"
)

func main() {
	d := ytdlp.New().WithOutputPath("").WithProgress(func(p ytdlp.Progress) {
		fmt.Printf("%.1f%%\r", p.Percent)
	})
	info, err := d.Download(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		panic(err)
	}
	fmt.Println("\nSaved:", info.Title)
}
```

## Quick Start (CLI)
```bash
# Best mp4 by default
ytdlp https://www.youtube.com/watch?v=dQw4w9WgXcQ

# Select by itag
ytdlp --format itag=22 https://www.youtube.com/watch?v=dQw4w9WgXcQ

# Constrain by height
ytdlp --format 'height<=480' https://www.youtube.com/watch?v=dQw4w9WgXcQ

# Playlist subset
ytdlp --playlist --limit 25 --concurrency 4 'https://www.youtube.com/playlist?list=PLxxxx'
```

### Common flags
- `--format` — `itag=NN`, `best`, `height<=N`
- `--ext` — `mp4`, `webm`
- `--output` — file or directory
- `--rate-limit` — `2MiB/s`, `500KiB/s`
- `--http-timeout` — `30s`, `1m`
- `--retries` — retry attempts (default 3)

See full reference in `docs/cli.md`.

Notes:
- If `OutputPath` is empty, file name is derived from `Title` and MIME (safe filename).
- Progressive MP4 selection preference: itag 22 → 18 → first progressive `video/mp4` with `avc1`.
- Format selectors: `ext`, `itag=NN`, `best|worst`, `height<=/height>=`.

## Playlists (MVP)
```go
items, err := ytdlp.New().GetPlaylistItemsAll(context.Background(), "PLxxxx", 200)
// items: []types.PlaylistItem{ VideoID, Title, Index }
```

## Documentation
- CLI flags and usage: see `docs/cli.md`
- API reference overview: see `docs/api/README.md`
- Format selection: see `docs/formats.md`
- Errors and troubleshooting: see `docs/errors.md` and `docs/troubleshooting.md`

## Make targets
- `make test` / `make race`
- `make cover` — coverage report (`coverage.out`, `coverage.html`)
- `make e2e` / `make e2e-url URL="https://..."`

## Errors
Library returns typed errors for common cases: unavailable/private/age-restricted/geo/rate-limited. Check error values from `errs` package.

## FAQ
- Why do I get "age restricted" or "login required"? — YouTube may require authentication for some videos. The library does not handle login; filter or skip such content on client side.
- Why 429/ratelimit? — Too many requests from your IP. Slow down requests, add backoff, or try later. The client already retries with exponential backoff for transient errors.
- Geo blocked? — Content not available in your region. The library returns a typed error; you must handle this on the application side.
- Download stuck at 0%? — Some hosts enforce throttling via `n` parameter. Ensure decipher is up-to-date; this library implements `n` transformation based on the player.js.

## E2E Test (manual)
Run only when you want a real download test:
```bash
YTDLP_E2E=1 go test -tags e2e ./e2e -v
```
Optionally specify a URL:
```bash
YTDLP_E2E=1 YTDLP_E2E_URL="https://www.youtube.com/watch?v=dQw4w9WgXcQ" go test -tags e2e ./e2e -v
```

## Android
- Pure Go; suitable for gomobile/Fyne builds.
- Ensure proper storage permissions and SAF/MediaStore usage on app side.

## Limitations (MVP)
- YouTube only.
- Progressive formats only (no adaptive muxing yet).
- Live streams, HLS/DASH are out of scope (for now).

## Roadmap (short)
- Robust decipher/n-throttling parser with test fixtures.
- Playlists via InnerTube browse/continuations.
- Adaptive formats + muxing in a later phase.

## License
MIT

