// Package clients holds the two "leaves the process" integrations used only
// by the opt-in certify feature: IPFS and the local chain. Nothing in the
// core scan/fix path imports this package.
package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// IPFSClient talks directly to a local Kubo node's HTTP API — no separate
// go-ipfs-api dependency needed for a single-file upload.
type IPFSClient struct {
	apiURL string // e.g. http://localhost:5001
	http   *http.Client
}

// NewIPFSClient builds a client pointed at a local Kubo API URL.
func NewIPFSClient(apiURL string) *IPFSClient {
	return &IPFSClient{
		apiURL: apiURL,
		http:   &http.Client{},
	}
}

type addResponse struct {
	Hash string `json:"Hash"`
	Name string `json:"Name"`
	Size string `json:"Size"`
}

// UploadFile pushes a local file to the Kubo node's /api/v0/add endpoint and
// returns its CID. Kubo's add endpoint expects a multipart form, not a plain
// POST body.
func (c *IPFSClient) UploadFile(path string) (cid string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy file into form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.apiURL+"/api/v0/add", body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call kubo /api/v0/add (is the local daemon running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("kubo add failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var parsed addResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode kubo response: %w", err)
	}

	return parsed.Hash, nil
}
