## package types

### Format
Fields:
- `Itag int`
- `URL string`
- `Quality string` (e.g., `720p`)
- `MimeType string` (e.g., `video/mp4; codecs=...`)
- `Bitrate int`
- `Size int64`
- `SignatureCipher string` (raw cipher; resolved to URL later)

### PlaylistItem
Fields:
- `VideoID string`
- `Title string`
- `Index int`


