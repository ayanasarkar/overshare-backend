package handlers

import (
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"overshare-backend/clients"
	"overshare-backend/models"
	"overshare-backend/store"
)

// Set by InitCertifyClients from main.go, only when certify is configured.
// Left nil otherwise — CertifyHandler checks for that and fails cleanly.
var (
	ipfsClient  *clients.IPFSClient
	chainClient *clients.ChainClient
)

// InitCertifyClients wires up the certify feature's two external clients.
// Not calling this at all is a valid configuration — it's how the core
// scan/fix path stays independently runnable (see No-Lag Checklist).
func InitCertifyClients(ipfs *clients.IPFSClient, chain *clients.ChainClient) {
	ipfsClient = ipfs
	chainClient = chain
}

// CertifyHandler implements POST /api/certify.
//
// Fix 10: gated to only accept a scan_id whose image has already gone
// through /api/fix. This is what keeps "nothing leaves your machine" honest
// for the core path — only an explicit, already-fixed image can ever reach
// IPFS/chain, and only when the user opts into certify at all.
func CertifyHandler(c *gin.Context) {
	var req models.CertifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	rec, ok := store.Get(req.ScanID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown scan_id"})
		return
	}

	if rec.FixedImagePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "scan_id has no fixed image yet — run /api/fix first (Fix 10 gate)",
		})
		return
	}

	if ipfsClient == nil || chainClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "certify is not configured on this server (CERTIFICATE_CONTRACT_ADDRESS unset) — core scan/fix is unaffected",
		})
		return
	}

	cid, err := ipfsClient.UploadFile(rec.FixedImagePath)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "ipfs upload failed: " + err.Error()})
		return
	}

	imageHash, err := hashFile(rec.FixedImagePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash fixed image"})
		return
	}

	var hashBytes [32]byte
	decoded, err := hex.DecodeString(imageHash)
	if err != nil || len(decoded) != 32 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected hash format"})
		return
	}
	copy(hashBytes[:], decoded)

	certID, txHash, err := chainClient.MintCertificate(hashBytes, cid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "mint certificate failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.CertifyResponse{
		CertID:    certID,
		TxHash:    txHash,
		IPFSCid:   cid,
		ImageHash: imageHash,
	})
}
