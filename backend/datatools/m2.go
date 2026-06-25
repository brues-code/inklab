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

	// World-space bounding box (y-up), for camera framing.
	BoundsMin [3]float32
	BoundsMax [3]float32
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
	c.arr()  // animations
	c.arr()  // animation lookup
	if vanilla {
		c.arr() // playable animation lookup (vanilla only)
	}
	c.arr() // bones
	c.arr() // key bone lookup
	vtxN, vtxOff := c.arr()
	viewsN, viewsOff := c.arr()
	c.arr() // colors
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
	// (remaining header arrays not needed for static rendering)

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

	return m, nil
}
