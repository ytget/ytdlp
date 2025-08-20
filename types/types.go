package types

// Format describes an available media format.
type Format struct {
	Itag            int
	URL             string
	Quality         string
	MimeType        string
	Bitrate         int
	Size            int64
	SignatureCipher string
}
