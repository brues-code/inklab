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
		// Zone maps are keyed by texture-folder name (e.g. "UngoroCrater") but
		// the stored zone name may differ in spacing/punctuation ("Ungoro
		// Crater", "Zul'Gurub"). Match on a normalized key.
		return a.findZoneMap(name)
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

// mapKeyAliases canonicalizes misspelled map keys so a correctly-spelled zone
// name still resolves its (typo'd) shipped map file. Both the requested name
// and the on-disk filename pass through normKey, so mapping the typo form to the
// correct form makes them collide. Keys/values are normKey output (lowercase
// alphanumerics). The octo client ships these capital-city maps misspelled.
var mapKeyAliases = map[string]string{
	"ogrimmar":  "orgrimmar", // file "Ogrimmar.jpg"  -> zone "Orgrimmar"
	"darnassis": "darnassus", // file "Darnassis.jpg" -> zone "Darnassus"
}

// normKey reduces a name to lowercase alphanumerics for loose matching, so
// "Ungoro Crater" and "Zul'Gurub" match the "UngoroCrater"/"ZulGurub" folders.
// Parenthetical suffixes are dropped so scraped names like "The Deadmines
// (Dungeon)" still match the "TheDeadmines" map file. A leading "the" is
// stripped ("The Barrens" -> Barrens) and known misspellings are canonicalized
// via mapKeyAliases.
func normKey(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range strings.ToLower(s) {
		switch {
		case r == '(' || r == '[':
			depth++
		case r == ')' || r == ']':
			if depth > 0 {
				depth--
			}
		case depth == 0 && ((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')):
			b.WriteRune(r)
		}
	}
	k := strings.TrimPrefix(b.String(), "the")
	if canon, ok := mapKeyAliases[k]; ok {
		return canon
	}
	return k
}

// findZoneMap resolves a zone name to data/maps/<file>, trying an exact match
// first then a normalized scan of the maps directory.
func (a *App) findZoneMap(name string) *ImageResult {
	dir := filepath.Join(a.DataDir, "maps")
	read := func(path, ext string) *ImageResult {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		mime := "image/jpeg"
		if ext == ".png" {
			mime = "image/png"
		}
		return &ImageResult{Data: base64.StdEncoding.EncodeToString(data), MimeType: mime, Source: "local"}
	}
	for _, ext := range []string{".jpg", ".png"} {
		if r := read(filepath.Join(dir, name+ext), ext); r != nil {
			return r
		}
	}
	target := normKey(name)
	if target != "" {
		if ents, err := os.ReadDir(dir); err == nil {
			bestFn, bestExt, bestLen := "", "", 0
			for _, e := range ents {
				fn := e.Name()
				ext := filepath.Ext(fn)
				if ext != ".jpg" && ext != ".png" {
					continue
				}
				k := normKey(strings.TrimSuffix(fn, ext))
				if k == target {
					if r := read(filepath.Join(dir, fn), ext); r != nil {
						return r
					}
				}
				// Longest-prefix fallback: the stored zone name often carries
				// extra words the texture folder lacks ("Stormwind City" ->
				// Stormwind, "Elwynn Forrest" -> Elwynn). Match the map file whose
				// key is the longest prefix of the requested name.
				if len(k) >= 3 && len(k) > bestLen && strings.HasPrefix(target, k) {
					bestFn, bestExt, bestLen = fn, ext, len(k)
				}
			}
			if bestFn != "" {
				if r := read(filepath.Join(dir, bestFn), bestExt); r != nil {
					return r
				}
			}
		}
	}
	return &ImageResult{Error: "zone map not found: " + name}
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
