## package downloader

Types:
- `type Downloader`
- `type Progress`

Constructors:
- `New(client *http.Client, progress func(Progress), rateLimitBps int64) *Downloader`

Methods:
- `(*Downloader) Download(ctx context.Context, url, outputPath string) error`

Notes:
- Chunked HTTP with retries and simple backoff
- Optional rate limiting (bytes per second)
- Resumes via temporary file when present


