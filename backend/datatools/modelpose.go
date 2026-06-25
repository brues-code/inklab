package datatools

// Skeletal posing for M2 models: builds each bone's world matrix from the
// Stand-frame local TRS captured by ParseM2, then skins the vertices so the
// model renders in its natural standing pose rather than the raw bind pose.
// Most creatures stand in their bind pose (every bone is identity, so this is a
// no-op), but serpentine/coiled models (naga, salamanders) author a straight
// bind pose and bend the body via Stand; without this they render as a straight
// line of geometry.

import "math"

// mat4 is a 4x4 matrix in column-major order (m[col*4+row]), applied to column
// vectors: v' = M * v.
type mat4 [16]float64

func identMat() mat4 { return mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }

func mulMat(a, b mat4) mat4 {
	var r mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			r[col*4+row] = a[0*4+row]*b[col*4+0] + a[1*4+row]*b[col*4+1] +
				a[2*4+row]*b[col*4+2] + a[3*4+row]*b[col*4+3]
		}
	}
	return r
}

func transMat(t [3]float64) mat4 {
	m := identMat()
	m[12], m[13], m[14] = t[0], t[1], t[2]
	return m
}

func scaleMat(s [3]float64) mat4 {
	return mat4{s[0], 0, 0, 0, 0, s[1], 0, 0, 0, 0, s[2], 0, 0, 0, 0, 1}
}

// quatMat builds a rotation matrix from a quaternion (x,y,z,w).
func quatMat(q [4]float64) mat4 {
	x, y, z, w := q[0], q[1], q[2], q[3]
	n := math.Sqrt(x*x + y*y + z*z + w*w)
	if n == 0 {
		return identMat()
	}
	x, y, z, w = x/n, y/n, z/n, w/n
	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	wx, wy, wz := w*x, w*y, w*z
	return mat4{
		1 - 2*(yy+zz), 2 * (xy + wz), 2 * (xz - wy), 0,
		2 * (xy - wz), 1 - 2*(xx+zz), 2 * (yz + wx), 0,
		2 * (xz + wy), 2 * (yz - wx), 1 - 2*(xx+yy), 0,
		0, 0, 0, 1,
	}
}

// xformPoint applies M to a point (w=1).
func xformPoint(m mat4, p [3]float32) [3]float32 {
	x, y, z := float64(p[0]), float64(p[1]), float64(p[2])
	return [3]float32{
		float32(m[0]*x + m[4]*y + m[8]*z + m[12]),
		float32(m[1]*x + m[5]*y + m[9]*z + m[13]),
		float32(m[2]*x + m[6]*y + m[10]*z + m[14]),
	}
}

// xformDir applies M's rotation/scale to a direction (w=0).
func xformDir(m mat4, p [3]float32) [3]float32 {
	x, y, z := float64(p[0]), float64(p[1]), float64(p[2])
	return [3]float32{
		float32(m[0]*x + m[4]*y + m[8]*z),
		float32(m[1]*x + m[5]*y + m[9]*z),
		float32(m[2]*x + m[6]*y + m[10]*z),
	}
}

// boneWorldMatrices composes each bone's world matrix for the Stand pose.
// A bone's local transform pivots about its rest point:
//
//	L = T(pivot) * T(trans) * R(rot) * S(scale) * T(-pivot)
//
// and its world matrix is parentWorld * L (root bones use identity). The matrix
// maps a bind-pose model-space point to its posed position, so it also posing
// attachment points that ride the bone.
func (m *M2Model) boneWorldMatrices() []mat4 {
	world := make([]mat4, len(m.Bones))
	computed := make([]bool, len(m.Bones))
	var compute func(i int) mat4
	compute = func(i int) mat4 {
		if computed[i] {
			return world[i]
		}
		b := m.Bones[i]
		piv := [3]float64{float64(b.Pivot[0]), float64(b.Pivot[1]), float64(b.Pivot[2])}
		local := mulMat(transMat(piv),
			mulMat(transMat([3]float64{float64(b.Trans[0]), float64(b.Trans[1]), float64(b.Trans[2])}),
				mulMat(quatMat([4]float64{float64(b.Rot[0]), float64(b.Rot[1]), float64(b.Rot[2]), float64(b.Rot[3])}),
					mulMat(scaleMat([3]float64{float64(b.Scale[0]), float64(b.Scale[1]), float64(b.Scale[2])}),
						transMat([3]float64{-piv[0], -piv[1], -piv[2]})))))
		w := local
		// Guard against an out-of-range or self-referential parent (corrupt data).
		if b.Parent >= 0 && int(b.Parent) < len(m.Bones) && int(b.Parent) != i {
			w = mulMat(compute(int(b.Parent)), local)
		}
		world[i] = w
		computed[i] = true
		return w
	}
	for i := range m.Bones {
		compute(i)
	}
	return world
}

// hasStandPose reports whether any bone carries a non-identity Stand transform —
// i.e. whether posing would change anything. Lets the common case (bind == stand)
// skip skinning entirely.
func (m *M2Model) hasStandPose() bool {
	for _, b := range m.Bones {
		if b.Trans != [3]float32{0, 0, 0} ||
			b.Rot != [4]float32{0, 0, 0, 1} ||
			b.Scale != [3]float32{1, 1, 1} {
			return true
		}
	}
	return false
}

// PoseToStand skins the model into its Stand pose in place: it transforms each
// vertex by its weighted bone world matrices, recomputes the bounding box, and
// poses attachment points so riders (shoulders, weapons) follow the body. It is
// a no-op when the model has no bones or stands in its bind pose, and is
// idempotent — calling it twice has no further effect (it clears the bone TRS
// once applied).
func (m *M2Model) PoseToStand() {
	if len(m.Bones) == 0 || !m.hasStandPose() {
		return
	}
	world := m.boneWorldMatrices()

	for i := range m.Vertices {
		v := &m.Vertices[i]
		var sumW float64
		for k := 0; k < 4; k++ {
			sumW += float64(v.Wts[k])
		}
		if sumW == 0 {
			continue // unrigged vertex stays at its bind position
		}
		var pos, nrm [3]float32
		for k := 0; k < 4; k++ {
			w := float64(v.Wts[k])
			if w == 0 {
				continue
			}
			bi := int(v.Bones[k])
			if bi >= len(world) {
				continue
			}
			f := float32(w / sumW)
			p := xformPoint(world[bi], v.Pos)
			n := xformDir(world[bi], v.Normal)
			pos[0] += p[0] * f
			pos[1] += p[1] * f
			pos[2] += p[2] * f
			nrm[0] += n[0] * f
			nrm[1] += n[1] * f
			nrm[2] += n[2] * f
		}
		v.Pos = pos
		// Renormalize the skinned normal (weighted rotations don't preserve length).
		if l := float32(math.Sqrt(float64(nrm[0]*nrm[0] + nrm[1]*nrm[1] + nrm[2]*nrm[2]))); l > 1e-6 {
			v.Normal = [3]float32{nrm[0] / l, nrm[1] / l, nrm[2] / l}
		}
	}

	// Pose attachment points (they ride a bone, and AttachmentPos is consumed by
	// the renderer to place shoulders/weapons).
	for i := range m.Attachments {
		bi := int(m.Attachments[i].Bone)
		if bi < len(world) {
			m.Attachments[i].Position = xformPoint(world[bi], m.Attachments[i].Position)
		}
	}

	// Recompute bounds from the posed geometry so framing fits the new silhouette.
	first := true
	for _, v := range m.Vertices {
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

	// Mark as applied: clear the TRS so a second call is a no-op.
	for i := range m.Bones {
		m.Bones[i].Trans = [3]float32{0, 0, 0}
		m.Bones[i].Rot = [4]float32{0, 0, 0, 1}
		m.Bones[i].Scale = [3]float32{1, 1, 1}
	}
}
