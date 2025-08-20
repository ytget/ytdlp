package cipher

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/robertkrimen/otto"
)

const (
	userAgentValue   = "Mozilla/5.0"
	ytBase           = "https://www.youtube.com"
	playerJSURLRe    = `"jsUrl":"([^"]+)"`
	decipherFuncName = "decipher"
	ncodeFuncName    = "ncode"
	jsURLGroupIndex  = 1 // capture group index for jsUrl
)

var (
	playerJSURLRegex = regexp.MustCompile(playerJSURLRe)
)

// simple player.js cache by URL
var (
	playerJSCache   = make(map[string]playerJSCacheEntry)
	playerJSCacheMu sync.Mutex
)

type playerJSCacheEntry struct {
	body  []byte
	expAt time.Time
}

const playerJSTTL = 10 * time.Minute

func getPlayerJS(httpClient *http.Client, playerJSURL string) ([]byte, error) {
	playerJSCacheMu.Lock()
	entry, ok := playerJSCache[playerJSURL]
	if ok && time.Now().Before(entry.expAt) {
		body := entry.body
		playerJSCacheMu.Unlock()
		return body, nil
	}
	playerJSCacheMu.Unlock()

	req, err := http.NewRequest("GET", playerJSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for player.js: %v", err)
	}
	req.Header.Set("User-Agent", userAgentValue)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download player.js: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read player.js content: %v", err)
	}

	playerJSCacheMu.Lock()
	playerJSCache[playerJSURL] = playerJSCacheEntry{body: body, expAt: time.Now().Add(playerJSTTL)}
	playerJSCacheMu.Unlock()
	return body, nil
}

// FetchPlayerJS finds the player.js URL by requesting the provided video page URL
// and scraping the "jsUrl" field from the response.
func FetchPlayerJS(httpClient *http.Client, videoURL string) (string, error) {
	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgentValue)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	matches := playerJSURLRegex.FindSubmatch(body)
	if len(matches) <= jsURLGroupIndex || len(matches[jsURLGroupIndex]) == 0 {
		return "", fmt.Errorf("could not find player js url in video page")
	}

	playerJSURL := strings.Replace(string(matches[jsURLGroupIndex]), `\/`, `/`, -1)

	return ytBase + playerJSURL, nil
}

// Decipher decrypts a signature using otto.
func Decipher(httpClient *http.Client, playerJSURL string, signature string) (string, error) {
	playerJSContent, err := getPlayerJS(httpClient, playerJSURL)
	if err != nil {
		return "", err
	}
	// Fast path: regex parser
	if out, ok := tryRegexDecipher(string(playerJSContent), signature); ok {
		return out, nil
	}

	vm := otto.New()
	_, err = vm.Run(string(playerJSContent))
	if err != nil {
		return "", fmt.Errorf("failed to run player.js in otto: %v", err)
	}

	value, err := vm.Call(decipherFuncName, nil, signature)
	if err != nil {
		return "", fmt.Errorf("failed to call decipher function: %v", err)
	}

	result, err := value.ToString()
	if err != nil {
		return "", fmt.Errorf("decipher function did not return a string: %v", err)
	}

	return result, nil
}

// DecipherN decodes the n-parameter (throttling) if player.js contains ncode().
func DecipherN(httpClient *http.Client, playerJSURL string, nval string) (string, error) {
	playerJSContent, err := getPlayerJS(httpClient, playerJSURL)
	if err != nil {
		return "", err
	}
	vm := otto.New()
	_, err = vm.Run(string(playerJSContent))
	if err != nil {
		return "", fmt.Errorf("failed to run player.js in otto: %v", err)
	}
	// Try to call ncode; if absent – return the original value
	fn, err := vm.Get(ncodeFuncName)
	if err != nil || !fn.IsFunction() {
		return nval, nil
	}
	value, err := vm.Call(ncodeFuncName, nil, nval)
	if err != nil {
		return "", fmt.Errorf("failed to call ncode function: %v", err)
	}
	result, err := value.ToString()
	if err != nil {
		return "", fmt.Errorf("ncode did not return a string: %v", err)
	}
	return result, nil
}
