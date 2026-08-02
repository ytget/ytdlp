package types

// CaptionTrack describes an available caption/subtitle track for a video.
type CaptionTrack struct {
	// LanguageCode is a BCP-47-ish code, e.g. "en", "en-US", "ru".
	LanguageCode string `json:"language_code"`
	// LanguageName is a human-readable label when available.
	LanguageName string `json:"language_name,omitempty"`
	// Kind is empty for manual captions and typically "asr" for auto-generated.
	Kind string `json:"kind,omitempty"`
	// BaseURL is the timedtext endpoint for this track.
	BaseURL string `json:"base_url"`
	// IsAuto is true when the track is auto-generated (ASR).
	IsAuto bool `json:"is_auto"`
	// IsTranslatable indicates the track can be translated by YouTube.
	IsTranslatable bool `json:"is_translatable,omitempty"`
	// VssID is YouTube's internal track id when present.
	VssID string `json:"vss_id,omitempty"`
}

// CaptionCue is a single timed caption segment.
type CaptionCue struct {
	// StartMs is the cue start offset in milliseconds.
	StartMs int64 `json:"start_ms"`
	// DurationMs is the cue duration in milliseconds.
	DurationMs int64 `json:"duration_ms"`
	// Text is the caption text (plain, newlines allowed).
	Text string `json:"text"`
}

// EndMs returns the exclusive end timestamp in milliseconds.
func (c CaptionCue) EndMs() int64 {
	return c.StartMs + c.DurationMs
}

// Captions is a downloaded caption track with normalized cues.
type Captions struct {
	LanguageCode string       `json:"language_code"`
	LanguageName string       `json:"language_name,omitempty"`
	IsAuto       bool         `json:"is_auto"`
	Cues         []CaptionCue `json:"cues"`
}
