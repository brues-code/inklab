package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"inklab/backend/datatools"
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
	case "talent_bg":
		basePath = filepath.Join(a.DataDir, "talent_bg")
		extensions = []string{".png", ".jpg"}
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

// clientMPQ returns a cached MPQ source for <baseDir>/Data, opening (and
// caching) it on first use and reopening if baseDir changed. Callers MUST hold
// clientSrcMu around this and any use of the returned source — the mpq set is
// not concurrency-safe.
func (a *App) clientMPQ(baseDir string) (datatools.ClientFiles, bool) {
	dataDir := filepath.Join(baseDir, "Data")
	if a.clientSrc != nil && a.clientSrcDir == dataDir {
		return a.clientSrc, true
	}
	if a.clientSrc != nil {
		a.clientSrc.Close()
		a.clientSrc = nil
		a.clientSrcDir = ""
	}
	src, err := datatools.NewMpqSource(dataDir)
	if err != nil {
		return nil, false
	}
	a.clientSrc = src
	a.clientSrcDir = dataDir
	return src, true
}

// RenderNpcModel renders an NPC's model on demand from the client MPQs under
// baseDir, so a page visit can generate a missing render instead of relying on a
// bulk pre-pass. It writes a per-creature image (body + armor + held weapons)
// when the creature carries weapons, plus the shared display render. Returns true
// if a client was available and rendering ran; the frontend then re-reads the
// produced files via GetLocalImage (so each cache key holds its own image) and
// falls back to the remote image when this returns false or produced nothing.
func (a *App) RenderNpcModel(entry int, displayID int, baseDir string) bool {
	if displayID <= 0 || baseDir == "" {
		return false
	}
	a.clientSrcMu.Lock()
	defer a.clientSrcMu.Unlock()
	cf, ok := a.clientMPQ(baseDir)
	if !ok {
		return false
	}
	a.npcService.RenderModelOnDemand(cf, entry, displayID)
	return true
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

