package main

import (
	"math"
	"testing"

	"github.com/akrck02/go-meta-quest/explore"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

func nearlyEqual(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-3 }

func TestThePlayerStartsOnTheGroundSouthOfTheSquare(test *testing.T) {
	world := sample.Island()
	square, found := nodePosition(world, startNodeName)
	if !found {
		test.Fatal("the sample island has no square")
	}
	rig := startRig(world)
	if !nearlyEqual(rig.Position[0], square[0]) || !nearlyEqual(rig.Position[2], square[2]+startDistance) {
		test.Fatalf("the start is at %v, want %v metres south of the square at %v", rig.Position, startDistance, square)
	}
	ground, found := world.GroundHeight(rig.Position[0], rig.Position[2])
	if !found || !nearlyEqual(rig.Position[1], ground) {
		test.Fatalf("the floor is at height %v, want the ground's %v", rig.Position[1], ground)
	}
	if ground <= 1 {
		test.Fatalf("the start is %v above the sea: on the beach or in the water, not in the village", ground)
	}
	if rig.Scale != explore.PersonScale || rig.YawRadians != 0 {
		test.Fatalf("the start rig is %+v, want a person facing north", rig)
	}
}

func TestWithoutASquareThePlayerStartsInTheMiddle(test *testing.T) {
	rig := startRig(scene.New())
	if rig.Position != (graphics.Vector{0, 0, 0, 1}) || rig.Scale != explore.PersonScale {
		test.Fatalf("the start rig is %+v, want a person at the origin", rig)
	}
}
