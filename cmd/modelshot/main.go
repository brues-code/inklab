// modelshot is a dev spike: resolve a creature display id to its model in the
// client MPQs, parse the vanilla M2, and summarize it — validating the parser
// against real data before building the rasterizer.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strings"

	"inklab/backend/datatools"
)

func main() {
	display := flag.Int("display", 384, "creature display id")
	dataDir := flag.String("data", `C:/WoW/Octo/Data`, "WoW client Data dir")
	out := flag.String("out", "", "render to this PNG path")
	size := flag.Int("size", 512, "render size")
	yaw := flag.Float64("yaw", 35, "yaw degrees")
	pitch := flag.Float64("pitch", 10, "pitch degrees")
	flag.Parse()

	cf, err := datatools.NewMpqSource(*dataDir)
	if err != nil {
		fmt.Println("open MPQ:", err)
		os.Exit(1)
	}
	defer cf.Close()

	cm, err := datatools.ResolveCreatureModel(cf, *display)
	if err != nil {
		fmt.Println("resolve:", err)
		os.Exit(1)
	}
	fmt.Printf("display %d -> modelID %d\n  modelPath=%q\n  texVars=%v\n",
		cm.DisplayID, cm.ModelID, cm.ModelPath, cm.TexVars)

	mb, usedPath, err := datatools.ReadModelFile(cf, cm.ModelPath)
	if err != nil {
		fmt.Println("read model:", err)
		os.Exit(1)
	}
	fmt.Printf("  read model from %q (%d bytes)\n", usedPath, len(mb))

	m, err := datatools.ParseM2(mb)
	if err != nil {
		fmt.Println("parse M2:", err)
		os.Exit(1)
	}

	fmt.Printf("\nM2 %q v%d\n", m.Name, m.Version)
	fmt.Printf("  vertices=%d indices=%d triangles=%d (faces=%d)\n",
		len(m.Vertices), len(m.Indices), len(m.Triangles), len(m.Triangles)/3)
	fmt.Printf("  submeshes=%d texUnits=%d textures=%d materials=%d combos=%d\n",
		len(m.SubMeshes), len(m.TexUnits), len(m.Textures), len(m.Materials), len(m.TextureCombos))
	fmt.Printf("  bounds min=%v max=%v\n", m.BoundsMin, m.BoundsMax)

	fmt.Println("  textures:")
	for i, t := range m.Textures {
		fmt.Printf("    [%d] type=%d file=%q\n", i, t.Type, t.FileName)
	}

	fmt.Println("  submeshes & resolved textures:")
	for i, s := range m.SubMeshes {
		tex := ""
		for _, tu := range m.TexUnits {
			if int(tu.SkinSectionIndex) == i {
				tex = cm.TextureForUnit(m, tu)
				break
			}
		}
		fmt.Printf("    sub[%d] id=%d vtx[%d..%d] triStart=%d triCount=%d tex=%q\n",
			i, s.ID, s.VertexStart, s.VertexStart+s.VertexCount, s.TriangleStart, s.TriangleCount, tex)
		if i >= 12 {
			fmt.Printf("    ...(%d submeshes total)\n", len(m.SubMeshes))
			break
		}
	}

	// sanity: do the texture BLPs exist?
	fmt.Println("  texture file existence:")
	seen := map[string]bool{}
	for _, tu := range m.TexUnits {
		p := cm.TextureForUnit(m, tu)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		_, e := cf.ReadFile(p)
		fmt.Printf("    %q -> exists=%v\n", p, e == nil)
	}
	_ = strings.TrimSpace

	if *out != "" {
		img, err := datatools.RenderCreatureModel(cf, *display,
			datatools.RenderOptions{Size: *size, YawDeg: *yaw, PitchDeg: *pitch})
		if err != nil {
			fmt.Println("render:", err)
			os.Exit(1)
		}
		f, err := os.Create(*out)
		if err != nil {
			fmt.Println("create:", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			fmt.Println("encode:", err)
			os.Exit(1)
		}
		// coverage: fraction of non-transparent pixels
		opaque := 0
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if img.RGBAAt(x, y).A > 0 {
					opaque++
				}
			}
		}
		total := b.Dx() * b.Dy()
		fmt.Printf("\nrendered -> %s  (%.1f%% non-transparent)\n", *out, 100*float64(opaque)/float64(total))
	}
}
