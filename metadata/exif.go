// Package metadata reads EXIF GPS data straight off the uploaded image, no
// dependency on Wrik's AI service. Per A2, get this working Day 1 — it's the
// safety net if his service is running late.
package metadata

import (
	"fmt"
	"os"

	"github.com/rwcarlsen/goexif/exif"
)

// GPSData mirrors models.GPSData (kept separate so this package has zero
// dependency on models — handlers/scan.go does the copy into the response).
type GPSData struct {
	Latitude  float64
	Longitude float64
	Timestamp string
}

// ExtractGPSMetadata reads EXIF GPS tags from an image file on disk.
// Returns (nil, nil) — not an error — when the image simply has no GPS data,
// since that's the common case (screenshots, downloaded images, etc.) and
// callers shouldn't treat "no GPS" as a failure.
func ExtractGPSMetadata(imagePath string) (*GPSData, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		// No EXIF segment at all — not an error condition for this feature.
		return nil, nil
	}

	lat, lon, err := x.LatLong()
	if err != nil {
		// EXIF present but no GPS tags — also not an error condition.
		return nil, nil
	}

	data := &GPSData{
		Latitude:  lat,
		Longitude: lon,
	}

	if tm, err := x.DateTime(); err == nil {
		data.Timestamp = tm.Format("2006-01-02T15:04:05Z07:00")
	}

	return data, nil
}
