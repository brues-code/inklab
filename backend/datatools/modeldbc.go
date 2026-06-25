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
	return cm, nil
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
	if t.Type == 0 {
		return t.FileName
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
