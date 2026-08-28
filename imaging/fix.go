// Package imaging wraps github.com/disintegration/imaging for the two fix
// actions Overshare offers: stripping metadata and blurring a region. Pure
// Go, no cgo/OpenCV risk, per A4.
package imaging

import (
	"fmt"
	"image"

	"github.com/disintegration/imaging"
)

// Region is a pixel-space rectangle to blur — same shape as models.BBox,
// kept separate so this package doesn't depend on models.
type Region struct {
	X      int
	Y      int
	Width  int
	Height int
}

// StripMetadata re-decodes and re-encodes the image. disintegration/imaging
// doesn't carry EXIF through on save, which is the simplest reliable way to
// drop GPS/camera metadata without hand-rolling EXIF-segment surgery.
func StripMetadata(srcPath, dstPath string) error {
	img, err := imaging.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open image: %w", err)
	}

	if err := imaging.Save(img, dstPath); err != nil {
		return fmt.Errorf("save stripped image: %w", err)
	}

	return nil
}

// BlurRegion applies a strong gaussian blur to one rectangular region (e.g. a
// face, ID card, or license plate flagged by the AI service) and leaves the
// rest of the image untouched.
func BlurRegion(srcPath, dstPath string, region Region) error {
	img, err := imaging.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open image: %w", err)
	}

	bounds := img.Bounds()
	rect := image.Rect(region.X, region.Y, region.X+region.Width, region.Y+region.Height).Intersect(bounds)
	if rect.Empty() {
		return fmt.Errorf("blur region (%d,%d,%d,%d) is outside image bounds", region.X, region.Y, region.Width, region.Height)
	}

	cropped := imaging.Crop(img, rect)
	blurred := imaging.Blur(cropped, 12.0)

	out := imaging.Clone(img)
	out = imaging.Paste(out, blurred, rect.Min)

	if err := imaging.Save(out, dstPath); err != nil {
		return fmt.Errorf("save blurred image: %w", err)
	}

	return nil
}
