package main

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// The cube's size and where it stands, in metres: a box you could hold in both arms, an arm and a
// half ahead of where the player first looked.
const (
	cubeSize     = 0.3
	cubeDistance = 1.5
	handCubeSize = 0.05 // small enough to sit inside a controller's ring
)

// cubeTurn turns the cube a little about two axes, so three of its faces face the player and the
// light shades each differently: a cube seen square-on looks like a flat square.
var cubeTurn = graphics.Rotate(graphics.Vector{0, 1, 0, 0}, math.Pi/5).Multiply(graphics.Rotate(graphics.Vector{1, 0, 0, 0}, math.Pi/8))

// world is the program's scene and the nodes it moves.
type world struct {
	scene  *scene.Scene
	cube   *scene.Node
	hands  [2]*scene.Node // left, right
	placed bool           // the cube was put in front of the player
}

// newWorld makes the scene: the cube, hidden until it is placed, and a small cube for each hand,
// hidden until the hand is tracked. One metre is one unit of the scene.
func newWorld() *world {
	made := &world{scene: scene.New()}
	cube := boxMesh("cube", cubeSize, graphics.Vector{0.85, 0.45, 0.2, 1})
	made.cube = &scene.Node{Name: "cube", Transform: graphics.Identity(), Mesh: cube, Hidden: true}
	made.scene.Nodes = append(made.scene.Nodes, made.cube)
	colours := [2]graphics.Vector{{0.25, 0.55, 0.9, 1}, {0.3, 0.8, 0.45, 1}}
	for index, colour := range colours {
		hand := &scene.Node{Name: "hand", Transform: graphics.Identity(), Mesh: boxMesh("hand", handCubeSize, colour),
			Hidden: true, Moving: true}
		made.hands[index] = hand
		made.scene.Nodes = append(made.scene.Nodes, hand)
	}
	return made
}

// place puts the cube cubeDistance ahead of the head, at its height, the first time it is asked
// with a tracked head: then it stays there in the room however the head moves, which is the whole
// point. An untracked head is a guess (or the origin, at floor level), so the cube waits for a
// tracked one rather than stand where the head is not. "Ahead" is along the player's space's -Z,
// the way the player faced as the program started.
func (made *world) place(rig camera.Rig, head graphics.Pose, tracked bool) {
	if made.placed || !tracked {
		return
	}
	spot := graphics.Pose{Orientation: graphics.IdentityQuaternion(),
		Position: graphics.Vector{head.Position[0], head.Position[1], head.Position[2] - cubeDistance, 1}}
	made.cube.Transform = rig.Place(spot).Multiply(cubeTurn)
	made.cube.Hidden = false
	made.placed = true
}

// follow puts a small cube on each tracked hand, where it is held, and hides an untracked one.
func (made *world) follow(rig camera.Rig, grips [2]graphics.Pose, tracked [2]bool) {
	for index, hand := range made.hands {
		hand.Hidden = !tracked[index]
		if tracked[index] {
			hand.Transform = rig.Place(grips[index])
		}
	}
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
