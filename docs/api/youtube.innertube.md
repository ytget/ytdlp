## package youtube/innertube

Types:
- `type Client` — low-level InnerTube client
- `type PlayerResponse` — subset of /player response

Constructors:
- `New(httpClient *http.Client) *Client`

Methods:
- `(*Client) GetPlayerResponse(videoID string) (*PlayerResponse, error)`
- `(*Client) GetPlaylistItems(playlistID string, limit int) ([]types.PlaylistItem, error)`
- `(*Client) GetPlaylistItemsAll(playlistID string, limit int) ([]types.PlaylistItem, error)`


