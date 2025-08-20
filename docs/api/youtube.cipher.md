## package youtube/cipher

Public helpers:
- `FetchPlayerJS(httpClient *http.Client, videoURL string) (string, error)` — locate player.js URL from a watch page
- `Decipher(httpClient *http.Client, playerJSURL, signature string) (string, error)` — decipher signature
- `DecipherN(httpClient *http.Client, playerJSURL, n string) (string, error)` — decode throttling parameter `n`


