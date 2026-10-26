// Package explore walks a player through a scene in the headset, at room scale: the left stick
// walks the way the player looks, the right stick turns in steps (smooth turning makes some people
// queasy; a program may ask for it), walking in the room walks in the scene too, and Y or B makes the player a giant, with
// the whole scene a model at their feet, and back again. A small cube sits in each controller's
// grip. Any scene.Scene can be explored; a program only says which, and where the player starts.
package explore

import (
	"github.com/akrck02/go-meta-quest/place"
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
	"github.com/akrck02/go-renderer/scene"
)

// The two sizes the player can be, in scene units a metre: a person, and by default a giant to
// whom a whole island is a model the size of a room.
const (
	PersonScale       = 1
	DefaultGiantScale = 20
)

// The nearest and farthest the eyes see. The near plane, in metres for a person (a hand held close
// to the face), grows with the player's scale: a giant's hands are further away in the scene. The
// far one is the scene's, in its units, and stays.
const (
	NearMetres = 0.05
	DefaultFar = 1500
)

// handCubeSize is the side of the cube in each hand, in metres: small enough to sit inside a
// controller's ring.
const handCubeSize = 0.05

// Options say what an exploration starts from.
type Options struct {
	// ApplicationName is the name the program gives the OpenXR runtime.
	ApplicationName string
	// Start is where the room stands in the scene at the start; a Scale of 0 is a person.
	Start camera.Rig
	// GiantScale is how big Y or B makes the player (0 = DefaultGiantScale).
	GiantScale float32
	// Far is the farthest the eyes see, in scene units (0 = DefaultFar).
	Far float32
	// WalkMetresSecond is how far the left stick all the way walks in a second, in the player's
	// own metres (0 = camera.DefaultRigWalkSpeedMetres): faster for a long scene.
	WalkMetresSecond float32
	// TurnRadiansSecond, when set, turns the player smoothly about their head with the right
	// stick, as fast as this all the way over, instead of in steps.
	TurnRadiansSecond float64
	// Update, when set, is called every frame after the sticks have moved the player, with the
	// controllers as a game pad: a program's own buttons, such as one that shows another scene.
	Update func(explorer *Explorer, state *input.State)
}

// Explorer is an exploration's state: the scene, where the room stands in it, how the sticks move
// it, the hands' cubes, and the eyes' pictures, kept from frame to frame so a frame makes nothing
// new.
type Explorer struct {
	world      *scene.Scene
	rig        camera.Rig
	walk       camera.RigWalk
	giantScale float32
	far        float32
	turnSpeed  float64        // radians a second with the right stick all the way over; 0 turns in steps
	hands      [2]*scene.Node // left, right
	views      []gles.View
	poses      []graphics.Pose
	fields     []graphics.FieldOfView
}

// New puts the player in the scene where the options say, with the sticks walking on its ground and
// a small cube in each hand, hidden until the hand is tracked. The hands' cubes are added to the
// scene.
func New(world *scene.Scene, options Options) *Explorer {
	made := &Explorer{world: world, rig: options.Start, giantScale: options.GiantScale, far: options.Far, turnSpeed: options.TurnRadiansSecond,
		views: make([]gles.View, 2), poses: make([]graphics.Pose, 2), fields: make([]graphics.FieldOfView, 2)}
	if made.rig.Scale == 0 {
		made.rig.Scale = PersonScale
	}
	if made.giantScale == 0 {
		made.giantScale = DefaultGiantScale
	}
	if made.far == 0 {
		made.far = DefaultFar
	}
	made.walk = camera.RigWalk{Ground: world.GroundHeight, SpeedMetres: options.WalkMetresSecond}
	colours := [2]graphics.Vector{{0.25, 0.55, 0.9, 1}, {0.3, 0.8, 0.45, 1}}
	for index, colour := range colours {
		hand := &scene.Node{Name: "hand", Transform: graphics.Identity(), Mesh: boxMesh("hand", handCubeSize, colour),
			Hidden: true, Moving: true}
		made.hands[index] = hand
		world.Nodes = append(world.Nodes, hand)
	}
	return made
}

// Rig is where the room stands in the scene now.
func (made *Explorer) Rig() camera.Rig { return made.rig }

// World is the scene explored now.
func (made *Explorer) World() *scene.Scene { return made.world }

// Show explores another scene instead, with the room where start says (a Scale of 0 is a person):
// the sticks walk on its ground, and the hands' cubes go with the player.
func (made *Explorer) Show(world *scene.Scene, start camera.Rig) {
	made.world = world
	made.rig = start
	if made.rig.Scale == 0 {
		made.rig.Scale = PersonScale
	}
	made.walk.Ground = world.GroundHeight
	for _, hand := range made.hands {
		world.Nodes = append(world.Nodes, hand)
	}
}

// Move puts the room somewhere else in the same scene, at any size (a Scale of 0 is a person): a
// jump to a place, or a giant of the program's own size. Unlike Show, the scene and the hands stay.
func (made *Explorer) Move(rig camera.Rig) {
	made.rig = rig
	if made.rig.Scale == 0 {
		made.rig.Scale = PersonScale
	}
}

// Steer applies the controllers to the rig: Y or B switches between a person and a giant, with the
// head over the same point of the ground; the sticks walk and turn. An untracked head is a guess,
// so the size waits for a tracked one rather than move the player to where the head is not. A
// program whose right stick scrolls a panel the player points at feeds it 0 sideways meanwhile, so
// the player does not turn as they scroll.
func (made *Explorer) Steer(state *input.State, seconds float64, head graphics.Pose, headTracked bool) {
	if state == nil {
		made.walk.Update(nil, seconds, head, &made.rig)
		return
	}
	if headTracked && (state.PadPressed(input.PadY) || state.PadPressed(input.PadB)) {
		made.rig = rescale(made.rig, head, made.otherScale(made.rig.Scale), made.walk.Ground)
	}
	if made.turnSpeed == 0 {
		made.walk.Update(state, seconds, head, &made.rig)
		return
	}
	turnX, turnY := state.RightStick()
	state.Feed().SetRightStick(0, turnY) // no steps: the turn is the smooth one
	made.walk.Update(state, seconds, head, &made.rig)
	state.Feed().SetRightStick(turnX, turnY)
	if turnX != 0 {
		made.rig = place.TurnedAround(made.rig, head, float32(-turnX*made.turnSpeed*seconds)) // pushing right turns clockwise
	}
}

// Follow puts a small cube on each tracked hand, where it is held, and hides an untracked one.
func (made *Explorer) Follow(grips [2]graphics.Pose, tracked [2]bool) {
	for index, hand := range made.hands {
		hand.Hidden = !tracked[index]
		if tracked[index] {
			hand.Transform = made.rig.Place(grips[index])
		}
	}
}

// Views fills the views (and the poses and fields the covering lens is made from) with the eyes
// seen from the rig, and returns them with the covering lens and its aspect, ready for
// gles.SceneRenderer.DrawViews. It reuses the explorer's slices: nothing is made.
func (made *Explorer) Views(eyes [2]openxr.Eye) ([]gles.View, camera.Lens, float32) {
	near := made.near()
	for index, seen := range eyes {
		made.poses[index], made.fields[index] = seen.Pose, seen.Field
		made.views[index] = gles.View{Target: seen.Target, Width: seen.Width, Height: seen.Height,
			View:       made.rig.EyeView(seen.Pose),
			Projection: graphics.PerspectiveFromFieldOfView(seen.Field, near, made.far),
			Eye:        made.rig.EyePosition(seen.Pose)}
	}
	covering, aspect := camera.CoveringLens(made.rig, made.poses, made.fields, near, made.far)
	return made.views, covering, aspect
}

// near is the near plane in scene units at the rig's scale.
func (made *Explorer) near() float32 {
	scale := made.rig.Scale
	if scale == 0 {
		scale = PersonScale
	}
	return NearMetres * scale
}

// otherScale is the size the player becomes when switching from a scale: a person becomes a
// giant, and anything else a person again.
func (made *Explorer) otherScale(scale float32) float32 {
	if scale == PersonScale || scale == 0 {
		return made.giantScale
	}
	return PersonScale
}

// rescale returns the rig at another scale with the head over the same point of the ground: only
// the size changes, not where the player is. The floor goes to the ground under the head, where
// there is ground; else it stays at its height.
func rescale(rig camera.Rig, head graphics.Pose, scale float32, ground camera.GroundFunc) camera.Rig {
	headInWorld := rig.EyePosition(head)
	rescaled := rig
	rescaled.Scale = scale
	rescaled.Position = graphics.Vector{0, 0, 0, 1}
	headFromFloor := rescaled.EyePosition(head) // the head from the room's origin, turned and scaled
	rescaled.Position = graphics.Vector{headInWorld[0] - headFromFloor[0], rig.Position[1], headInWorld[2] - headFromFloor[2], 1}
	if ground == nil {
		return rescaled
	}
	if height, found := ground(headInWorld[0], headInWorld[2]); found {
		rescaled.Position[1] = height
	}
	return rescaled
}

// boxMesh is a cube of a size, centred on its node, lit and of one colour.
func boxMesh(name string, size float32, colour graphics.Vector) *scene.Mesh {
	half := size / 2
	// each face's corners, counter-clockwise seen from outside
	faces := [6][4][3]float32{
		{{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1}},     // front
		{{1, -1, -1}, {-1, -1, -1}, {-1, 1, -1}, {1, 1, -1}}, // back
		{{-1, -1, -1}, {-1, -1, 1}, {-1, 1, 1}, {-1, 1, -1}}, // left
		{{1, -1, 1}, {1, -1, -1}, {1, 1, -1}, {1, 1, 1}},     // right
		{{-1, 1, 1}, {1, 1, 1}, {1, 1, -1}, {-1, 1, -1}},     // top
		{{-1, -1, -1}, {1, -1, -1}, {1, -1, 1}, {-1, -1, 1}}, // bottom
	}
	mesh := &scene.Mesh{Name: name, Material: &scene.Material{Name: name, Kind: scene.LitKind, BaseColor: colour}}
	for _, face := range faces {
		for _, corner := range [6]int{0, 1, 2, 0, 2, 3} {
			point := face[corner]
			mesh.Positions = append(mesh.Positions, point[0]*half, point[1]*half, point[2]*half)
		}
	}
	mesh.ComputeNormals()
	return mesh
}
