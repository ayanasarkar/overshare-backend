package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"overshare-backend/imaging"
	"overshare-backend/models"
	"overshare-backend/store"
)

const fixedDir = "fixed"

// FixHandler implements POST /api/fix. Looks up the scan_id from a prior
// /api/scan call, applies strip-metadata and/or blur-region(s) in that
// order, and records the fixed image path on the scan record so /api/certify
// (Fix 10) can later confirm this scan_id has actually been fixed.
func FixHandler(c *gin.Context) {
	var req models.FixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	rec, ok := store.Get(req.ScanID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown scan_id — call /api/scan first"})
		return
	}

	if !req.StripMetadata && len(req.BlurRegions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fix action requested (set strip_metadata and/or blur_regions)"})
		return
	}

	if err := os.MkdirAll(fixedDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare fixed dir"})
		return
	}

	srcPath := rec.OriginalImagePath
	dstPath := filepath.Join(fixedDir, req.ScanID+filepath.Ext(srcPath))

	if req.StripMetadata {
		if err := imaging.StripMetadata(srcPath, dstPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("strip metadata failed: %v", err)})
			return
		}
		srcPath = dstPath // chain any blur on top of the stripped version
	}

	for _, region := range req.BlurRegions {
		if err := imaging.BlurRegion(srcPath, dstPath, imaging.Region{
			X:      region.X,
			Y:      region.Y,
			Width:  region.Width,
			Height: region.Height,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("blur region failed: %v", err)})
			return
		}
		srcPath = dstPath
	}

	fixedHash, err := hashFile(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash fixed image"})
		return
	}

	rec.FixedImagePath = dstPath
	store.Save(req.ScanID, rec)

	c.JSON(http.StatusOK, models.FixResponse{
		ScanID:         req.ScanID,
		FixedImagePath: dstPath,
		FixedImageHash: fixedHash,
	})
}
