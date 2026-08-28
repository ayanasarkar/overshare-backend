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

	isVoiceFix := len(req.MuteSegments) > 0 || req.AnonymizeVoice
	isImageFix := req.StripMetadata || len(req.BlurRegions) > 0

	if !isVoiceFix && !isImageFix {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fix action requested"})
		return
	}
	if isVoiceFix && isImageFix {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot mix image and audio fix actions in one request"})
		return
	}

	if err := os.MkdirAll(fixedDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare fixed dir"})
		return
	}

	srcPath := rec.OriginalImagePath
	dstPath := filepath.Join(fixedDir, req.ScanID+filepath.Ext(srcPath))

	if isVoiceFix {
		fixedBytes, err := callVoiceFix(srcPath, req.MuteSegments, req.AnonymizeVoice)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("voice fix failed: %v", err)})
			return
		}
		if err := os.WriteFile(dstPath, fixedBytes, 0o644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not write fixed audio"})
			return
		}
	} else {
		if req.StripMetadata {
			if err := imaging.StripMetadata(srcPath, dstPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("strip metadata failed: %v", err)})
				return
			}
			srcPath = dstPath
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
	}

	fixedHash, err := hashFile(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash fixed file"})
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

func callVoiceFix(audioPath string, muteSegments []models.TimeSpan, anonymize bool) ([]byte, error) {
	return nil, fmt.Errorf("voice fix integration not yet wired up (target: %s/voice-fix)", aiServiceBaseURL)
}
