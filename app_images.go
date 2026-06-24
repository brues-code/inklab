package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ImageResult represents the result of an image fetch
type ImageResult struct {
	Data     string `json:"data"`     // Base64 encoded image data
	MimeType string `json:"mimeType"` // MIME type (image/jpeg, image/png, etc.)
	Source   string `json:"source"`   // "local" or "remote"
	Error    string `json:"error"`    // Error message if any
}

// GetLocalImage reads a local image file and returns it as base64
// imageType: "icon", "npc_model", "npc_map"
// name: file name (e.g., "inv_sword_01" for icons, "model_15114" for npc)
func (a *App) GetLocalImage(imageType string, name string) *ImageResult {
	var basePath string
	var extensions []string

	switch imageType {
	case "icon":
		basePath = filepath.Join(a.DataDir, "icons")
		extensions = []string{".png", ".jpg", ".jpeg"}
	case "npc_model", "npc_map":
		basePath = filepath.Join(a.DataDir, "npc_images")
		extensions = []string{".jpg", ".png", ".jpeg"}
	case "zone_map":
		// Locally-generated zone maps keyed by texture-folder name (= zone name).
		basePath = filepath.Join(a.DataDir, "maps")
		extensions = []string{".jpg", ".png"}
	default:
		return &ImageResult{Error: "unknown image type: " + imageType}
	}

	// Try each extension
	for _, ext := range extensions {
		filePath := filepath.Join(basePath, name+ext)
		if data, err := os.ReadFile(filePath); err == nil {
			mimeType := "image/jpeg"
			if ext == ".png" {
				mimeType = "image/png"
			}
			return &ImageResult{
				Data:     base64.StdEncoding.EncodeToString(data),
				MimeType: mimeType,
				Source:   "local",
			}
		}
	}

	return &ImageResult{Error: "file not found: " + name}
}

// FetchRemoteImage fetches an image from a remote URL and returns it as base64
// Also optionally saves it locally for caching
func (a *App) FetchRemoteImage(url string, imageType string, name string) *ImageResult {
	if url == "" {
		return &ImageResult{Error: "empty URL"}
	}

	// Create HTTP request with User-Agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &ImageResult{Error: "failed to create request: " + err.Error()}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &ImageResult{Error: "failed to fetch: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &ImageResult{Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ImageResult{Error: "failed to read response: " + err.Error()}
	}

	// Determine MIME type
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	// Optionally save locally
	if imageType != "" && name != "" {
		var saveDir string
		switch imageType {
		case "icon":
			saveDir = filepath.Join(a.DataDir, "icons")
		case "npc_model", "npc_map":
			saveDir = filepath.Join(a.DataDir, "npc_images")
		}
		if saveDir != "" {
			os.MkdirAll(saveDir, 0755)
			ext := ".jpg"
			if strings.Contains(mimeType, "png") {
				ext = ".png"
			}
			savePath := filepath.Join(saveDir, name+ext)
			os.WriteFile(savePath, data, 0644)
			fmt.Printf("[Image] Saved to local: %s\n", savePath)
		}
	}

	return &ImageResult{
		Data:     base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
		Source:   "remote",
	}
}
