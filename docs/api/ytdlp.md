## package ytdlp

High-level API for downloading YouTube videos.

### Key Types
- `type Downloader`
- `type DownloadOptions`
- `type Progress`
- `type VideoInfo`

### Key Methods
- `New() *Downloader`
- `(*Downloader) WithFormat(quality, ext string) *Downloader`
- `(*Downloader) WithHTTPClient(c *http.Client) *Downloader`
- `(*Downloader) WithProgress(func(Progress)) *Downloader`
- `(*Downloader) WithOutputPath(path string) *Downloader`
- `(*Downloader) WithRateLimit(bps int64) *Downloader`
- `(*Downloader) Download(ctx context.Context, videoURL string) (*VideoInfo, error)`
- `(*Downloader) GetPlaylistItems(ctx, playlistID string, limit int) ([]types.PlaylistItem, error)`
- `(*Downloader) GetPlaylistItemsAll(ctx, playlistID string, limit int) ([]types.PlaylistItem, error)`


