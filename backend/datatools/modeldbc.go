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
	ModelPath   string    // e.g. Creature\Murloc\Murloc.mdx
	TexVars     [3]string // CreatureDisplayInfo texture variations (skin names)
	Extra       *CreatureExtra
	Attachments []AttachedItem // item models drawn at attachment points (shoulders, weapons)

	// Robe marks that the equipped chest displays as a robe, so SelectGeosets
	// picks the long robe skirt geoset instead of the default legs. Robe-ness is
	// a property of the appearance (a chest-slot item can display as a robe), so
	// it's detected from the chest ItemDisplayInfo's inventory icon, not the item
	// slot. Set during ResolveCreatureModel.
	Robe bool
}

// AttachedItem is an item model (shoulder pad, weapon, ...) drawn at one of the
// character model's attachment points, with its externally supplied texture.
type AttachedItem struct {
	ModelPath    string
	TexturePath  string
	AttachmentID uint32
}

// CreatureExtra holds the humanoid appearance from CreatureDisplayInfoExtra,
// present only for character (Character\Race\Sex) models. BakeName is the
// pre-composited body skin the client ships in Textures\BakedNpcTextures.
type CreatureExtra struct {
	Race                int
	Sex                 int
	HairStyle           int
	HairColor           int
	FacialHair          int
	BakeName            string
	HairTexture         string // resolved CharSections hair texture for the hair geoset
	ShoulderItemDisplay int    // ItemDisplayInfo id for the equipped shoulders (0 = none)
	ChestItemDisplay    int    // ItemDisplayInfo id for the equipped chest (for robe lookup; 0 = none)
}

// M2 attachment point IDs.
const (
	attachShield        = 0
	attachHandRight     = 1
	attachHandLeft      = 2
	attachShoulderRight = 5
	attachShoulderLeft  = 6
	attachHelm          = 11 // top of the head
)

// CreatureDisplayInfoExtra equipment columns. Appearance fields are 1-7, then
// equipment NPCItemDisplay slots start at 8 (helm), 9 (shoulder), ...
const (
	extraHelmField     = 8
	extraShoulderField = 9
	extraChestField    = 11
)

// Item object-component dirs for equipped pieces drawn as attached models.
const (
	shoulderComponentDir = `Item\ObjectComponents\Shoulder\`
	headComponentDir     = `Item\ObjectComponents\Head\`
)

// ResolveShoulders resolves a shoulder ItemDisplayInfo id to its left/right
// models + textures. ItemDisplayInfo layout: modelName[0,1]=fields 1,2 (left,
// right); modelTexture[0,1]=fields 3,4. Texture names are bare and resolve under
// the shoulder component dir.
func ResolveShoulders(cf ClientFiles, itemDisplayID int) []AttachedItem {
	if itemDisplayID <= 0 {
		return nil
	}
	b, err := cf.ReadDBC("ItemDisplayInfo.dbc")
	if err != nil {
		return nil
	}
	d, err := openDBCReader(b)
	if err != nil {
		return nil
	}
	r, ok := d.findByID(uint32(itemDisplayID))
	if !ok {
		return nil
	}
	texPath := func(name string) string {
		if name == "" {
			return ""
		}
		if !strings.HasSuffix(strings.ToLower(name), ".blp") {
			name += ".blp"
		}
		return shoulderComponentDir + name
	}
	var out []AttachedItem
	add := func(model, tex string, att uint32) {
		if model == "" {
			return
		}
		out = append(out, AttachedItem{
			ModelPath:    shoulderComponentDir + model,
			TexturePath:  texPath(tex),
			AttachmentID: att,
		})
	}
	add(d.str(r, 1), d.str(r, 3), attachShoulderLeft)  // modelName[0] = left
	add(d.str(r, 2), d.str(r, 4), attachShoulderRight) // modelName[1] = right
	return out
}

// ResolveHelm resolves a helm ItemDisplayInfo id to its attached head model.
// Helm models are race/sex specific: ItemDisplayInfo stores a base name (e.g.
// "Helm_Robe_C_04.mdx") and the actual file appends a 3-char race+sex suffix
// ("..._HUM" for Human Male). The model + its texture live under the Head
// component dir; it rides the helm attachment point (top of head).
func ResolveHelm(cf ClientFiles, itemDisplayID int, modelPath string) (AttachedItem, bool) {
	if itemDisplayID <= 0 {
		return AttachedItem{}, false
	}
	suffix := helmModelSuffix(modelPath)
	if suffix == "" {
		return AttachedItem{}, false // non-character model: no helm slot
	}
	b, err := cf.ReadDBC("ItemDisplayInfo.dbc")
	if err != nil {
		return AttachedItem{}, false
	}
	d, err := openDBCReader(b)
	if err != nil {
		return AttachedItem{}, false
	}
	r, ok := d.findByID(uint32(itemDisplayID))
	if !ok {
		return AttachedItem{}, false
	}
	model := d.str(r, 1) // modelName[0]
	if model == "" {
		return AttachedItem{}, false
	}
	stem := model[:len(model)-len(path.Ext(model))]
	tex := d.str(r, 3) // modelTexture[0]
	if tex != "" && !strings.HasSuffix(strings.ToLower(tex), ".blp") {
		tex += ".blp"
	}
	texPath := ""
	if tex != "" {
		texPath = headComponentDir + tex
	}
	return AttachedItem{
		ModelPath:    headComponentDir + stem + "_" + suffix + ".mdx",
		TexturePath:  texPath,
		AttachmentID: attachHelm,
	}, true
}

// chestIsRobe reports whether a chest ItemDisplayInfo displays as a robe, from
// its inventory icon (e.g. "INV_Robe_02"). Robe geometry is a property of the
// appearance, not the item slot — a chest-slot "cuirass" can display as a robe,
// so the item's inventory type is unreliable; the display's icon is the clean
// client-side signal. Checks both inventoryIcon slots (fields 5, 6).
func chestIsRobe(cf ClientFiles, itemDisplayID int) bool {
	if itemDisplayID <= 0 {
		return false
	}
	b, err := cf.ReadDBC("ItemDisplayInfo.dbc")
	if err != nil {
		return false
	}
	d, err := openDBCReader(b)
	if err != nil {
		return false
	}
	r, ok := d.findByID(uint32(itemDisplayID))
	if !ok {
		return false
	}
	for _, f := range []int{5, 6} { // inventoryIcon[0], inventoryIcon[1]
		if strings.Contains(strings.ToLower(d.str(r, f)), "robe") {
			return true
		}
	}
	return false
}

// helmModelSuffix derives the 3-char race+sex helm-model suffix from a character
// model path like "Character\Human\Male\HumanMale.mdx" -> "HUM" (first two
// letters of the race folder + the sex initial). Returns "" for non-character
// paths.
func helmModelSuffix(modelPath string) string {
	parts := strings.Split(strings.ReplaceAll(modelPath, "/", `\`), `\`)
	if len(parts) < 3 || !strings.EqualFold(parts[0], "Character") {
		return ""
	}
	race, sex := parts[1], parts[2]
	if len(race) < 2 || len(sex) < 1 {
		return ""
	}
	return strings.ToUpper(race[:2]) + strings.ToUpper(sex[:1])
}

// Item object-component dirs (model + texture live under the same dir).
const (
	weaponComponentDir = `Item\ObjectComponents\Weapon\`
	shieldComponentDir = `Item\ObjectComponents\Shield\`
)

// resolveComponentItem resolves an ItemDisplayInfo id to a single-model attached
// item (model = field 1, texture = field 3) under the given component dir.
func resolveComponentItem(cf ClientFiles, itemDisplayID int, dir string, attachID uint32) (AttachedItem, bool) {
	if itemDisplayID <= 0 {
		return AttachedItem{}, false
	}
	b, err := cf.ReadDBC("ItemDisplayInfo.dbc")
	if err != nil {
		return AttachedItem{}, false
	}
	d, err := openDBCReader(b)
	if err != nil {
		return AttachedItem{}, false
	}
	r, ok := d.findByID(uint32(itemDisplayID))
	if !ok {
		return AttachedItem{}, false
	}
	model := d.str(r, 1)
	if model == "" {
		return AttachedItem{}, false
	}
	tex := d.str(r, 3)
	if tex != "" && !strings.HasSuffix(strings.ToLower(tex), ".blp") {
		tex += ".blp"
	}
	texPath := ""
	if tex != "" {
		texPath = dir + tex
	}
	return AttachedItem{
		ModelPath:    dir + model,
		TexturePath:  texPath,
		AttachmentID: attachID,
	}, true
}

// ResolveWeapon resolves a weapon's ItemDisplayInfo id to its model + texture at
// a hand attachment point (1 = main/right, 2 = off/left).
func ResolveWeapon(cf ClientFiles, itemDisplayID int, attachID uint32) (AttachedItem, bool) {
	return resolveComponentItem(cf, itemDisplayID, weaponComponentDir, attachID)
}

// ResolveShield resolves a shield's ItemDisplayInfo id to its model + texture,
// placed at the left-forearm shield attachment.
func ResolveShield(cf ClientFiles, itemDisplayID int) (AttachedItem, bool) {
	return resolveComponentItem(cf, itemDisplayID, shieldComponentDir, attachShield)
}

// AttachmentPos returns the model-space (y-up) position of an attachment point
// by ID. In vanilla the stored position is already absolute model space (it
// equals the bone pivot), so it's used directly — no bone transform needed.
func (m *M2Model) AttachmentPos(id uint32) ([3]float32, bool) {
	for _, a := range m.Attachments {
		if a.ID == id {
			return a.Position, true
		}
	}
	return [3]float32{}, false
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

// DebugDBCRecord dumps every field of a DBC record both as u32 and as a string
// (dev only, for confirming column layouts).
func DebugDBCRecord(cf ClientFiles, name string, id uint32) {
	b, err := cf.ReadDBC(name)
	if err != nil {
		fmt.Printf("  [debug] %s: %v\n", name, err)
		return
	}
	d, err := openDBCReader(b)
	if err != nil {
		fmt.Printf("  [debug] %s: %v\n", name, err)
		return
	}
	r, ok := d.findByID(id)
	if !ok {
		fmt.Printf("  [debug] %s id=%d not found\n", name, id)
		return
	}
	fmt.Printf("  [debug] %s id=%d (fields=%d):\n", name, id, d.fc)
	for f := 0; f < d.fc; f++ {
		s := d.str(r, f)
		fmt.Printf("    f%d u32=%d str=%q\n", f, d.u32(r, f), s)
	}
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
					if extraShoulderField < ex.fc {
						cm.Extra.ShoulderItemDisplay = int(ex.u32(er, extraShoulderField))
						cm.Attachments = append(cm.Attachments, ResolveShoulders(cf, cm.Extra.ShoulderItemDisplay)...)
					}
					if extraHelmField < ex.fc {
						if helm, ok := ResolveHelm(cf, int(ex.u32(er, extraHelmField)), cm.ModelPath); ok {
							cm.Attachments = append(cm.Attachments, helm)
						}
					}
					if extraChestField < ex.fc {
						cm.Extra.ChestItemDisplay = int(ex.u32(er, extraChestField))
						cm.Robe = chestIsRobe(cf, cm.Extra.ChestItemDisplay)
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

	// Groups 100/200/300 are "facial" groups. On most models these are optional
	// facial HAIR (beard/moustache/sideburns), hair-textured and shown only when
	// the creature's facialHair setting selects the variant. But some models
	// (e.g. the custom Goblin) put the actual FACE/HEAD geometry in these groups,
	// textured with the body skin — skipping it beheads the model. Distinguish by
	// texture type: a body-textured facial group is real face geometry and must
	// always render its default (lowest-id) variant; a hair-textured one is
	// optional. Precompute, per facial group, its default submesh and whether
	// it's body-textured.
	faceDefault := map[int]int{} // group -> submesh index of its lowest id
	faceIsBody := map[int]bool{} // group -> uses the body skin (real face), not hair
	for i, s := range m.SubMeshes {
		g := int(s.ID) / 100 * 100
		if g != 100 && g != 200 && g != 300 {
			continue
		}
		if cur, ok := faceDefault[g]; !ok || s.ID < m.SubMeshes[cur].ID {
			faceDefault[g] = i
		}
		switch submeshTexType(m, i) {
		case 0, 1, 2: // hardcoded / body skin / extra body — not hair (type 6)
			faceIsBody[g] = true
		}
	}

	// Trousers group (1300): a robe shows the long skirt variant (1302) instead
	// of the default legs (1301). cm.Robe is set by the caller from the chest
	// item's inventory type.
	const trouserGroup = 1300
	trouserPick := trouserGroup + 1 // 1301 = normal legs
	if cm.Robe {
		trouserPick = trouserGroup + 2 // 1302 = robe skirt
	}

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
		case g == 100 || g == 200 || g == 300:
			// Body-textured face geometry: always show the default variant so the
			// head renders. Hair-textured facial hair that wasn't selected above is
			// an alternate; skip it.
			if faceIsBody[g] && faceDefault[g] == i {
				sel[i] = true
			}
		case g == trouserGroup: // legs vs robe skirt
			sel[i] = id == trouserPick
		case g != 0 && id == g+1: // variant 1 = each group's default "skin" geoset
			sel[i] = true
		}
	}
	return sel
}

// submeshTexType returns the primary texture's Type for a submesh (via its
// texture unit → texture-combo lookup), or -1 if none. Type 6 is hair; 0/1/2
// are hardcoded/body/extra-body skins.
func submeshTexType(m *M2Model, sub int) int {
	for _, tu := range m.TexUnits {
		if int(tu.SkinSectionIndex) != sub {
			continue
		}
		if int(tu.TextureComboIndex) < len(m.TextureCombos) {
			ti := m.TextureCombos[tu.TextureComboIndex]
			if int(ti) < len(m.Textures) {
				return int(m.Textures[ti].Type)
			}
		}
		return -1
	}
	return -1
}
