package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"overshare-backend/metadata"
	"overshare-backend/models"
	"overshare-backend/store"
)

const (
	precomputedPath  = "demo-assets/precomputed_scans.json"
	aiServiceBaseURL = "http://localhost:8001"
	uploadsDir       = "uploads"
)

var precomputedCache map[string]models.ScanResponse

func LoadPrecomputedCache() error {
	data, err := os.ReadFile(precomputedPath)
	if err != nil {
		return fmt.Errorf("read precomputed cache at %s: %w", precomputedPath, err)
	}
	var cache map[string]models.ScanResponse
	if err := json.Unmarshal(data, &cache); err != nil {
		return fmt.Errorf("parse precomputed cache: %w", err)
	}
	precomputedCache = cache
	return nil
}

func ScanHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing image file (expected multipart field \"image\")"})
		return
	}

	mediaType := c.DefaultPostForm("media_type", "image")
	if mediaType != "image" && mediaType != "audio" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported media_type: %s", mediaType)})
		return
	}

	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare uploads dir"})
		return
	}

	savedPath := filepath.Join(uploadsDir, fileHeader.Filename)
	if err := c.SaveUploadedFile(fileHeader, savedPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save upload"})
		return
	}

	fileHash, err := hashFile(savedPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash file"})
		return
	}

	if cached, ok := precomputedCache[fileHash]; ok {
		cached.ScanID = fileHash
		cached.ImageHash = fileHash
		cached.Source = "precomputed"

		rec := &store.ScanRecord{OriginalImagePath: savedPath}
		if existing, ok := store.Get(fileHash); ok {
			rec.FixedImagePath = existing.FixedImagePath
		}
		rec.Response = cached
		store.Save(fileHash, rec)

		c.JSON(http.StatusOK, cached)
		return
	}

	var response models.ScanResponse
	switch mediaType {
	case "image":
		response, err = runImagePipeline(savedPath, fileHash)
	case "audio":
		response, err = runVoicePipeline(savedPath, fileHash)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported media_type: %s", mediaType)})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	store.Save(fileHash, &store.ScanRecord{OriginalImagePath: savedPath, Response: response})
	c.JSON(http.StatusOK, response)
}

func runImagePipeline(savedPath, fileHash string) (models.ScanResponse, error) {
	gps, err := metadata.ExtractGPSMetadata(savedPath)
	if err != nil {
		return models.ScanResponse{}, fmt.Errorf("gps extraction failed: %w", err)
	}

	var gpsOut *models.GPSData
	if gps != nil {
		gpsOut = &models.GPSData{
			Latitude:  gps.Latitude,
			Longitude: gps.Longitude,
			Timestamp: gps.Timestamp,
		}
	}

	textRegions, objects, docFlags, explanation, err := callAIService(savedPath)
	if err != nil {
		return models.ScanResponse{}, fmt.Errorf("ai service call failed: %w", err)
	}

	return models.ScanResponse{
		ScanID:        fileHash,
		ImageHash:     fileHash,
		MediaType:     "image",
		GPS:           gpsOut,
		TextRegions:   textRegions,
		Objects:       objects,
		DocumentFlags: docFlags,
		Explanation:   explanation,
		Source:        "live",
	}, nil
}

func runVoicePipeline(savedPath, fileHash string) (models.ScanResponse, error) {
	gps, err := callVoiceMetadata(savedPath)
	if err != nil {
		return models.ScanResponse{}, err
	}

	transcript, err := callTranscribe(savedPath)
	if err != nil {
		return models.ScanResponse{}, err
	}

	flags, err := callVoiceFlags(transcript)
	if err != nil {
		return models.ScanResponse{}, err
	}

	return models.ScanResponse{
		ScanID:     fileHash,
		ImageHash:  fileHash,
		MediaType:  "audio",
		GPS:        gps,
		Transcript: transcript,
		Flags:      flags,
		Source:     "live",
	}, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func callAIService(imagePath string) ([]models.TextRegion, []models.DetectedObject, []models.DocumentFlag, string, error) {
	return nil, nil, nil, "", fmt.Errorf("ai service integration not yet wired up (target: %s)", aiServiceBaseURL)
}

func callVoiceMetadata(audioPath string) (*models.GPSData, error) {
	return nil, fmt.Errorf("voice ai service integration not yet wired up (target: %s/voice-metadata)", aiServiceBaseURL)
}

func callTranscribe(audioPath string) ([]models.TranscriptSegment, error) {
	return nil, fmt.Errorf("voice ai service integration not yet wired up (target: %s/transcribe)", aiServiceBaseURL)
}

func callVoiceFlags(transcript []models.TranscriptSegment) ([]models.Flag, error) {
	return nil, fmt.Errorf("voice ai service integration not yet wired up (target: %s/voice-flags)", aiServiceBaseURL)
}
