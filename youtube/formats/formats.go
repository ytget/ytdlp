package formats

import (
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
			itag = int(v)
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
	for i := range formats {
		if formats[i].URL != "" {
			continue
		}
		if formats[i].SignatureCipher == "" {
			continue
		}

		parsedCipher, err := url.ParseQuery(formats[i].SignatureCipher)
		if err != nil {
			return err
		}

		sig := parsedCipher.Get("s")
		sp := parsedCipher.Get("sp")
		if sp == "" {
			sp = "signature"
		}
		cipherURL := parsedCipher.Get("url")
		if cipherURL == "" || sig == "" {
			continue
		}

		decipheredSig, err := cipher.Decipher(httpClient, playerJSURL, sig)
		if err != nil {
			return err
		}

		finalURL, err := url.Parse(cipherURL)
		if err != nil {
			return err
		}

		query := finalURL.Query()
		query.Set(sp, decipheredSig)
		// Apply n-parameter decoding if present
		if nval := query.Get("n"); nval != "" {
			if nOut, err := cipher.DecipherN(httpClient, playerJSURL, nval); err == nil && nOut != "" {
				query.Set("n", nOut)
			}
		}
		finalURL.RawQuery = query.Encode()

		formats[i].URL = finalURL.String()
	}
	return nil
}

// SelectFormat chooses the best format according to criteria.
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
// then first progressive MP4 with avc1 codec.
func SelectFormat(formats []types.Format, quality, ext string) *types.Format {
	// 1) keep only direct-URL formats
	direct := make([]types.Format, 0, len(formats))
	for i := range formats {
		if hasDirectURL(formats[i]) {
			direct = append(direct, formats[i])
		}
	}
	if len(direct) == 0 {
		return nil
	}

	// 2) filter by extension if provided
	filtered := make([]types.Format, 0, len(direct))
	for i := range direct {
		if mimeSubtypeEquals(direct[i], ext) {
			filtered = append(filtered, direct[i])
		}
	}
	if len(filtered) == 0 {
		filtered = direct
	}

	// 3) explicit itag selector
	q := strings.TrimSpace(strings.ToLower(quality))
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

	// 4) height constraints
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

	// 5) best/worst using height then bitrate
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

	// 6) Backward compatibility: itag 22 -> 18
	var itag22, itag18 *types.Format
	for i := range filtered {
		if filtered[i].Itag == 22 {
			itag22 = &filtered[i]
		}
		if filtered[i].Itag == 18 {
			itag18 = &filtered[i]
		}
	}
	if itag22 != nil {
		return itag22
	}
	if itag18 != nil {
		return itag18
	}

	// 7) progressive mp4 with avc1 preference
	for i := range filtered {
		if strings.Contains(filtered[i].MimeType, "video/mp4") && strings.Contains(filtered[i].MimeType, "avc1") {
			return &filtered[i]
		}
	}
	// 8) fallback
	return &filtered[0]
}
