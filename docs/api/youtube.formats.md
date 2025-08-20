## package youtube/formats

Key functions:
- `ParseFormats(*innertube.PlayerResponse) ([]types.Format, error)`
- `DecryptSignatures(httpClient *http.Client, formats []types.Format, playerJSURL string) error`
- `SelectFormat(formats []types.Format, quality, ext string) *types.Format`

Selector syntax: `itag=NN`, `best|worst`, `height<=N`, `height>=N`, with optional extension filter.


