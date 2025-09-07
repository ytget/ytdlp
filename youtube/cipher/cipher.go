package cipher

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robertkrimen/otto"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	userAgentValue   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	ytBase           = "https://www.youtube.com"
	playerJSURLRe    = `"jsUrl":"([^"]+)"`
	decipherFuncName = "decipher"
	ncodeFuncName    = "ncode"
	jsURLGroupIndex  = 1 // capture group index for jsUrl
)

var (
	playerJSURLRegex = regexp.MustCompile(playerJSURLRe)

	// Performance metrics
	metrics = struct {
		totalRequests     int64
		cacheHits         int64
		cacheMisses       int64
		avgDecipherTime   time.Duration
		totalDecipherTime time.Duration
		mu                sync.Mutex
	}{}
)

func init() {
	// Start cache cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()

			// Clean up player.js cache
			playerJSCacheMu.Lock()
			for url, entry := range playerJSCache {
				if now.After(entry.expAt) {
					delete(playerJSCache, url)
				}
			}
			playerJSCacheMu.Unlock()

			// Clean up signature cache
			signatureCacheMu.Lock()
			for sig, entry := range signatureCache {
				if now.After(entry.expAt) {
					delete(signatureCache, sig)
				}
			}
			signatureCacheMu.Unlock()

			// Log metrics
			metrics.mu.Lock()
			fmt.Printf("[METRICS] Total requests: %d, Cache hits: %d (%.1f%%), Cache misses: %d (%.1f%%), Avg decipher time: %v\n",
				metrics.totalRequests,
				metrics.cacheHits,
				float64(metrics.cacheHits)/float64(metrics.totalRequests)*100,
				metrics.cacheMisses,
				float64(metrics.cacheMisses)/float64(metrics.totalRequests)*100,
				metrics.avgDecipherTime,
			)
			metrics.mu.Unlock()
		}
	}()
}

// simple player.js cache by URL
var (
	playerJSCache   = make(map[string]playerJSCacheEntry)
	playerJSCacheMu sync.Mutex

	// Cache for deciphered signatures
	signatureCache   = make(map[string]signatureCacheEntry)
	signatureCacheMu sync.Mutex
)

type playerJSCacheEntry struct {
	body  []byte
	expAt time.Time
}

type signatureCacheEntry struct {
	value string
	expAt time.Time
}

const (
	playerJSTTL     = 10 * time.Minute
	signatureTTL    = 1 * time.Hour
	cleanupInterval = 5 * time.Minute
)

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
		return nil, NewError(ErrCodePlayerJSDownload, "Failed to create request for player.js", err)
	}
	req.Header.Set("User-Agent", userAgentValue)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, NewError(ErrCodePlayerJSDownload, "Failed to download player.js", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewError(ErrCodePlayerJSDownload, "Failed to read player.js content", err)
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
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

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

// DebugGetPlayerJS returns player.js body and source indicator ("cache" or "network").
func DebugGetPlayerJS(httpClient *http.Client, playerJSURL string) ([]byte, string, error) {
	playerJSCacheMu.Lock()
	entry, ok := playerJSCache[playerJSURL]
	if ok && time.Now().Before(entry.expAt) {
		body := entry.body
		playerJSCacheMu.Unlock()
		return body, "cache", nil
	}
	playerJSCacheMu.Unlock()

	req, err := http.NewRequest("GET", playerJSURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request for player.js: %v", err)
	}
	req.Header.Set("User-Agent", userAgentValue)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download player.js: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read player.js content: %v", err)
	}

	playerJSCacheMu.Lock()
	playerJSCache[playerJSURL] = playerJSCacheEntry{body: body, expAt: time.Now().Add(playerJSTTL)}
	playerJSCacheMu.Unlock()
	return body, "network", nil
}

// Decipher decrypts a signature using multiple fallback methods.
func Decipher(httpClient *http.Client, playerJSURL string, signature string) (string, error) {
	start := time.Now()
	fmt.Printf("[DEBUG] Starting Decipher for signature: %s\n", signature)

	// Update metrics
	metrics.mu.Lock()
	metrics.totalRequests++
	metrics.mu.Unlock()

	// Check cache first
	signatureCacheMu.Lock()
	if entry, ok := signatureCache[signature]; ok && time.Now().Before(entry.expAt) {
		signatureCacheMu.Unlock()
		fmt.Printf("[DEBUG] Using cached signature\n")

		// Update cache hit metrics
		metrics.mu.Lock()
		metrics.cacheHits++
		metrics.mu.Unlock()

		return entry.value, nil
	}
	signatureCacheMu.Unlock()

	// Update cache miss metrics
	metrics.mu.Lock()
	metrics.cacheMisses++
	metrics.mu.Unlock()

	playerJSContent, err := getPlayerJS(httpClient, playerJSURL)
	if err != nil {
		fmt.Printf("[ERROR] Failed to get player.js: %v\n", err)
		return "", NewError(ErrCodePlayerJSDownload, "Failed to download player.js", err)
	}
	fmt.Printf("[DEBUG] Got player.js content, size: %d bytes, time: %v\n", len(playerJSContent), time.Since(start))

	// Method 1: Minimal JS environment (preferred)
	fmt.Printf("[DEBUG] Trying minimal JS decipher method\n")
	if out, ok := tryMiniJSDecipher(string(playerJSContent), signature); ok {
		fmt.Printf("[DEBUG] Minimal JS decipher successful\n")
		// Cache successful result
		signatureCacheMu.Lock()
		signatureCache[signature] = signatureCacheEntry{
			value: out,
			expAt: time.Now().Add(signatureTTL),
		}
		signatureCacheMu.Unlock()
		return out, nil
	}
	fmt.Printf("[DEBUG] Minimal JS decipher failed\n")

	// Method 2: Regex parser (fast fallback)
	fmt.Printf("[DEBUG] Trying regex decipher method\n")
	if out, ok := tryRegexDecipher(string(playerJSContent), signature); ok {
		fmt.Printf("[DEBUG] Regex decipher successful\n")
		// Cache successful result
		signatureCacheMu.Lock()
		signatureCache[signature] = signatureCacheEntry{
			value: out,
			expAt: time.Now().Add(signatureTTL),
		}
		signatureCacheMu.Unlock()
		return out, nil
	}
	fmt.Printf("[DEBUG] Regex decipher failed\n")

	// Method 3: Full otto execution (last resort)
	fmt.Printf("[DEBUG] Trying otto decipher method\n")
	if out, ok := tryOttoDecipher(string(playerJSContent), signature); ok {
		fmt.Printf("[DEBUG] Otto decipher successful\n")
		// Cache successful result
		signatureCacheMu.Lock()
		signatureCache[signature] = signatureCacheEntry{
			value: out,
			expAt: time.Now().Add(signatureTTL),
		}
		signatureCacheMu.Unlock()
		return out, nil
	}
	fmt.Printf("[DEBUG] Otto decipher failed\n")

	// Method 4: Pattern-based fallback (last guard)
	fmt.Printf("[DEBUG] Trying pattern fallback method\n")
	if out, ok := tryPatternFallback(string(playerJSContent), signature); ok {
		fmt.Printf("[DEBUG] Pattern fallback successful\n")
		// Cache successful result
		signatureCacheMu.Lock()
		signatureCache[signature] = signatureCacheEntry{
			value: out,
			expAt: time.Now().Add(signatureTTL),
		}
		signatureCacheMu.Unlock()
		return out, nil
	}
	fmt.Printf("[DEBUG] Pattern fallback failed\n")

	// Update timing metrics
	elapsed := time.Since(start)
	metrics.mu.Lock()
	metrics.totalDecipherTime += elapsed
	metrics.avgDecipherTime = time.Duration(int64(metrics.totalDecipherTime) / metrics.totalRequests)
	metrics.mu.Unlock()

	return "", NewError(ErrCodeSignatureDecipher, "All decipher methods failed", map[string]any{
		"signature": signature,
		"attempts":  []string{"miniJS", "regex", "otto", "pattern"},
		"elapsed":   elapsed.String(),
	})
}

// tryOttoDecipher attempts to run the full player.js in otto with error handling.
func tryOttoDecipher(playerJS string, signature string) (string, bool) {
	vm, err := runJSWithTimeout(playerJS, 30*time.Second)
	if err != nil {
		// If otto fails, try to extract and run only the essential parts
		sanitizedJS := sanitizePlayerJS(playerJS)
		vm, err = runJSWithTimeout(sanitizedJS, 30*time.Second)
		if err != nil {
			return "", false
		}
	}

	value, err := vm.Call(decipherFuncName, nil, signature)
	if err != nil {
		return "", false
	}

	result, err := value.ToString()
	if err != nil {
		return "", false
	}

	return result, true
}

// sanitizePlayerJS removes or replaces problematic RegExp patterns that otto can't handle.
func sanitizePlayerJS(playerJS string) string {
	// Remove problematic RegExp patterns that cause otto to fail
	// These patterns include lookaheads, negative lookaheads, and other modern RegExp features

	// Replace lookahead patterns (?=...)
	lookaheadRe := regexp.MustCompile(`\?=[^)]*\)`)
	playerJS = lookaheadRe.ReplaceAllString(playerJS, "")

	// Replace negative lookahead patterns (?!...)
	negLookaheadRe := regexp.MustCompile(`\?![^)]*\)`)
	playerJS = negLookaheadRe.ReplaceAllString(playerJS, "")

	// Replace lookbehind patterns (?<=...)
	lookbehindRe := regexp.MustCompile(`\?<=[^)]*\)`)
	playerJS = lookbehindRe.ReplaceAllString(playerJS, "")

	// Replace negative lookbehind patterns (?<!...)
	negLookbehindRe := regexp.MustCompile(`\?<![^)]*\)`)
	playerJS = negLookbehindRe.ReplaceAllString(playerJS, "")

	// Replace named capture groups (?<name>...)
	namedCaptureRe := regexp.MustCompile(`\?<[^>]*>`)
	playerJS = namedCaptureRe.ReplaceAllString(playerJS, "")

	// Replace atomic groups (?>...)
	atomicGroupRe := regexp.MustCompile(`\?>[^)]*\)`)
	playerJS = atomicGroupRe.ReplaceAllString(playerJS, "")

	// Clean up any remaining problematic patterns
	// Remove any remaining ? patterns that might cause issues
	questionRe := regexp.MustCompile(`\?[^)]*\)`)
	playerJS = questionRe.ReplaceAllString(playerJS, "")

	// Clean up any empty parentheses that might be left
	emptyParensRe := regexp.MustCompile(`\(\s*\)`)
	playerJS = emptyParensRe.ReplaceAllString(playerJS, "")

	// Clean up any remaining single parentheses
	singleParenRe := regexp.MustCompile(`\(\s*;`)
	playerJS = singleParenRe.ReplaceAllString(playerJS, ";")

	// Clean up any remaining single parentheses at end of lines
	singleParenEndRe := regexp.MustCompile(`\(\s*$`)
	playerJS = singleParenEndRe.ReplaceAllString(playerJS, "")

	return playerJS
}

// tryPatternFallback implements a basic pattern-based signature transformation
// as a last resort when all other methods fail.
func tryPatternFallback(playerJS string, signature string) (string, bool) {
	// This is a simplified fallback that tries to identify common patterns
	// without executing JavaScript

	// Look for common transformation patterns in the code
	if strings.Contains(playerJS, "reverse") && strings.Contains(playerJS, "join") {
		// Simple reverse pattern
		runes := []rune(signature)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), true
	}

	// Look for splice patterns
	if strings.Contains(playerJS, "splice") {
		// Try to detect splice offset from the code
		spliceRe := regexp.MustCompile(`splice\((\d+)`)
		if matches := spliceRe.FindStringSubmatch(playerJS); len(matches) > 1 {
			if offset, err := strconv.Atoi(matches[1]); err == nil && len(signature) > offset {
				return signature[offset:], true
			}
		}
		// Fallback to common splice offsets
		for offset := 1; offset <= 10; offset++ {
			if len(signature) > offset {
				return signature[offset:], true
			}
		}
	}

	return "", false
}

// runJSWithTimeout runs JavaScript code with a timeout to prevent hanging
func runJSWithTimeout(code string, timeout time.Duration) (*otto.Otto, error) {
	vm := otto.New()

	// Create a channel to signal completion
	done := make(chan error, 1)

	go func() {
		_, err := vm.Run(code)
		done <- err
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return vm, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("JavaScript execution timeout after %v", timeout)
	}
}

// tryMiniJSDecipher extracts the transform object and decipher call sequence, assembles
// a minimal JavaScript program, and executes it in otto to avoid modern JS features elsewhere.
func tryMiniJSDecipher(playerJS string, signature string) (string, bool) {
	// 1) Locate decipher function (name, param, body)
	fnRe := regexp.MustCompile(`function\s*([a-zA-Z0-9$]*)\s*\(\s*([a-zA-Z0-9$]+)\s*\)\s*\{([\s\S]*?)\}`)
	matches := fnRe.FindAllStringSubmatch(playerJS, -1)
	var param, body string
	for _, m := range matches {
		p := m[2]
		b := m[3]
		if strings.Contains(b, p+`.split("")`) && strings.Contains(b, `return `+p+`.join("")`) {
			param = p
			body = b
			break
		}
	}
	if param == "" || body == "" {
		return "", false
	}

	// 2) Find object name from callsites
	objNameRe := regexp.MustCompile(`([a-zA-Z0-9$]+)\.[a-zA-Z0-9$]+\(` + regexp.QuoteMeta(param) + `(?:,\s*\d+)?\)`) // first object name occurrence
	om := objNameRe.FindStringSubmatch(body)
	if len(om) < 2 {
		return "", false
	}
	obj := om[1]

	// 3) Extract transform object literal (support var|let|const and optional semicolon)
	objRe := regexp.MustCompile(`(?:var|let|const)\s+` + regexp.QuoteMeta(obj) + `\s*=\s*\{([\s\S]*?)\}\s*;?`)
	om2 := objRe.FindStringSubmatch(playerJS)
	if len(om2) < 2 {
		return "", false
	}
	objBody := om2[1]

	// 4) Extract ordered calls and optional numeric arguments
	callRe := regexp.MustCompile(regexp.QuoteMeta(obj) + `\.([a-zA-Z0-9$]+)\(` + regexp.QuoteMeta(param) + `(?:,\s*(\d+))?\)`) // captures fn and optional number
	calls := callRe.FindAllStringSubmatch(body, -1)
	if len(calls) == 0 {
		return "", false
	}

	// 5) Assemble a minimal JS code snippet
	var sb strings.Builder
	sb.WriteString("var ")
	sb.WriteString(obj)
	sb.WriteString("={")
	sb.WriteString(objBody)
	sb.WriteString("};\n")
	sb.WriteString("function ")
	sb.WriteString(decipherFuncName)
	sb.WriteString("(")
	sb.WriteString(param)
	sb.WriteString("){")
	sb.WriteString("\n")
	sb.WriteString(param)
	sb.WriteString("=")
	sb.WriteString(param)
	sb.WriteString(".split(\"\");\n")
	for _, c := range calls {
		fn := c[1]
		arg := c[2]
		sb.WriteString(obj)
		sb.WriteString(".")
		sb.WriteString(fn)
		sb.WriteString("(")
		sb.WriteString(param)
		if arg != "" {
			sb.WriteString(",")
			sb.WriteString(arg)
		}
		sb.WriteString(");\n")
	}
	sb.WriteString("return ")
	sb.WriteString(param)
	sb.WriteString(".join(\"\");}")
	sb.WriteString("\n")

	vm, err := runJSWithTimeout(sb.String(), 30*time.Second)
	if err != nil {
		return "", false
	}
	val, err := vm.Call(decipherFuncName, nil, signature)
	if err != nil {
		return "", false
	}
	res, err := val.ToString()
	if err != nil {
		return "", false
	}
	return res, true
}

// DecipherN decodes the n-parameter (throttling) if player.js contains ncode().
func DecipherN(httpClient *http.Client, playerJSURL string, nval string) (string, error) {
	playerJSContent, err := getPlayerJS(httpClient, playerJSURL)
	if err != nil {
		return "", err
	}

	// Try original first
	vm, err := runJSWithTimeout(string(playerJSContent), 30*time.Second)
	if err != nil {
		// Fallback to sanitized version
		sanitizedJS := sanitizePlayerJS(string(playerJSContent))
		vm, err = runJSWithTimeout(sanitizedJS, 30*time.Second)
		if err != nil {
			return nval, nil
		}
	}

	// Try to call ncode; if absent – return the original value
	fn, err := vm.Get(ncodeFuncName)
	if err != nil || !fn.IsFunction() {
		return nval, nil
	}
	value, err := vm.Call(ncodeFuncName, nil, nval)
	if err != nil {
		// If ncode call fails, return original value
		return nval, nil
	}
	result, err := value.ToString()
	if err != nil {
		// If ncode result is not a string, return original value
		return nval, nil
	}
	return result, nil
}
