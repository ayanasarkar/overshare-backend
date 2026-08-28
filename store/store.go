// Package store is a deliberately simple in-memory record of scans, keyed by
// scan_id (the image's sha256). It exists so /api/fix and /api/certify can
// find the original (and, once fixed, the fixed) image path for a given
// scan_id without re-uploading. Fine for a demo process; not meant to survive
// a restart.
package store

import (
	"sync"

	"overshare-backend/models"
)

// ScanRecord tracks everything handlers need for a given scan_id across the
// scan -> fix -> certify lifecycle.
type ScanRecord struct {
	OriginalImagePath string
	FixedImagePath    string // empty until /api/fix has run for this scan_id
	Response          models.ScanResponse
}

var (
	mu    sync.RWMutex
	scans = map[string]*ScanRecord{}
)

// Save stores or overwrites the record for id.
func Save(id string, rec *ScanRecord) {
	mu.Lock()
	defer mu.Unlock()
	scans[id] = rec
}

// Get looks up the record for id.
func Get(id string) (*ScanRecord, bool) {
	mu.RLock()
	defer mu.RUnlock()
	rec, ok := scans[id]
	return rec, ok
}
