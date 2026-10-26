package pointer

import (
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/openxr"
	"github.com/akrck02/go-renderer/scene"
)

// beamThickness is the beam's side, in metres: thin as a thread, still seen at arm's length.
const beamThickness = 0.003

// DefaultBeamLength is how far the beam reaches, in metres, when it meets nothing.
const DefaultBeamLength = 3

// Beam is a thin bar from a controller along where it aims, as long as the ray goes to what it
// meets: the person sees what they point at. It is a scene node; add Node to the scene once.
type Beam struct {
	Node *scene.Node
	// Length is how far it reaches when it meets nothing (0 = DefaultBeamLength).
	Length float32
}

// NewBeam makes a beam of a colour, hidden until it is placed on a tracked hand. It is unlit: a
// line of light, the same from every side.
func NewBeam(colour graphics.Vector) *Beam {
	return &Beam{Node: &scene.Node{Name: "beam", Transform: graphics.Identity(), Mesh: barMesh(colour), Hidden: true, Moving: true}}
}

// Place puts the beam on the hand's aim, through the rig (where the room stands in the scene), as
// long as distance (metres, what the ray meets) or the beam's own Length when distance is 0. An
// untracked hand hides it.
func (beam *Beam) Place(rig camera.Rig, hand openxr.Hand, distance float32) {
	beam.Node.Hidden = !hand.Tracked
	if !hand.Tracked {
		return
	}
	if distance <= 0 {
		distance = beam.Length
	}
	if distance <= 0 {
		distance = DefaultBeamLength
	}
	// the bar runs from 0 to -1 along z: stretched to the distance, thin across
	stretch := graphics.Scale(graphics.Vector{beamThickness, beamThickness, distance, 0})
	beam.Node.Transform = rig.Place(hand.Aim).Multiply(stretch)
}

// barMesh is a box one unit long from z 0 to z -1 and one unit across, of one unlit colour.
func barMesh(colour graphics.Vector) *scene.Mesh {
	faces := [6][4][3]float32{
		{{-1, -1, 0}, {1, -1, 0}, {1, 1, 0}, {-1, 1, 0}},     // the near end
		{{1, -1, -1}, {-1, -1, -1}, {-1, 1, -1}, {1, 1, -1}}, // the far end
		{{-1, -1, -1}, {-1, -1, 0}, {-1, 1, 0}, {-1, 1, -1}},
		{{1, -1, 0}, {1, -1, -1}, {1, 1, -1}, {1, 1, 0}},
		{{-1, 1, 0}, {1, 1, 0}, {1, 1, -1}, {-1, 1, -1}},
		{{-1, -1, -1}, {1, -1, -1}, {1, -1, 0}, {-1, -1, 0}},
	}
	mesh := &scene.Mesh{Name: "beam", Material: &scene.Material{Name: "beam", Kind: scene.UnlitKind, BaseColor: colour}}
	for _, face := range faces {
		for _, corner := range [6]int{0, 1, 2, 0, 2, 3} {
			point := face[corner]
			mesh.Positions = append(mesh.Positions, point[0]/2, point[1]/2, point[2])
		}
	}
	mesh.ComputeNormals()
	return mesh
}
