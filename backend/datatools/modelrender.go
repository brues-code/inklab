package datatools

// Software rasterizer for vanilla creature M2 models: orthographic, textured,
// z-buffered, with simple Lambert lighting. Renders a static (bind-pose) view
// to an RGBA image with a transparent background. Humanoid Character\ models
// (composite skins + geoset selection) are out of scope here — this targets
// creature skins (texture type 0 / 11-13).

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// RenderOptions controls a model render.
type RenderOptions struct {
	Size        int     // output is Size x Size
	YawDeg      float64 // rotation around the up axis (3/4 view)
	PitchDeg    float64
	Supersample int // SSAA factor; render at Size*ss then downscale (default 2)
}

// DefaultRenderOptions is a front-3/4 portrait framing with 2x SSAA.
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{Size: 512, YawDeg: 35, PitchDeg: 10, Supersample: 2}
}

// RenderCreatureModel resolves a display id, parses its M2, and rasterizes a
// textured static view.
func RenderCreatureModel(cf ClientFiles, displayID int, opt RenderOptions) (*image.RGBA, error) {
	cm, err := ResolveCreatureModel(cf, displayID)
	if err != nil {
		return nil, err
	}
	// Character (humanoid) models without a baked skin can't be textured, so skip
	// them (the caller falls back to octowow). Those with a baked skin render here.
	if cm.IsCharacter() && cm.BakedSkinPath() == "" {
		return nil, fmt.Errorf("display %d is a character model without a baked skin (%s)", displayID, cm.ModelPath)
	}
	mb, _, err := ReadModelFile(cf, cm.ModelPath)
	if err != nil {
		return nil, err
	}
	m, err := ParseM2(mb)
	if err != nil {
		return nil, err
	}
	return RenderM2(cf, cm, m, opt), nil
}

// RenderCreatureModelToFile renders a creature display to a PNG file. Returns an
// error (without creating the file) for character models or any parse/render
// failure, so callers can fall back to another source.
func RenderCreatureModelToFile(cf ClientFiles, displayID int, outPath string, opt RenderOptions) error {
	img, err := RenderCreatureModel(cf, displayID, opt)
	if err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

type screenVert struct {
	x, y, depth float64
	u, v        float32
	light       float64
}

// RenderM2 rasterizes a parsed model using the display's texture variations.
func RenderM2(cf ClientFiles, cm *CreatureModel, m *M2Model, opt RenderOptions) *image.RGBA {
	if opt.Size <= 0 {
		opt.Size = 512
	}
	ss := opt.Supersample
	if ss < 1 {
		ss = 1
	}
	// Render at ss× resolution, then box-downscale for anti-aliasing + soft edges.
	W, H := opt.Size*ss, opt.Size*ss
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	zbuf := make([]float64, W*H)
	for i := range zbuf {
		zbuf[i] = math.Inf(1)
	}

	// Decode each submesh's texture once (cache by path); also report its
	// material blend mode (0/1 opaque/key, 2 alpha, 3 additive).
	texCache := map[string]*image.RGBA{}
	texFor := func(sub int) (*image.RGBA, int) {
		for _, tu := range m.TexUnits {
			if int(tu.SkinSectionIndex) != sub {
				continue
			}
			blend := 0
			if int(tu.MaterialIndex) < len(m.Materials) {
				blend = int(m.Materials[tu.MaterialIndex].Blend)
			}
			p := cm.TextureForUnit(m, tu)
			if p == "" {
				return nil, blend
			}
			if t, ok := texCache[p]; ok {
				return t, blend
			}
			t := loadBLPTexture(cf, p)
			texCache[p] = t
			return t, blend
		}
		return nil, 0
	}

	// Orthographic camera with yaw+pitch.
	cx := (m.BoundsMin[0] + m.BoundsMax[0]) / 2
	cy := (m.BoundsMin[1] + m.BoundsMax[1]) / 2
	cz := (m.BoundsMin[2] + m.BoundsMax[2]) / 2
	yaw := opt.YawDeg * math.Pi / 180
	pitch := opt.PitchDeg * math.Pi / 180
	sYaw, cYaw := math.Sin(yaw), math.Cos(yaw)
	sPit, cPit := math.Sin(pitch), math.Cos(pitch)

	rot := func(x, y, z float64) (float64, float64, float64) {
		x2 := x*cYaw + z*sYaw
		z2 := -x*sYaw + z*cYaw
		y2 := y*cPit - z2*sPit
		z3 := y*sPit + z2*cPit
		return x2, y2, z3
	}

	// Fit the rotated silhouette: project the 8 bbox corners, take their 2D
	// extent, and scale/center to that. Works for tall, wide and rotated models
	// (vs. scaling by a single 3D axis, which mis-frames after rotation).
	var minX, maxX, minY, maxY float64
	firstC := true
	for cxi := 0; cxi < 2; cxi++ {
		for cyi := 0; cyi < 2; cyi++ {
			for czi := 0; czi < 2; czi++ {
				bx := m.BoundsMin[0]
				if cxi == 1 {
					bx = m.BoundsMax[0]
				}
				by := m.BoundsMin[1]
				if cyi == 1 {
					by = m.BoundsMax[1]
				}
				bz := m.BoundsMin[2]
				if czi == 1 {
					bz = m.BoundsMax[2]
				}
				x2, y2, _ := rot(float64(bx-cx), float64(by-cy), float64(bz-cz))
				if firstC || x2 < minX {
					minX = x2
				}
				if firstC || x2 > maxX {
					maxX = x2
				}
				if firstC || y2 < minY {
					minY = y2
				}
				if firstC || y2 > maxY {
					maxY = y2
				}
				firstC = false
			}
		}
	}
	projCx, projCy := (minX+maxX)/2, (minY+maxY)/2
	fit := maxf(maxX-minX, maxY-minY)
	if fit <= 0 {
		fit = 1
	}
	scale := float64(W) * 0.85 / fit

	project := func(p [3]float32) (float64, float64, float64) {
		x2, y2, z3 := rot(float64(p[0])-float64(cx), float64(p[1])-float64(cy), float64(p[2])-float64(cz))
		return (x2-projCx)*scale + float64(W)/2, float64(H)/2 - (y2-projCy)*scale, z3
	}
	rotN := func(n [3]float32) [3]float64 {
		x2, y2, z3 := rot(float64(n[0]), float64(n[1]), float64(n[2]))
		return [3]float64{x2, y2, z3}
	}

	light := normalize3([3]float64{-0.4, 0.7, -0.8}) // upper-left, toward camera

	selected := cm.SelectGeosets(cf, m)
	isChar := cm.IsCharacter()

	for si, sub := range m.SubMeshes {
		if !selected[si] {
			continue
		}
		tex, blend := texFor(si)
		// On character models, an untextured submesh (e.g. hair we can't resolve)
		// would draw as a gray blob — skip it instead.
		if tex == nil && isChar {
			continue
		}
		triEnd := sub.TriangleStart + uint32(sub.TriangleCount)
		for t := sub.TriangleStart; t+2 < triEnd && int(t+2) < len(m.Triangles); t += 3 {
			var verts [3]screenVert
			ok := true
			for k := 0; k < 3; k++ {
				ti := m.Triangles[t+uint32(k)]
				if int(ti) >= len(m.Indices) {
					ok = false
					break
				}
				vi := m.Indices[ti]
				if int(vi) >= len(m.Vertices) {
					ok = false
					break
				}
				vert := m.Vertices[vi]
				sx, sy, depth := project(vert.Pos)
				n := rotN(vert.Normal)
				li := 0.4 + 0.7*math.Max(0, dot3(n, light))
				if li > 1 {
					li = 1
				}
				verts[k] = screenVert{sx, sy, depth, vert.UV[0], vert.UV[1], li}
			}
			if ok {
				rasterTri(img, zbuf, W, H, verts[0], verts[1], verts[2], tex, blend)
			}
		}
	}
	if ss > 1 {
		return downscale(img, opt.Size, ss)
	}
	return img
}

// downscale box-filters a ss× supersampled image to outSize, averaging RGBA with
// premultiplied color so transparent edges don't darken the silhouette.
func downscale(src *image.RGBA, outSize, ss int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, outSize, outSize))
	n := ss * ss
	for y := 0; y < outSize; y++ {
		for x := 0; x < outSize; x++ {
			var rsum, gsum, bsum, asum int
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					c := src.RGBAAt(x*ss+dx, y*ss+dy)
					a := int(c.A)
					rsum += int(c.R) * a // premultiplied
					gsum += int(c.G) * a
					bsum += int(c.B) * a
					asum += a
				}
			}
			out := color.RGBA{A: uint8(asum / n)}
			if asum > 0 {
				out.R = uint8(rsum / asum)
				out.G = uint8(gsum / asum)
				out.B = uint8(bsum / asum)
			}
			dst.SetRGBA(x, y, out)
		}
	}
	return dst
}

// rasterTri fills a triangle: barycentric coverage, z-test, texture sampling
// (alpha-tested), and per-vertex Lambert shading.
func rasterTri(img *image.RGBA, zbuf []float64, W, H int, a, b, c screenVert, tex *image.RGBA, blend int) {
	minX := int(math.Floor(min3(a.x, b.x, c.x)))
	maxX := int(math.Ceil(max3(a.x, b.x, c.x)))
	minY := int(math.Floor(min3(a.y, b.y, c.y)))
	maxY := int(math.Ceil(max3(a.y, b.y, c.y)))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= W {
		maxX = W - 1
	}
	if maxY >= H {
		maxY = H - 1
	}
	area := edge(a, b, c)
	if area == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := screenVert{x: float64(x) + 0.5, y: float64(y) + 0.5}
			w0 := edge(b, c, px)
			w1 := edge(c, a, px)
			w2 := edge(a, b, px)
			// inside test (allow either winding)
			if !((w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0)) {
				continue
			}
			l0, l1, l2 := w0/area, w1/area, w2/area
			depth := l0*a.depth + l1*b.depth + l2*c.depth
			idx := y*W + x
			// Depth test against opaque geometry (transparent layers don't write z,
			// so multiple translucent/additive layers can stack).
			if depth >= zbuf[idx] {
				continue
			}
			u := l0*float64(a.u) + l1*float64(b.u) + l2*float64(c.u)
			v := l0*float64(a.v) + l1*float64(b.v) + l2*float64(c.v)
			var cr, cg, cb uint8 = 200, 200, 200
			var ca uint8 = 255
			if tex != nil {
				cr, cg, cb, ca = sampleTex(tex, u, v)
			}
			li := l0*a.light + l1*b.light + l2*c.light
			sr := clamp255(float64(cr) * li)
			sg := clamp255(float64(cg) * li)
			sb := clamp255(float64(cb) * li)
			dst := img.RGBAAt(x, y)

			switch blend {
			case 2: // alpha blend (translucent) — over existing, no z-write
				af := float64(ca) / 255
				img.SetRGBA(x, y, color.RGBA{
					uint8(sr*af + float64(dst.R)*(1-af)),
					uint8(sg*af + float64(dst.G)*(1-af)),
					uint8(sb*af + float64(dst.B)*(1-af)),
					maxU8(dst.A, ca),
				})
			case 3: // additive (glows, auras) — order-independent, no z-write
				af := float64(ca) / 255
				// Alpha tracks the brightness added, so dark additive texels stay
				// transparent (no dark panel) and bright glows show on any backdrop.
				addLum := maxf(sr, maxf(sg, sb)) * af
				img.SetRGBA(x, y, color.RGBA{
					uint8(clamp255(float64(dst.R) + sr*af)),
					uint8(clamp255(float64(dst.G) + sg*af)),
					uint8(clamp255(float64(dst.B) + sb*af)),
					maxU8(dst.A, uint8(clamp255(addLum))),
				})
			default: // 0 opaque / 1 alpha-key — alpha-tested, writes z
				if ca < 128 {
					continue
				}
				zbuf[idx] = depth
				img.SetRGBA(x, y, color.RGBA{uint8(sr), uint8(sg), uint8(sb), 255})
			}
		}
	}
}

func maxU8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

// edge is the signed area of the triangle (p0,p1,p2) (positive = CCW).
func edge(p0, p1, p2 screenVert) float64 {
	return (p1.x-p0.x)*(p2.y-p0.y) - (p1.y-p0.y)*(p2.x-p0.x)
}

func sampleTex(t *image.RGBA, u, v float64) (uint8, uint8, uint8, uint8) {
	w, h := t.Bounds().Dx(), t.Bounds().Dy()
	// wrap
	u = u - math.Floor(u)
	v = v - math.Floor(v)
	x := int(u * float64(w))
	y := int(v * float64(h))
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= w {
		x = w - 1
	}
	if y >= h {
		y = h - 1
	}
	c := t.RGBAAt(x, y)
	return c.R, c.G, c.B, c.A
}

// loadBLPTexture reads + decodes a BLP path to RGBA, or nil on failure.
func loadBLPTexture(cf ClientFiles, blpPath string) *image.RGBA {
	b, err := cf.ReadFile(blpPath)
	if err != nil {
		return nil
	}
	img, err := decodeBLP2Bytes(b)
	if err != nil {
		return nil
	}
	return img
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func min3(a, b, c float64) float64 { return math.Min(a, math.Min(b, c)) }
func max3(a, b, c float64) float64 { return math.Max(a, math.Max(b, c)) }
func dot3(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func normalize3(v [3]float64) [3]float64 {
	l := math.Sqrt(dot3(v, v))
	if l == 0 {
		return v
	}
	return [3]float64{v[0] / l, v[1] / l, v[2] / l}
}
func clamp255(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
