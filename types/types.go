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

// VideoInfo describes video information.
type VideoInfo struct {
	ID          string
	Title       string
	Description string
	Duration    int
	Uploader    string
	UploadDate  string
	ViewCount   int64
	LikeCount   int64
	Formats     []Format
}

// PlaylistInfo describes playlist information.
type PlaylistInfo struct {
	ID          string
	Title       string
	Description string
	Author      string
	VideoCount  int
	ViewCount   int64
}
