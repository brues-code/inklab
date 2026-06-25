package datatools

// Vanilla (1.12) M2/MDX model parser, sufficient for static rendering. Mirrors
// the structure of wow.export's MIT-licensed M2LegacyLoader (Kruithne,
// github.com/Kruithne/wow.export) for the vanilla layout: a legacy M2 ("MD20")
// has no chunk wrapper, and its header is a flat sequence of (count, offset)
// arrays whose data lives out-of-line. We walk that header and read only the
// arrays needed to draw a model (geometry, views/skins, textures, materials),
// skipping all animation tracks — those are read out-of-line and don't shift
// the header cursor, so a static pose needs none of them.

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	m2MagicMD20    = 0x3032444D // 'MD20'
	m2VerVanillaLo = 256
	m2VerVanillaHi = 257
	m2VerWotLK     = 264
)

// M2Vertex is one model vertex (positions/normals already converted from WoW
// z-up to y-up, matching wow.export).
type M2Vertex struct {
	Pos    [3]float32
	Normal [3]float32
	UV     [2]float32
	Bones  [4]uint8
	Wts    [4]uint8
}

// M2SubMesh is a contiguous run of triangles sharing geometry (a geoset).
type M2SubMesh struct {
	ID            uint16
	VertexStart   uint16
	VertexCount   uint16
	TriangleStart uint32 // already includes level<<16
	TriangleCount uint16
}

// M2TexUnit (batch) binds a submesh to a material + texture.
type M2TexUnit struct {
	SkinSectionIndex  uint16
	MaterialIndex     uint16
	TextureComboIndex uint16
	ColorIndex        uint16
}

// M2Texture is a texture slot: Type 0 = hardcoded FileName, 11/12/13 = monster
// skins resolved from CreatureDisplayInfo texture variations.
type M2Texture struct {
	Type     uint32
	FileName string
}

// M2Material holds render flags + blend mode.
type M2Material struct {
	Flags uint16
	Blend uint16
}

// M2Color is a material color tint (RGB 0-1) + alpha, taken from the peak
// keyframe of the color/alpha tracks (effect models animate these; the texture
// itself is often grayscale).
type M2Color struct {
	RGB   [3]float32
	Alpha float32
}

// M2Bone is the subset of a bone needed to place attachments and pose the model
// in its natural Stand frame: its rest pivot (model space, y-up), parent index,
// and the bone's local translation/rotation/scale evaluated at the first
// keyframe of the Stand animation. Most creatures stand in their bind pose
// (identity TRS), but serpentine/coiled models (naga, salamanders) author a
// straight bind pose and rely on Stand to bend the body, so we capture it.
type M2Bone struct {
	Parent int16
	Pivot  [3]float32
	Trans  [3]float32 // stand-pose local translation (y-up)
	Rot    [4]float32 // stand-pose local rotation quaternion (x,y,z,w), y-up
	Scale  [3]float32 // stand-pose local scale
}

// M2Attachment is an attachment point (e.g. shoulders, weapon hands): an ID, the
// bone it rides, and a position offset relative to that bone (model space, y-up).
type M2Attachment struct {
	ID       uint32
	Bone     uint16
	Position [3]float32
}

// M2Model is the parsed subset needed to render a static model.
type M2Model struct {
	Name     string
	Version  uint32
	Vertices []M2Vertex

	// View 0 (the highest-detail skin).
	Indices   []uint16 // skin → global vertex index
	Triangles []uint16 // index into Indices, 3 per face
	SubMeshes []M2SubMesh
	TexUnits  []M2TexUnit

	Textures      []M2Texture
	Materials     []M2Material
	TextureCombos []uint16 // textureComboIndex → texture index
	Colors        []M2Color // material color tracks (peak keyframe)

	Bones       []M2Bone       // rest pivots, for attachment placement
	Attachments []M2Attachment // attachment points (shoulders, hands, ...)
	Cameras     []M2Camera     // embedded cameras (portrait/paperdoll framing)

	// World-space bounding box (y-up), for camera framing.
	BoundsMin [3]float32
	BoundsMax [3]float32
}

// M2Camera is an embedded camera. Type 0 is the portrait camera (the head/bust
// framing the game renders in unit frames); type 1 is the paperdoll/character
// sheet camera; -1/others are cinematic. Position/Target are the static base
// vectors converted to the model's y-up space; FOV is in radians. The animation
// tracks (position/target/roll) are skipped — a static portrait uses the bases.
type M2Camera struct {
	Type     int32
	FOV      float32
	Position [3]float32 // position_base (eye)
	Target   [3]float32 // target_position_base (look-at)
}

type m2cur struct {
	b   []byte
	pos int
}

func (c *m2cur) seek(p int)      { c.pos = p }
func (c *m2cur) u32() uint32     { v := binary.LittleEndian.Uint32(c.b[c.pos:]); c.pos += 4; return v }
func (c *m2cur) u16() uint16     { v := binary.LittleEndian.Uint16(c.b[c.pos:]); c.pos += 2; return v }
func (c *m2cur) u8() uint8       { v := c.b[c.pos]; c.pos++; return v }
func (c *m2cur) f32() float32    { v := math.Float32frombits(binary.LittleEndian.Uint32(c.b[c.pos:])); c.pos += 4; return v }
func (c *m2cur) arr() (uint32, uint32) { return c.u32(), c.u32() } // (count, offset)

// ParseM2 parses a vanilla legacy M2 (MD20) into the subset needed for static
// rendering. Returns an error for non-vanilla or malformed input.
func ParseM2(b []byte) (*M2Model, error) {
	if len(b) < 8 || binary.LittleEndian.Uint32(b) != m2MagicMD20 {
		return nil, fmt.Errorf("not a legacy M2 (bad MD20 magic)")
	}
	m := &M2Model{}
	c := &m2cur{b: b, pos: 4}
	m.Version = c.u32()
	if m.Version < m2VerVanillaLo || m.Version > m2VerWotLK {
		return nil, fmt.Errorf("unsupported M2 version %d", m.Version)
	}
	vanilla := m.Version <= m2VerVanillaHi

	readStr := func(off, n uint32) string {
		if off == 0 || n == 0 || int(off+n) > len(b) {
			return ""
		}
		return strings.TrimRight(string(b[off:off+n]), "\x00")
	}

	// --- header walk (each array is 8 bytes here; data is out-of-line) ---
	nameN, nameOff := c.arr()
	m.Name = readStr(nameOff, nameN)
	c.u32()  // flags
	c.arr()  // global loops
	animN, animOff := c.arr() // animations
	c.arr()  // animation lookup
	if vanilla {
		c.arr() // playable animation lookup (vanilla only)
	}
	bonesN, bonesOff := c.arr() // bones
	c.arr()                     // key bone lookup
	vtxN, vtxOff := c.arr()
	viewsN, viewsOff := c.arr()
	colorsN, colorsOff := c.arr()
	texN, texOff := c.arr()
	c.arr() // texture weights
	if vanilla {
		c.arr() // texture flipbooks (vanilla only)
	}
	c.arr() // texture transforms
	c.arr() // replaceable texture lookup
	matN, matOff := c.arr()
	c.arr() // bone combos
	comboN, comboOff := c.arr()
	// Continue the header walk to reach attachments (last array we need). Layout
	// after texture combos: texture-transform bone map, transparency lookup,
	// texture-transform lookup, then the INLINE bounding box + collision spheres
	// (56 bytes), then 3 collision arrays, then attachments.
	c.arr()      // texture transform bone map
	c.arr()      // transparency lookup
	c.arr()      // texture transform lookup
	c.pos += 56  // bounding box (min3,max3,radius) + collision box (min3,max3,radius)
	c.arr()      // collision indices
	c.arr()      // collision positions
	c.arr()      // collision normals
	attN, attOff := c.arr()
	c.arr() // attachment lookup
	c.arr() // events
	c.arr() // lights
	camN, camOff := c.arr() // cameras
	// (camera lookup follows; not needed — the Type field identifies the portrait cam)

	// --- vertices (48 bytes each) ---
	c.seek(int(vtxOff))
	m.Vertices = make([]M2Vertex, vtxN)
	first := true
	for i := range m.Vertices {
		var v M2Vertex
		// position: x, z(-y), y  (z-up -> y-up)
		x := c.f32()
		z := c.f32()
		y := c.f32()
		v.Pos = [3]float32{x, y, -z}
		for j := 0; j < 4; j++ {
			v.Wts[j] = c.u8()
		}
		for j := 0; j < 4; j++ {
			v.Bones[j] = c.u8()
		}
		nx := c.f32()
		nz := c.f32()
		ny := c.f32()
		v.Normal = [3]float32{nx, ny, -nz}
		v.UV = [2]float32{c.f32(), c.f32()}
		c.f32() // uv2 x
		c.f32() // uv2 y
		m.Vertices[i] = v
		for k := 0; k < 3; k++ {
			if first || v.Pos[k] < m.BoundsMin[k] {
				m.BoundsMin[k] = v.Pos[k]
			}
			if first || v.Pos[k] > m.BoundsMax[k] {
				m.BoundsMax[k] = v.Pos[k]
			}
		}
		first = false
	}

	// --- view 0 (skin) ---
	if viewsN > 0 {
		c.seek(int(viewsOff))
		idxN, idxOff := c.arr()
		triN, triOff := c.arr()
		c.arr() // properties
		subN, subOff := c.arr()
		tuN, tuOff := c.arr()
		// (skin.bones u32 follows, then next view; we only need view 0)

		c.seek(int(idxOff))
		m.Indices = make([]uint16, idxN)
		for i := range m.Indices {
			m.Indices[i] = c.u16()
		}
		c.seek(int(triOff))
		m.Triangles = make([]uint16, triN)
		for i := range m.Triangles {
			m.Triangles[i] = c.u16()
		}
		c.seek(int(subOff))
		m.SubMeshes = make([]M2SubMesh, subN)
		for i := range m.SubMeshes {
			s := M2SubMesh{}
			s.ID = c.u16()
			level := c.u16()
			s.VertexStart = c.u16()
			s.VertexCount = c.u16()
			triStart := c.u16()
			s.TriangleCount = c.u16()
			c.u16() // boneCount
			c.u16() // boneStart
			c.u16() // boneInfluences
			c.u16() // centerBoneIndex
			c.f32() // centerPosition x
			c.f32() // y
			c.f32() // z
			s.TriangleStart = uint32(triStart) + (uint32(level) << 16)
			m.SubMeshes[i] = s
		}
		c.seek(int(tuOff))
		m.TexUnits = make([]M2TexUnit, tuN)
		for i := range m.TexUnits {
			tu := M2TexUnit{}
			c.u8()  // flags
			c.u8()  // priority
			c.u16() // shaderID
			tu.SkinSectionIndex = c.u16()
			c.u16() // flags2
			tu.ColorIndex = c.u16()
			tu.MaterialIndex = c.u16()
			c.u16() // materialLayer
			c.u16() // textureCount
			tu.TextureComboIndex = c.u16()
			c.u16() // textureCoordComboIndex
			c.u16() // textureWeightComboIndex
			c.u16() // textureTransformComboIndex
			m.TexUnits[i] = tu
		}
	}

	// --- textures ---
	c.seek(int(texOff))
	m.Textures = make([]M2Texture, texN)
	for i := range m.Textures {
		t := M2Texture{}
		t.Type = c.u32()
		c.u32() // flags
		nlen, noff := c.arr()
		if t.Type == 0 {
			t.FileName = readStr(noff, nlen)
		}
		m.Textures[i] = t
	}

	// --- materials ---
	c.seek(int(matOff))
	m.Materials = make([]M2Material, matN)
	for i := range m.Materials {
		m.Materials[i] = M2Material{Flags: c.u16(), Blend: c.u16()}
	}

	// --- texture combos (lookup) ---
	c.seek(int(comboOff))
	m.TextureCombos = make([]uint16, comboN)
	for i := range m.TextureCombos {
		m.TextureCombos[i] = c.u16()
	}

	// --- colors (each: a color track [float3] + alpha track [int16]) ---
	// Each entry is two inline M2Tracks of 28 bytes; within a track the values
	// array (count, offset) sits at byte 20. We take the peak keyframe so an
	// animated tint reads at full strength rather than at a (possibly dark) frame 0.
	f32at := func(off int) float32 {
		if off+4 > len(b) {
			return 0
		}
		return math.Float32frombits(binary.LittleEndian.Uint32(b[off:]))
	}
	u32at := func(off int) int {
		if off+4 > len(b) {
			return 0
		}
		return int(binary.LittleEndian.Uint32(b[off:]))
	}
	m.Colors = make([]M2Color, colorsN)
	for i := range m.Colors {
		col := M2Color{RGB: [3]float32{1, 1, 1}, Alpha: 1}
		base := int(colorsOff) + i*56
		// color track values
		cCount, cOff := u32at(base+20), u32at(base+24)
		bestLum := -1.0
		for k := 0; k < cCount; k++ {
			p := cOff + k*12
			r, g, bl := f32at(p), f32at(p+4), f32at(p+8)
			if lum := float64(r) + float64(g) + float64(bl); lum > bestLum {
				bestLum = lum
				col.RGB = [3]float32{r, g, bl}
			}
		}
		// alpha track values (int16 fixed-point, /32767)
		aCount, aOff := u32at(base+48), u32at(base+52)
		amax, found := int16(0), false
		for k := 0; k < aCount; k++ {
			p := aOff + k*2
			if p+2 > len(b) {
				break
			}
			if v := int16(binary.LittleEndian.Uint16(b[p:])); !found || v > amax {
				amax, found = v, true
			}
		}
		if found {
			col.Alpha = float32(amax) / 32767
		}
		m.Colors[i] = col
	}

	// --- bones (rest pivots only) + attachments ---
	// Struct strides depend on the M2Track size, which is 28 bytes in vanilla
	// (single timeline: interp+gseq+ranges+timestamps+values) and 20 in WotLK.
	// TBC+ also adds a 4-byte boneNameCRC. The pivot is the last field of a bone.
	trackSize := 28
	if m.Version >= m2VerWotLK {
		trackSize = 20
	}
	boneSize := 4 + 4 + 2 + 2 + 3*trackSize + 12 // id,flags,parent,submesh,3 tracks,pivot
	pivotOff := 4 + 4 + 2 + 2 + 3*trackSize
	if m.Version >= 260 { // TBC+ boneNameCRC after submeshID
		boneSize += 4
		pivotOff += 4
	}
	conv := func(x, y, z float32) [3]float32 { return [3]float32{x, z, -y} } // z-up → y-up

	// Find the Stand animation's index (animID 0). Bone tracks store keyframes on
	// a single timeline; the per-track "interpolation ranges" map an animation
	// index to its [first,last] keyframe range, so we read each bone's TRS at the
	// first keyframe of Stand. Only vanilla (single-timeline tracks, 68-byte
	// animation structs) is handled — newer formats index tracks per-animation
	// differently, so they fall back to the identity bind pose (Trans 0, Rot
	// identity, Scale 1), which is the prior behaviour.
	standIdx := 0
	if vanilla {
		const animStride = 68
		for i := 0; i < int(animN); i++ {
			a := int(animOff) + i*animStride
			if a+2 > len(b) {
				break
			}
			if binary.LittleEndian.Uint16(b[a:]) == 0 { // animID 0 = Stand
				standIdx = i
				break
			}
		}
	}
	// standValueOff returns the byte offset of the Stand-frame value in a bone
	// track (header at trackBase), or -1 if the track has no keyframes.
	standValueOff := func(trackBase, elemSize int) int {
		valN := int(u32(b, trackBase+20))
		valOff := int(u32(b, trackBase+24))
		if valN == 0 {
			return -1
		}
		key := 0
		if int16(u16(b, trackBase+2)) < 0 { // globalSequence < 0 → per-animation ranges
			rngN := int(u32(b, trackBase+4))
			rngOff := int(u32(b, trackBase+8))
			if standIdx < rngN {
				key = int(u32(b, rngOff+standIdx*8)) // range .first = first keyframe index
			}
		}
		if key < 0 || key >= valN {
			key = 0
		}
		return valOff + key*elemSize
	}
	m.Bones = make([]M2Bone, bonesN)
	for i := range m.Bones {
		base := int(bonesOff) + i*boneSize
		if base+boneSize > len(b) {
			break
		}
		parent := int16(binary.LittleEndian.Uint16(b[base+8:]))
		px := math.Float32frombits(binary.LittleEndian.Uint32(b[base+pivotOff:]))
		py := math.Float32frombits(binary.LittleEndian.Uint32(b[base+pivotOff+4:]))
		pz := math.Float32frombits(binary.LittleEndian.Uint32(b[base+pivotOff+8:]))
		bone := M2Bone{
			Parent: parent,
			Pivot:  conv(px, py, pz),
			Trans:  [3]float32{0, 0, 0},
			Rot:    [4]float32{0, 0, 0, 1},
			Scale:  [3]float32{1, 1, 1},
		}
		if vanilla {
			transBase := base + 12               // first track (translation)
			rotBase := transBase + trackSize     // rotation
			scaleBase := transBase + 2*trackSize // scale
			if o := standValueOff(transBase, 12); o >= 0 {
				bone.Trans = conv(float32(f32(b, o)), float32(f32(b, o+4)), float32(f32(b, o+8)))
			}
			if o := standValueOff(rotBase, 16); o >= 0 {
				// quaternion (x,y,z,w) float32; rotate axis into y-up like a vector.
				qx, qy, qz, qw := float32(f32(b, o)), float32(f32(b, o+4)), float32(f32(b, o+8)), float32(f32(b, o+12))
				bone.Rot = [4]float32{qx, qz, -qy, qw}
			}
			if o := standValueOff(scaleBase, 12); o >= 0 {
				// scale is per-axis magnitude; permute axes (y↔z) without sign flip.
				sx, sy, sz := float32(f32(b, o)), float32(f32(b, o+4)), float32(f32(b, o+8))
				bone.Scale = [3]float32{sx, sz, sy}
			}
		}
		m.Bones[i] = bone
	}

	// Attachment: id u32, bone u16, unknown u16, position float3, then a track.
	attSize := 4 + 2 + 2 + 12 + trackSize
	m.Attachments = make([]M2Attachment, attN)
	for i := range m.Attachments {
		base := int(attOff) + i*attSize
		if base+8+12 > len(b) {
			break
		}
		id := binary.LittleEndian.Uint32(b[base:])
		bone := binary.LittleEndian.Uint16(b[base+4:])
		ax := math.Float32frombits(binary.LittleEndian.Uint32(b[base+8:]))
		ay := math.Float32frombits(binary.LittleEndian.Uint32(b[base+12:]))
		az := math.Float32frombits(binary.LittleEndian.Uint32(b[base+16:]))
		m.Attachments[i] = M2Attachment{ID: id, Bone: bone, Position: conv(ax, ay, az)}
	}

	// --- cameras (portrait/paperdoll framing) ---
	// Vanilla & WotLK M2Camera both carry a `fov` field; it was removed in Cata,
	// which this parser doesn't target. Stride and the two base-vector offsets
	// are driven by the track size (28 vanilla / 20 WotLK):
	//   type u32, fov f32, far f32, near f32, positions track, position_base vec3,
	//   target_position track, target_position_base vec3, roll track.
	camStride := 40 + 3*trackSize
	posBaseOff := 16 + trackSize
	targetBaseOff := 28 + 2*trackSize
	m.Cameras = make([]M2Camera, 0, camN)
	for i := 0; i < int(camN); i++ {
		base := int(camOff) + i*camStride
		if base+targetBaseOff+12 > len(b) {
			break
		}
		cam := M2Camera{
			Type:     int32(binary.LittleEndian.Uint32(b[base:])),
			FOV:      f32at(base + 4),
			Position: conv(f32at(base+posBaseOff), f32at(base+posBaseOff+4), f32at(base+posBaseOff+8)),
			Target:   conv(f32at(base+targetBaseOff), f32at(base+targetBaseOff+4), f32at(base+targetBaseOff+8)),
		}
		m.Cameras = append(m.Cameras, cam)
	}

	return m, nil
}

// PortraitCamera returns the model's portrait camera (Type 0) if present.
func (m *M2Model) PortraitCamera() (M2Camera, bool) {
	for _, c := range m.Cameras {
		if c.Type == 0 {
			return c, true
		}
	}
	return M2Camera{}, false
}
