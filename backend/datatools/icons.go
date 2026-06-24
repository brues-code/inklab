package datatools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IconGenResult summarizes an icon extraction run.
type IconGenResult struct {
	Generated int      `json:"generated"`
	Skipped   int      `json:"skipped"`
	Warnings  []string `json:"warnings"`
}

// GenerateIcons decodes every BLP in iconsDir (the client's Interface\Icons)
// and writes it to outDir/<lowercase-name>.jpg, matching how the app looks
// icons up (by lowercased name). DXT1/DXT3/DXT5 are supported; the handful of
// palettized icons are skipped.
func GenerateIcons(iconsDir, outDir string, progress func(name string, i, total int)) (*IconGenResult, error) {
	ents, err := os.ReadDir(iconsDir)
	if err != nil {
		return nil, fmt.Errorf("read Icons dir: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	var blps []string
	for _, e := range ents {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".blp") {
			blps = append(blps, e.Name())
		}
	}

	res := &IconGenResult{}
	for i, fn := range blps {
		if progress != nil {
			progress(fn, i+1, len(blps))
		}
		img, err := decodeBLP2(filepath.Join(iconsDir, fn))
		if err != nil {
			res.Skipped++
			if len(res.Warnings) < 20 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", fn, err))
			}
			continue
		}
		name := strings.ToLower(strings.TrimSuffix(fn, filepath.Ext(fn)))
		if err := writeJPEG(filepath.Join(outDir, name+".jpg"), img); err != nil {
			res.Skipped++
			continue
		}
		res.Generated++
	}
	return res, nil
}
