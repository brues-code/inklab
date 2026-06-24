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
	return GenerateIconsFrom(NewDirSourceIcons(iconsDir), outDir, progress)
}

// GenerateIconsFrom decodes every icon BLP from any ClientFiles source (loose
// folder or in-memory MPQ) into outDir/<lowercase-name>.jpg.
func GenerateIconsFrom(cf ClientFiles, outDir string, progress func(name string, i, total int)) (*IconGenResult, error) {
	names, err := cf.ListIcons()
	if err != nil {
		return nil, fmt.Errorf("list icons: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}

	res := &IconGenResult{}
	for i, base := range names {
		if progress != nil {
			progress(base, i+1, len(names))
		}
		blp, err := cf.ReadIcon(base)
		if err != nil {
			res.Skipped++
			continue
		}
		img, err := decodeBLP2Bytes(blp)
		if err != nil {
			res.Skipped++
			if len(res.Warnings) < 20 {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", base, err))
			}
			continue
		}
		name := strings.ToLower(base)
		if err := writeJPEG(filepath.Join(outDir, name+".jpg"), img); err != nil {
			res.Skipped++
			continue
		}
		res.Generated++
	}
	return res, nil
}
