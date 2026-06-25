package datatools

// Resolves a creature display id to its model file + skin textures via the
// vanilla DBC chain: CreatureDisplayInfo (display -> model id + texture
// variations) -> CreatureModelData (model id -> .mdx path).

import (
	"encoding/binary"
	"fmt"
	"path"
	"strings"
)

// CreatureModel is the resolved render input for a display id.
type CreatureModel struct {
	DisplayID int
	ModelID   int
	ModelPath string    // e.g. Creature\Murloc\Murloc.mdx
	TexVars   [3]string // CreatureDisplayInfo texture variations (skin names)
	Extra     *CreatureExtra
}

// CreatureExtra holds the humanoid appearance from CreatureDisplayInfoExtra,
// present only for character (Character\Race\Sex) models. BakeName is the
// pre-composited body skin the client ships in Textures\BakedNpcTextures.
type CreatureExtra struct {
	Race       int
	Sex        int
	HairStyle   int
	HairColor   int
	FacialHair  int
	BakeName    string
	HairTexture string // resolved CharSections hair texture for the hair geoset
}

// IsCharacter reports whether this display uses a humanoid character model.
func (cm *CreatureModel) IsCharacter() bool {
	return strings.HasPrefix(strings.ToLower(cm.ModelPath), `character\`)
}

// BakedSkinPath returns the baked composite-skin texture path, or "".
func (cm *CreatureModel) BakedSkinPath() string {
	if cm.Extra == nil || cm.Extra.BakeName == "" {
		return ""
	}
	return `Textures\BakedNpcTextures\` + cm.Extra.BakeName
}

// dbcReader is a minimal WDBC accessor (record/field/string-block).
type dbcReader struct {
	b   []byte
	rc  int
	fc  int
	rs  int
	sb  int // string block start
}

func openDBCReader(b []byte) (*dbcReader, error) {
	if len(b) < 20 || string(b[0:4]) != "WDBC" {
		return nil, fmt.Errorf("not a WDBC")
	}
	d := &dbcReader{
		b:  b,
		rc: int(binary.LittleEndian.Uint32(b[4:8])),
		fc: int(binary.LittleEndian.Uint32(b[8:12])),
		rs: int(binary.LittleEndian.Uint32(b[12:16])),
	}
	d.sb = 20 + d.rc*d.rs
	return d, nil
}

func (d *dbcReader) u32(rec, field int) uint32 {
	o := 20 + rec*d.rs + field*4
	if o+4 > len(d.b) {
		return 0
	}
	return binary.LittleEndian.Uint32(d.b[o : o+4])
}

func (d *dbcReader) str(rec, field int) string {
	o := d.sb + int(d.u32(rec, field))
	if o < d.sb || o >= len(d.b) {
		return ""
	}
	e := o
	for e < len(d.b) && d.b[e] != 0 {
		e++
	}
	return string(d.b[o:e])
}

// findByID scans for the record whose field 0 equals id.
func (d *dbcReader) findByID(id uint32) (int, bool) {
	for r := 0; r < d.rc; r++ {
		if d.u32(r, 0) == id {
			return r, true
		}
	}
	return 0, false
}

// ResolveCreatureModel resolves a display id to its model path + skin textures.
// Field layout is the vanilla DBC chain (CreatureDisplayInfo: modelID=1,
// textureVariation=6,7,8; CreatureModelData: modelName=2).
func ResolveCreatureModel(cf ClientFiles, displayID int) (*CreatureModel, error) {
	cdiBytes, err := cf.ReadDBC("CreatureDisplayInfo.dbc")
	if err != nil {
		return nil, fmt.Errorf("read CreatureDisplayInfo: %w", err)
	}
	cdi, err := openDBCReader(cdiBytes)
	if err != nil {
		return nil, err
	}
	rec, ok := cdi.findByID(uint32(displayID))
	if !ok {
		return nil, fmt.Errorf("display id %d not found", displayID)
	}
	cm := &CreatureModel{
		DisplayID: displayID,
		ModelID:   int(cdi.u32(rec, 1)),
		TexVars:   [3]string{cdi.str(rec, 6), cdi.str(rec, 7), cdi.str(rec, 8)},
	}

	cmdBytes, err := cf.ReadDBC("CreatureModelData.dbc")
	if err != nil {
		return nil, fmt.Errorf("read CreatureModelData: %w", err)
	}
	cmd, err := openDBCReader(cmdBytes)
	if err != nil {
		return nil, err
	}
	mrec, ok := cmd.findByID(uint32(cm.ModelID))
	if !ok {
		return nil, fmt.Errorf("model id %d not found", cm.ModelID)
	}
	cm.ModelPath = cmd.str(mrec, 2)

	// Humanoid appearance (CreatureDisplayInfo.extendedDisplayInfoID, field 3).
	if extID := cdi.u32(rec, 3); extID > 0 {
		if exb, err := cf.ReadDBC("CreatureDisplayInfoExtra.dbc"); err == nil {
			if ex, err := openDBCReader(exb); err == nil {
				if er, ok := ex.findByID(extID); ok {
					cm.Extra = &CreatureExtra{
						Race:       int(ex.u32(er, 1)),
						Sex:        int(ex.u32(er, 2)),
						HairStyle:  int(ex.u32(er, 5)),
						HairColor:  int(ex.u32(er, 6)),
						FacialHair: int(ex.u32(er, 7)),
						BakeName:   ex.str(er, ex.fc-1), // bakeName is the last field
					}
					cm.Extra.HairTexture = ResolveHairTexture(cf, cm.Extra.Race, cm.Extra.Sex, cm.Extra.HairColor)
				}
			}
		}
	}
	return cm, nil
}

// ResolveHairTexture returns the hair texture for a (race, sex, hairColor) from
// CharSections.dbc (baseSection 3 = Hair, texture at field 6), preferring a
// matching color and falling back to any hair row for the race/sex.
func ResolveHairTexture(cf ClientFiles, race, sex, hairColor int) string {
	b, err := cf.ReadDBC("CharSections.dbc")
	if err != nil {
		return ""
	}
	d, err := openDBCReader(b)
	if err != nil {
		return ""
	}
	fallback := ""
	for r := 0; r < d.rc; r++ {
		if int(d.u32(r, 1)) != race || int(d.u32(r, 2)) != sex || d.u32(r, 3) != 3 {
			continue
		}
		tex := d.str(r, 6)
		if tex == "" {
			continue
		}
		if int(d.u32(r, 5)) == hairColor {
			return tex
		}
		if fallback == "" {
			fallback = tex
		}
	}
	return fallback
}

// ResolveHairGeoset returns the group-0 hair geoset id for a (race, sex,
// hairStyle) from CharHairGeosets.dbc, or -1 if not found.
func ResolveHairGeoset(cf ClientFiles, race, sex, hairStyle int) int {
	b, err := cf.ReadDBC("CharHairGeosets.dbc")
	if err != nil {
		return -1
	}
	d, err := openDBCReader(b)
	if err != nil {
		return -1
	}
	for r := 0; r < d.rc; r++ {
		if int(d.u32(r, 1)) == race && int(d.u32(r, 2)) == sex && int(d.u32(r, 3)) == hairStyle {
			return int(d.u32(r, 4))
		}
	}
	return -1
}

// ReadModelFile reads the model bytes, tolerating the .mdx/.m2 extension split
// (CreatureModelData stores .mdx; the actual file may be .m2 or vice versa).
func ReadModelFile(cf ClientFiles, modelPath string) ([]byte, string, error) {
	candidates := []string{modelPath}
	ext := strings.ToLower(path.Ext(modelPath))
	stem := modelPath[:len(modelPath)-len(path.Ext(modelPath))]
	if ext == ".mdx" {
		candidates = append(candidates, stem+".m2", stem+".M2")
	} else if ext == ".m2" {
		candidates = append(candidates, stem+".mdx", stem+".MDX")
	}
	var lastErr error
	for _, p := range candidates {
		if b, err := cf.ReadFile(p); err == nil {
			return b, p, nil
		} else {
			lastErr = err
		}
	}
	return nil, "", fmt.Errorf("model not found (%v): %w", candidates, lastErr)
}

// TextureForUnit resolves the BLP path for a texture unit's primary texture.
// Type 0 textures carry a hardcoded filename; monster-skin types (11/12/13) are
// replaced by the display's texture variations, joined to the model directory.
func (cm *CreatureModel) TextureForUnit(m *M2Model, tu M2TexUnit) string {
	if int(tu.TextureComboIndex) >= len(m.TextureCombos) {
		return ""
	}
	texIdx := m.TextureCombos[tu.TextureComboIndex]
	if int(texIdx) >= len(m.Textures) {
		return ""
	}
	t := m.Textures[texIdx]
	switch t.Type {
	case 0:
		return t.FileName
	case 1, 8: // character body skin -> the baked composite the client ships
		return cm.BakedSkinPath()
	case 6: // character hair
		if cm.Extra != nil {
			return cm.Extra.HairTexture
		}
	}
	// Monster skins: variation index = type - 11. The skin BLP lives in the
	// model file's directory.
	vi := int(t.Type) - 11
	if vi >= 0 && vi < len(cm.TexVars) && cm.TexVars[vi] != "" {
		dir := path.Dir(strings.ReplaceAll(cm.ModelPath, `\`, "/"))
		name := cm.TexVars[vi]
		if !strings.HasSuffix(strings.ToLower(name), ".blp") {
			name += ".blp"
		}
		return strings.ReplaceAll(path.Join(dir, name), "/", `\`)
	}
	return ""
}

// SelectGeosets returns the set of submesh INDICES to render. Creatures draw
// every submesh; character models draw the base body + the selected hairstyle +
// the default (lowest) variant of each body-part group, skipping alternates
// (other hairstyles, facial hair, tabards/cloaks/etc.). The baked skin already
// carries the equipped armor as texture, so this yields a clothed humanoid.
func (cm *CreatureModel) SelectGeosets(cf ClientFiles, m *M2Model) map[int]bool {
	sel := map[int]bool{}
	if !cm.IsCharacter() || cm.Extra == nil {
		for i := range m.SubMeshes {
			sel[i] = true
		}
		return sel
	}
	hair := ResolveHairGeoset(cf, cm.Extra.Race, cm.Extra.Sex, cm.Extra.HairStyle)
	for i, s := range m.SubMeshes {
		id := int(s.ID)
		g := id / 100 * 100
		switch {
		case id == 0: // base body
			sel[i] = true
		case g == 0 && id == hair: // selected hairstyle
			sel[i] = true
		case (g == 100 || g == 200 || g == 300) && cm.Extra.FacialHair > 0 && id == g+cm.Extra.FacialHair:
			sel[i] = true // facial hair (beard/moustache/sideburns) variant
		case (g == 100 || g == 200 || g == 300):
			// other facial-group variants are alternates; skip
		case g != 0 && id == g+1: // variant 1 = each group's default "skin" geoset
			sel[i] = true
		}
	}
	return sel
}
