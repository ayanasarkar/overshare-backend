// Package models holds the response/request shapes shared across the Go backend,
// Wrik's AI service (:8001), and his React frontend. Keep this in sync with
// overshareApi.js on his side — freeze this before he starts building against it
// (see Sync Points, Day 1).
package models

// BBox is a pixel-space bounding box, origin top-left, matching how the AI
// service (YOLO / OCR) and the frontend canvas overlay both expect boxes.
type BBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GPSData is what ExtractGPSMetadata (metadata/exif.go) returns, and what
// ends up in ScanResponse.GPS. Nil/omitted when the image has no GPS EXIF.
type GPSData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
}

// TextRegion is one OCR hit from Wrik's /ocr route (or the precomputed cache).
type TextRegion struct {
	Text       string  `json:"text"`
	Box        BBox    `json:"box"`
	Confidence float64 `json:"confidence"`
	// Category is a soft label like "phone", "email", "address" — set by his
	// OCR post-processing, not guaranteed for every hit.
	Category string `json:"category,omitempty"`
}

// DetectedObject is one YOLOv8n hit from Wrik's /detect route.
type DetectedObject struct {
	Label      string  `json:"label"`
	Box        BBox    `json:"box"`
	Confidence float64 `json:"confidence"`
}

// DocumentFlag marks a region flagged as e.g. an ID card, passport, or credit
// card by /document-flags — the "fix" side treats these as blur candidates.
type DocumentFlag struct {
	Type       string  `json:"type"`
	Box        BBox    `json:"box"`
	Confidence float64 `json:"confidence"`
}

// Flag is a shared "something to review" hit for the voice/audio path —
// Voice-Feature-Ayana.md §2. Bbox is unused here (images keep their existing
// TextRegion/DetectedObject/DocumentFlag types, untouched); TimeRange is
// populated for audio flags like spoken PII or voice-GPS mentions.
type Flag struct {
	Type        string    `json:"type"`
	ObjectClass string    `json:"object_class,omitempty"`
	MatchedText string    `json:"matched_text,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
	Bbox        []int     `json:"bbox,omitempty"`
	TimeRange   *TimeSpan `json:"time_range,omitempty"`
}

// TimeSpan is a start/end range in seconds — used by Flag.TimeRange and by
// FixRequest.MuteSegments.
type TimeSpan struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// TranscriptSegment is one chunk of speech-to-text output from Wrik's
// /transcribe route (voice scan pipeline only).
type TranscriptSegment struct {
	Text       string  `json:"text"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence"`
}

// ScanResponse is the full payload for POST /api/scan. Source is either
// "precomputed" (judged demo path, from demo-assets/precomputed_scans.json)
// or "live" (bonus post-pitch path, hits Wrik's AI service).
type ScanResponse struct {
	ScanID        string              `json:"scan_id"`
	ImageHash     string              `json:"image_hash"`
	MediaType     string              `json:"media_type,omitempty"`
	GPS           *GPSData            `json:"gps,omitempty"`
	TextRegions   []TextRegion        `json:"text_regions"`
	Objects       []DetectedObject    `json:"objects"`
	DocumentFlags []DocumentFlag      `json:"document_flags"`
	Transcript    []TranscriptSegment `json:"transcript,omitempty"`
	Flags         []Flag              `json:"flags,omitempty"`
	Explanation   string              `json:"explanation"`
	FaceMatch     *bool               `json:"face_match,omitempty"`
	Source        string              `json:"source"`
}

// FixRequest is the body for POST /api/fix. BlurRegions are applied in order,
// after StripMetadata if both are requested. MuteSegments/AnonymizeVoice are
// audio-only (mutually exclusive with the image fields — enforced in
// handlers/fix.go).
type FixRequest struct {
	ScanID         string     `json:"scan_id"`
	StripMetadata  bool       `json:"strip_metadata"`
	BlurRegions    []BBox     `json:"blur_regions,omitempty"`
	MuteSegments   []TimeSpan `json:"mute_segments,omitempty"`
	AnonymizeVoice bool       `json:"anonymize_voice,omitempty"`
}

// FixResponse is the result of POST /api/fix.
type FixResponse struct {
	ScanID         string `json:"scan_id"`
	FixedImagePath string `json:"fixed_image_path"`
	FixedImageHash string `json:"fixed_image_hash"`
}

// CertifyRequest is the body for POST /api/certify. Fix 10: scan_id must
// point at an image that has already been through /api/fix — enforced in
// handlers/certify.go, not just assumed here.
type CertifyRequest struct {
	ScanID string `json:"scan_id"`
}

// CertifyResponse is the result of a successful mint.
type CertifyResponse struct {
	CertID    uint64 `json:"cert_id"`
	TxHash    string `json:"tx_hash"`
	IPFSCid   string `json:"ipfs_cid"`
	ImageHash string `json:"image_hash"`
}
