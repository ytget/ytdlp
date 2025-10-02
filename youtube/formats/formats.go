package formats

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ytget/ytdlp/types"
	"github.com/ytget/ytdlp/youtube/cipher"
	"github.com/ytget/ytdlp/youtube/innertube"
)

var heightRe = regexp.MustCompile(`([0-9]{3,4})p`)

func getSubtype(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = mime[:i]
	}
	parts := strings.Split(mime, "/")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func parseHeight(label string) int {
	m := heightRe.FindStringSubmatch(label)
	if len(m) >= 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return v
		}
	}
	return 0
}

// ParseFormats parses the InnerTube player response and returns a list of
// available media formats (both progressive and adaptive) with minimal fields.
func ParseFormats(data *innertube.PlayerResponse) ([]types.Format, error) {
	var formats []types.Format
	allFormats := append(data.StreamingData.Formats, data.StreamingData.AdaptiveFormats...)

	for _, formatData := range allFormats {
		f, ok := formatData.(map[string]any)
		if !ok {
			continue
		}

		var itag int
		if v, ok := f["itag"].(float64); ok {
			iTAG := int(v)
			itag = iTAG
		}

		var bitrate int
		if v, ok := f["bitrate"].(float64); ok {
			bitrate = int(v)
		}

		var size int64
		if v, ok := f["contentLength"].(string); ok {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				size = parsed
			}
		}

		mimeType, _ := f["mimeType"].(string)
		quality, _ := f["qualityLabel"].(string)

		format := types.Format{
			Itag:     itag,
			MimeType: mimeType,
			Quality:  quality,
			Bitrate:  bitrate,
			Size:     size,
		}

		if urlVal, ok := f["url"].(string); ok {
			format.URL = urlVal
		} else if sc, ok := f["signatureCipher"].(string); ok {
			format.SignatureCipher = sc
		}

		formats = append(formats, format)
	}
	return formats, nil
}

// DecryptSignatures decrypts signatures for formats that use a signatureCipher
// by invoking cipher.Decipher and updating the URL in-place.
func DecryptSignatures(httpClient *http.Client, formats []types.Format, playerJSURL string) error {
	successCount := 0
	totalCount := 0
	skippedCount := 0

	// First pass: try to decrypt signatures
	for i := range formats {
		if formats[i].URL != "" {
			successCount++
			continue
		}
		if formats[i].SignatureCipher == "" {
			continue
		}

		totalCount++

		parsedCipher, err := url.ParseQuery(formats[i].SignatureCipher)
		if err != nil {
			fmt.Printf("Warning: Failed to parse cipher for format %d: %v\n", formats[i].Itag, err)
			skippedCount++
			continue
		}

		sig := parsedCipher.Get("s")
		sp := parsedCipher.Get("sp")
		if sp == "" {
			sp = "signature"
		}
		cipherURL := parsedCipher.Get("url")
		if cipherURL == "" || sig == "" {
			fmt.Printf("Warning: Missing signature or URL for format %d\n", formats[i].Itag)
			skippedCount++
			continue
		}

		// Try to decrypt with timeout
		decipheredSig, err := cipher.Decipher(httpClient, playerJSURL, sig)
		if err != nil {
			// Log error but continue with other formats
			fmt.Printf("Warning: Failed to decipher signature for format %d: %v\n", formats[i].Itag, err)
			skippedCount++
			continue
		}

		finalURL, err := url.Parse(cipherURL)
		if err != nil {
			fmt.Printf("Warning: Failed to parse URL for format %d: %v\n", formats[i].Itag, err)
			skippedCount++
			continue
		}

		query := finalURL.Query()
		query.Set(sp, decipheredSig)
		// Apply n-parameter decoding if present
		if nval := query.Get("n"); nval != "" {
			if nOut, err := cipher.DecipherN(httpClient, playerJSURL, nval); err == nil && nOut != "" {
				query.Set("n", nOut)
			}
		}
		// Ensure ratebypass
		if query.Get("ratebypass") == "" {
			query.Set("ratebypass", "yes")
		}
		// Encourage redirect behavior to non-alt hosts
		if query.Get("alr") == "" {
			query.Set("alr", "yes")
		}
		finalURL.RawQuery = query.Encode()

		formats[i].URL = finalURL.String()
		successCount++
	}

	fmt.Printf("Signature decryption: %d/%d formats processed successfully, %d skipped\n", successCount, totalCount, skippedCount)

	// If we have no formats with URLs, try to use formats without signatures
	if successCount == 0 {
		fmt.Printf("No formats with decrypted signatures, trying formats without signatures...\n")
		for i := range formats {
			if formats[i].URL != "" {
				successCount++
			}
		}
		fmt.Printf("Total formats available: %d\n", successCount)
	}

	return nil
}

// SelectFormat chooses the best format according to criteria without requiring direct URLs.
// Supported selectors:
//   - ext: file extension ("mp4", "webm")
//   - itag=NN: specific format by itag (e.g., "itag=22" for 720p MP4)
//   - best: highest quality (height, then bitrate)
//   - worst: lowest quality
//   - height<=NNN: height no more than NNN (e.g., "height<=720")
//   - height>=NNN: height no less than NNN (e.g., "height>=480")
//
// If selector is absent or no match found, heuristic is used:
// prefer itag 22 (720p MP4), then itag 18 (360p MP4),
// then progressive mp4 with avc1, else first available.
func SelectFormat(formats []types.Format, quality, ext string) *types.Format {
	all := make([]types.Format, 0, len(formats))
	all = append(all, formats...)

	// filter by extension if provided
	filtered := make([]types.Format, 0, len(all))
	for i := range all {
		if mimeSubtypeEquals(all[i], ext) {
			filtered = append(filtered, all[i])
		}
	}
	if len(filtered) == 0 {
		filtered = all
	}

	q := strings.TrimSpace(strings.ToLower(quality))
	// explicit itag selector
	if strings.HasPrefix(q, "itag=") {
		val := strings.TrimPrefix(q, "itag=")
		if it, err := strconv.Atoi(val); err == nil {
			for i := range filtered {
				if itagEquals(filtered[i], it) {
					return &filtered[i]
				}
			}
		}
	}

	// height constraints
	var minH, maxH int
	if strings.HasPrefix(q, "height<=") {
		if v, err := strconv.Atoi(strings.TrimPrefix(q, "height<=")); err == nil {
			maxH = v
		}
	}
	if strings.HasPrefix(q, "height>=") {
		if v, err := strconv.Atoi(strings.TrimPrefix(q, "height>=")); err == nil {
			minH = v
		}
	}
	if minH > 0 || maxH > 0 {
		tmp := filtered[:0]
		for i := range filtered {
			if withinHeight(filtered[i], minH, maxH) {
				tmp = append(tmp, filtered[i])
			}
		}
		if len(tmp) > 0 {
			filtered = tmp
		}
	}

	// best/worst using height then bitrate
	if q == "best" || q == "worst" {
		best := filtered[0]
		for _, f := range filtered[1:] {
			if betterByHeightThenBitrate(f, best) {
				best = f
			}
		}
		if q == "best" {
			return &best
		}
		// worst: pick opposite
		worst := filtered[0]
		for _, f := range filtered[1:] {
			if betterByHeightThenBitrate(worst, f) {
				worst = f
			}
		}
		return &worst
	}

	// Backward compatibility: itag 22 -> 18
	var itag22, itag18 *types.Format
	for i := range filtered {
		if filtered[i].Itag == 22 {
			iTAG22 := filtered[i]
			itag22 = &iTAG22
		}
		if filtered[i].Itag == 18 {
			iTAG18 := filtered[i]
			itag18 = &iTAG18
		}
	}
	if itag22 != nil {
		return itag22
	}
	if itag18 != nil {
		return itag18
	}

	// progressive mp4 with avc1 preference
	for i := range filtered {
		if strings.Contains(filtered[i].MimeType, "video/mp4") && strings.Contains(filtered[i].MimeType, "avc1") {
			return &filtered[i]
		}
	}
	// prefer any with direct URL
	for i := range filtered {
		if hasDirectURL(filtered[i]) {
			return &filtered[i]
		}
	}
	// fallback
	return &filtered[0]
}

// ResolveFormatURL builds the final downloadable URL for a selected format.
// If URL is present, optionally decodes 'n'. If signatureCipher is present, deciphers 's' and builds URL.
func ResolveFormatURL(httpClient *http.Client, f types.Format, playerJSURL string) (string, error) {
	if strings.TrimSpace(f.URL) != "" {
		u, err := url.Parse(f.URL)
		if err != nil {
			return "", fmt.Errorf("parse direct url failed: %v", err)
		}
		q := u.Query()
		if nval := q.Get("n"); nval != "" {
			if nout, err := cipher.DecipherN(httpClient, playerJSURL, nval); err == nil && nout != "" {
				q.Set("n", nout)
				u.RawQuery = q.Encode()
			}
		}
		// Ensure ratebypass for ranged requests
		if q.Get("ratebypass") == "" {
			q.Set("ratebypass", "yes")
			u.RawQuery = q.Encode()
		}
		// Encourage redirect behavior to non-alt hosts
		if q.Get("alr") == "" {
			q.Set("alr", "yes")
			u.RawQuery = q.Encode()
		}
		return u.String(), nil
	}
	if strings.TrimSpace(f.SignatureCipher) == "" {
		return "", fmt.Errorf("no url or signatureCipher for selected format")
	}
	parsed, err := url.ParseQuery(f.SignatureCipher)
	if err != nil {
		return "", fmt.Errorf("parse signatureCipher failed: %v", err)
	}
	sig := parsed.Get("s")
	sp := parsed.Get("sp")
	if sp == "" {
		sp = "signature"
	}
	cipherURL := parsed.Get("url")
	if cipherURL == "" || sig == "" {
		return "", fmt.Errorf("signatureCipher missing signature or url")
	}
	decodedSig, err := cipher.Decipher(httpClient, playerJSURL, sig)
	if err != nil {
		return "", fmt.Errorf("decipher signature failed: %v", err)
	}
	u, err := url.Parse(cipherURL)
	if err != nil {
		return "", fmt.Errorf("parse cipher url failed: %v", err)
	}
	q := u.Query()
	q.Set(sp, decodedSig)
	if nval := q.Get("n"); nval != "" {
		if nout, err := cipher.DecipherN(httpClient, playerJSURL, nval); err == nil && nout != "" {
			q.Set("n", nout)
		}
	}
	if q.Get("ratebypass") == "" {
		q.Set("ratebypass", "yes")
	}
	if q.Get("alr") == "" {
		q.Set("alr", "yes")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
