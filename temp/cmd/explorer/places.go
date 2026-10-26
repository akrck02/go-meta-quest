package main

import (
	"github.com/akrck02/go-meta-quest/explore"
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// Where the player starts: on the ground a short walk south of the village square, facing north,
// so the first thing seen is the square with the houses behind it. When the scene has no square,
// the player starts on the ground at the middle of the world.
const (
	startNodeName = "square"
	startDistance = 12 // metres south (+Z) of the square
)

// startRig is the rig at the start: the room's floor on the ground startDistance south of the
// square, not turned, so the way the player faces at the start (-Z) looks at the square, at a
// person's size.
func startRig(world *scene.Scene) camera.Rig {
	spot := graphics.Vector{0, 0, 0, 1}
	if square, found := nodePosition(world, startNodeName); found {
		spot = graphics.Vector{square[0], 0, square[2] + startDistance, 1}
	}
	if ground, found := world.GroundHeight(spot[0], spot[2]); found {
		spot[1] = ground
	}
	return camera.Rig{Position: spot, Scale: explore.PersonScale}
}

// nodePosition is where the first node of a name stands in the world: its origin, moved by its
// own transform and its parents'.
func nodePosition(world *scene.Scene, name string) (graphics.Vector, bool) {
	var position graphics.Vector
	found := false
	world.Walk(func(node *scene.Node, transform graphics.Matrix) {
		if found || node.Name != name {
			return
		}
		position, found = transform.Apply(graphics.Vector{0, 0, 0, 1}), true
	})
	return position, found
}
