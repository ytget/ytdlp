package captions

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ytget/ytdlp/v2/types"
	"github.com/ytget/ytdlp/v2/youtube/innertube"
)

// ParseTracks extracts caption tracks from a player response.
func ParseTracks(pr *innertube.PlayerResponse) []types.CaptionTrack {
	if pr == nil {
		return nil
	}
	raw := pr.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
	if len(raw) == 0 {
		return nil
	}

	out := make([]types.CaptionTrack, 0, len(raw))
	for _, t := range raw {
		baseURL := strings.TrimSpace(t.BaseURL)
		if baseURL == "" {
			continue
		}
		// JSON may escape & as \u0026; keep as-is (Go json.Unmarshal already decodes).
		lang := strings.TrimSpace(t.LanguageCode)
		name := t.Name.Text()
		kind := strings.TrimSpace(t.Kind)
		isAuto := strings.EqualFold(kind, "asr") || strings.HasPrefix(strings.ToLower(t.VssID), "a.")
		out = append(out, types.CaptionTrack{
			LanguageCode:   lang,
			LanguageName:   name,
			Kind:           kind,
			BaseURL:        baseURL,
			IsAuto:         isAuto,
			IsTranslatable: t.IsTranslatable,
			VssID:          t.VssID,
		})
	}
	return out
}

// SelectTrack picks the best caption track for preferredLang ("en", "ru", ...).
// Preference: manual matching lang → auto matching lang → first manual → first any.
// preferredLang may be empty; then first manual, else first track.
func SelectTrack(tracks []types.CaptionTrack, preferredLang string) (types.CaptionTrack, bool) {
	if len(tracks) == 0 {
		return types.CaptionTrack{}, false
	}
	pref := normalizeLang(preferredLang)

	if pref != "" {
		if t, ok := findTrack(tracks, pref, false); ok {
			return t, true
		}
		if t, ok := findTrack(tracks, pref, true); ok {
			return t, true
		}
	}
	for _, t := range tracks {
		if !t.IsAuto {
			return t, true
		}
	}
	return tracks[0], true
}

func findTrack(tracks []types.CaptionTrack, pref string, auto bool) (types.CaptionTrack, bool) {
	for _, t := range tracks {
		if t.IsAuto != auto {
			continue
		}
		code := normalizeLang(t.LanguageCode)
		if code == pref || strings.HasPrefix(code, pref+"-") || strings.HasPrefix(pref, code+"-") {
			return t, true
		}
	}
	for _, t := range tracks {
		if t.IsAuto != auto {
			continue
		}
		name := strings.ToLower(t.LanguageName)
		if name != "" && (strings.Contains(name, pref) || langNameMatches(name, pref)) {
			return t, true
		}
	}
	return types.CaptionTrack{}, false
}

func normalizeLang(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "system" || s == "auto" {
		return ""
	}
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func langNameMatches(name, pref string) bool {
	switch pref {
	case "en":
		return strings.Contains(name, "english")
	case "ru":
		return strings.Contains(name, "russian") || strings.Contains(name, "рус")
	case "pt":
		return strings.Contains(name, "portuguese") || strings.Contains(name, "português") || strings.Contains(name, "portugues")
	default:
		return false
	}
}

// Download fetches and parses captions for a track.
// Prefers srv1 (classic transcript XML), then json3, then the track's default URL.
func Download(ctx context.Context, httpClient *http.Client, track types.CaptionTrack) (*types.Captions, error) {
	if strings.TrimSpace(track.BaseURL) == "" {
		return nil, fmt.Errorf("captions: empty base url")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	var lastErr error
	// srv1 is the most reliable/simple structure for conversion to SRT/TXT.
	for _, format := range []string{"srv1", "json3", "", "srv3", "vtt"} {
		endpoint := track.BaseURL
		if format != "" {
			u, err := withFmt(track.BaseURL, format)
			if err != nil {
				lastErr = err
				continue
			}
			endpoint = u
		}
		body, err := getBody(ctx, httpClient, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		cues, err := decodeCaptionBody(body, format)
		if err != nil {
			lastErr = err
			continue
		}
		if len(cues) == 0 {
			lastErr = fmt.Errorf("captions: no cues in %s response", formatOrDefault(format))
			continue
		}
		return &types.Captions{
			LanguageCode: track.LanguageCode,
			LanguageName: track.LanguageName,
			IsAuto:       track.IsAuto,
			Cues:         cues,
		}, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("captions: failed to download/parse track")
}

func formatOrDefault(format string) string {
	if format == "" {
		return "default"
	}
	return format
}

func withFmt(rawURL, format string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("fmt", format)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func decodeCaptionBody(body []byte, format string) ([]types.CaptionCue, error) {
	trim := strings.TrimSpace(string(body))
	if trim == "" {
		return nil, fmt.Errorf("captions: empty body")
	}

	switch format {
	case "json3":
		return parseJSON3(body)
	case "vtt":
		return parseVTT(trim)
	case "srv1", "":
		// try classic transcript first, then generic xml / json3 / vtt
		if cues, err := parseXMLCaptions(body); err == nil && len(cues) > 0 {
			return cues, nil
		}
		if strings.HasPrefix(trim, "{") {
			return parseJSON3(body)
		}
		if strings.HasPrefix(trim, "WEBVTT") {
			return parseVTT(trim)
		}
		return parseXMLCaptions(body)
	default:
		if cues, err := parseXMLCaptions(body); err == nil && len(cues) > 0 {
			return cues, nil
		}
		if strings.HasPrefix(trim, "{") {
			return parseJSON3(body)
		}
		if strings.HasPrefix(trim, "WEBVTT") {
			return parseVTT(trim)
		}
		return parseXMLCaptions(body)
	}
}

func parseJSON3(body []byte) ([]types.CaptionCue, error) {
	var payload json3Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("captions: parse json3: %w", err)
	}
	cues := make([]types.CaptionCue, 0, len(payload.Events))
	for _, ev := range payload.Events {
		if len(ev.Segs) == 0 {
			continue
		}
		var b strings.Builder
		for _, seg := range ev.Segs {
			b.WriteString(seg.UTF8)
		}
		text := normalizeCaptionText(b.String())
		if text == "" {
			continue
		}
		start := ev.TStartMs
		dur := ev.DDurationMs
		if dur <= 0 {
			dur = 2000
		}
		cues = append(cues, types.CaptionCue{
			StartMs:    start,
			DurationMs: dur,
			Text:       text,
		})
	}
	return cues, nil
}

type json3Payload struct {
	Events []json3Event `json:"events"`
}

type json3Event struct {
	TStartMs    int64      `json:"tStartMs"`
	DDurationMs int64      `json:"dDurationMs"`
	Segs        []json3Seg `json:"segs"`
}

type json3Seg struct {
	UTF8 string `json:"utf8"`
}

type timedTextXML struct {
	XMLName xml.Name `xml:"timedtext"`
	Body    struct {
		Paragraphs []struct {
			TimeMs string `xml:"t,attr"`
			DurMs  string `xml:"d,attr"`
			Text   string `xml:",chardata"`
			Spans  []struct {
				Text string `xml:",chardata"`
			} `xml:"s"`
		} `xml:"p"`
		Texts []struct {
			Start string `xml:"start,attr"`
			Dur   string `xml:"dur,attr"`
			Body  string `xml:",chardata"`
		} `xml:"text"`
	} `xml:"body"`
	Texts []struct {
		Start string `xml:"start,attr"`
		Dur   string `xml:"dur,attr"`
		Body  string `xml:",chardata"`
	} `xml:"text"`
}

func parseXMLCaptions(body []byte) ([]types.CaptionCue, error) {
	// classic transcript format
	type classic struct {
		Texts []struct {
			Start string `xml:"start,attr"`
			Dur   string `xml:"dur,attr"`
			Body  string `xml:",chardata"`
		} `xml:"text"`
	}
	var c classic
	if err := xml.Unmarshal(body, &c); err == nil && len(c.Texts) > 0 {
		out := make([]types.CaptionCue, 0, len(c.Texts))
		for _, t := range c.Texts {
			text := normalizeCaptionText(t.Body)
			if text == "" {
				continue
			}
			startMs := secStringToMs(t.Start)
			durMs := secStringToMs(t.Dur)
			if durMs <= 0 {
				durMs = 2000
			}
			out = append(out, types.CaptionCue{StartMs: startMs, DurationMs: durMs, Text: text})
		}
		if len(out) > 0 {
			return out, nil
		}
	}

	var tt timedTextXML
	if err := xml.Unmarshal(body, &tt); err != nil {
		return nil, fmt.Errorf("captions: parse xml: %w", err)
	}
	out := make([]types.CaptionCue, 0)
	for _, p := range tt.Body.Paragraphs {
		var text string
		if len(p.Spans) > 0 {
			var b strings.Builder
			for _, s := range p.Spans {
				b.WriteString(s.Text)
			}
			text = b.String()
		} else {
			text = p.Text
		}
		text = normalizeCaptionText(text)
		if text == "" {
			continue
		}
		startMs, _ := strconv.ParseInt(strings.TrimSpace(p.TimeMs), 10, 64)
		durMs, _ := strconv.ParseInt(strings.TrimSpace(p.DurMs), 10, 64)
		if durMs <= 0 {
			durMs = 2000
		}
		out = append(out, types.CaptionCue{StartMs: startMs, DurationMs: durMs, Text: text})
	}
	for _, t := range tt.Body.Texts {
		text := normalizeCaptionText(t.Body)
		if text == "" {
			continue
		}
		startMs := secStringToMs(t.Start)
		durMs := secStringToMs(t.Dur)
		if durMs <= 0 {
			durMs = 2000
		}
		out = append(out, types.CaptionCue{StartMs: startMs, DurationMs: durMs, Text: text})
	}
	for _, t := range tt.Texts {
		text := normalizeCaptionText(t.Body)
		if text == "" {
			continue
		}
		startMs := secStringToMs(t.Start)
		durMs := secStringToMs(t.Dur)
		if durMs <= 0 {
			durMs = 2000
		}
		out = append(out, types.CaptionCue{StartMs: startMs, DurationMs: durMs, Text: text})
	}
	return out, nil
}

func parseVTT(s string) ([]types.CaptionCue, error) {
	lines := strings.Split(s, "\n")
	var out []types.CaptionCue
	// timestamp line: 00:00:00.000 --> 00:00:01.750
	tsRe := regexp.MustCompile(`(?i)(\d{2}:\d{2}:\d{2}\.\d{3}|\d{2}:\d{2}\.\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}\.\d{3}|\d{2}:\d{2}\.\d{3})`)
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		m := tsRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		startMs := vttTSToMs(m[1])
		endMs := vttTSToMs(m[2])
		var textParts []string
		for j := i + 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if t == "" {
				i = j
				break
			}
			if tsRe.MatchString(t) {
				i = j - 1
				break
			}
			// strip cue settings / tags
			t = tagRe.ReplaceAllString(t, "")
			t = normalizeCaptionText(t)
			if t != "" {
				textParts = append(textParts, t)
			}
			i = j
		}
		text := strings.Join(textParts, " ")
		if text == "" {
			continue
		}
		dur := endMs - startMs
		if dur <= 0 {
			dur = 1000
		}
		out = append(out, types.CaptionCue{StartMs: startMs, DurationMs: dur, Text: text})
	}
	return out, nil
}

func vttTSToMs(ts string) int64 {
	ts = strings.TrimSpace(ts)
	parts := strings.Split(ts, ":")
	var h, m int64
	var secPart string
	switch len(parts) {
	case 3:
		h, _ = strconv.ParseInt(parts[0], 10, 64)
		m, _ = strconv.ParseInt(parts[1], 10, 64)
		secPart = parts[2]
	case 2:
		m, _ = strconv.ParseInt(parts[0], 10, 64)
		secPart = parts[1]
	default:
		return 0
	}
	secParts := strings.SplitN(secPart, ".", 2)
	s, _ := strconv.ParseInt(secParts[0], 10, 64)
	ms := int64(0)
	if len(secParts) == 2 {
		frac := secParts[1]
		for len(frac) < 3 {
			frac += "0"
		}
		if len(frac) > 3 {
			frac = frac[:3]
		}
		ms, _ = strconv.ParseInt(frac, 10, 64)
	}
	return ((h*60+m)*60+s)*1000 + ms
}

func secStringToMs(v string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if strings.Contains(v, ".") {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return int64(f * 1000.0)
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	// bare integers in classic timedtext start/dur are seconds when small, but srv attrs are ms.
	// classic uses floats almost always. Treat plain int as milliseconds if large, else seconds.
	if n >= 100000 {
		return n
	}
	// heuristic: values like "4" mean 4 seconds in classic? rare. Prefer ms for srv body attrs.
	// For classic start without decimal, treat as seconds.
	return n * 1000
}

var tagRe = regexp.MustCompile(`<[^>]+>`)
var numEntityRe = regexp.MustCompile(`&#(\d+);`)
var hexEntityRe = regexp.MustCompile(`&#x([0-9a-fA-F]+);`)

func normalizeCaptionText(s string) string {
	// YouTube often double-encodes: &amp;#39; -> &#39; -> '
	for i := 0; i < 3; i++ {
		next := html.UnescapeString(s)
		if next == s {
			break
		}
		s = next
	}
	s = numEntityRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := numEntityRe.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		n, err := strconv.Atoi(sub[1])
		if err != nil {
			return m
		}
		return string(rune(n))
	})
	s = hexEntityRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := hexEntityRe.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		n, err := strconv.ParseInt(sub[1], 16, 32)
		if err != nil {
			return m
		}
		return string(rune(n))
	})
	s = tagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\u200b", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func getBody(ctx context.Context, httpClient *http.Client, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	// Browser-like headers; let transport handle compression.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://www.youtube.com/")
	req.Header.Set("Origin", "https://www.youtube.com")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 403/410 often mean the signed timedtext URL expired.
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("captions: timedtext expired or forbidden (http %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("captions: http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if len(bytesTrimSpace(body)) == 0 {
		return nil, fmt.Errorf("captions: empty body")
	}
	return body, nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
