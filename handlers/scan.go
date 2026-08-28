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

// precomputedCache is loaded once at startup by LoadPrecomputedCache, called
// from main.go before the server starts accepting requests.
var precomputedCache map[string]models.ScanResponse

// LoadPrecomputedCache reads demo-assets/precomputed_scans.json into memory,
// keyed by image sha256. Must succeed before the server starts — if this
// file is missing or malformed, the judged demo path has nothing to serve.
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

// ScanHandler implements POST /api/scan (multipart form, field name "image").
//
// Cache-first, per A3 and the No-Lag Checklist's first item: hash the
// upload and check precomputed_scans.json before touching GPS extraction or
// the AI service at all. The judged demo path never gets past step 1.
func ScanHandler(c *gin.Context) {
fileHeader, err := c.FormFile("image")
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "missing image file (expected multipart field \"image\")"})
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

imageHash, err := hashFile(savedPath)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash image"})
return
}

// 1. Cache-first check — judged demo path stops here.
if cached, ok := precomputedCache[imageHash]; ok {
    cached.ScanID = imageHash
    cached.ImageHash = imageHash
    cached.Source = "precomputed"

    rec := &store.ScanRecord{OriginalImagePath: savedPath}
    if existing, ok := store.Get(imageHash); ok {
        rec.FixedImagePath = existing.FixedImagePath // preserve prior fix
    }
    rec.Response = cached
    store.Save(imageHash, rec)

    c.JSON(http.StatusOK, cached)
    return
}
// 2. Cache miss — bonus, post-pitch live-scan path only.
gps, err := metadata.ExtractGPSMetadata(savedPath)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "gps extraction failed"})
return
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
c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("ai service call failed: %v", err)})
return
}

response := models.ScanResponse{
ScanID:        imageHash,
ImageHash:     imageHash,
GPS:           gpsOut,
TextRegions:   textRegions,
Objects:       objects,
DocumentFlags: docFlags,
Explanation:   explanation,
Source:        "live",
}

store.Save(imageHash, &store.ScanRecord{
OriginalImagePath: savedPath,
Response:          response,
})

c.JSON(http.StatusOK, response)
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

// callAIService is a scaffold stub. Wrik's ai_service isn't up yet, so this
// returns an error instead of fabricating results. Wire this up to
// POST {aiServiceBaseURL}/ocr, /detect, /document-flags, /explain (all four,
// per C1) once his service is live. Keeping this as one function means
// ScanHandler's shape doesn't change when that wiring lands.
func callAIService(imagePath string) ([]models.TextRegion, []models.DetectedObject, []models.DocumentFlag, string, error) {
return nil, nil, nil, "", fmt.Errorf("ai service integration not yet wired up (target: %s)", aiServiceBaseURL)
}
