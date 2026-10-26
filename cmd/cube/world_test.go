package main

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
)

func nearlyEqual(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-4 }

func standingHead() graphics.Pose {
	return graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 1.6, 0, 1}}
}

func TestTheCubeIsHiddenUntilPlaced(test *testing.T) {
	made := newWorld()
	if !made.cube.Hidden {
		test.Fatal("the cube shows before it was placed")
	}
}

func TestTheCubeStandsAheadAtEyeHeight(test *testing.T) {
	made := newWorld()
	made.place(camera.Rig{}, standingHead(), true)
	middle := made.cube.Transform.Apply(graphics.Vector{0, 0, 0, 1})
	if !nearlyEqual(middle[0], 0) || !nearlyEqual(middle[1], 1.6) || !nearlyEqual(middle[2], -cubeDistance) {
		test.Fatalf("the cube's middle is at %v, want 0, 1.6, %v", middle, -cubeDistance)
	}
	if made.cube.Hidden {
		test.Fatal("a placed cube is still hidden")
	}
}

func TestTheCubeStaysWhereItWasPlaced(test *testing.T) {
	made := newWorld()
	made.place(camera.Rig{}, standingHead(), true)
	before := made.cube.Transform
	moved := standingHead()
	moved.Position = graphics.Vector{0.5, 1.2, 0.3, 1}
	made.place(camera.Rig{}, moved, true)
	if made.cube.Transform != before {
		test.Fatal("the cube moved with the head")
	}
}

func TestTheCubeWaitsForATrackedHead(test *testing.T) {
	made := newWorld()
	made.place(camera.Rig{}, standingHead(), false)
	if made.placed || !made.cube.Hidden {
		test.Fatal("the cube was placed by an untracked head")
	}
	made.place(camera.Rig{}, standingHead(), true)
	if !made.placed || made.cube.Hidden {
		test.Fatal("a tracked head did not place the cube")
	}
}

func TestTheCubeIsItsSize(test *testing.T) {
	mesh := boxMesh("cube", cubeSize, graphics.Vector{1, 1, 1, 1})
	lowest, highest := float32(math.Inf(1)), float32(math.Inf(-1))
	for index := 0; index < mesh.VertexCount(); index++ {
		x := mesh.Positions[3*index]
		lowest, highest = min(lowest, x), max(highest, x)
	}
	if !nearlyEqual(highest-lowest, cubeSize) {
		test.Fatalf("the cube is %v wide, want %v", highest-lowest, cubeSize)
	}
	if mesh.VertexCount() != 36 || len(mesh.Normals) != len(mesh.Positions) {
		test.Fatalf("got %d vertices and %d normal values, want 36 and one normal each", mesh.VertexCount(), len(mesh.Normals)/3)
	}
}

func TestTheCubesFacesPointOutwards(test *testing.T) {
	mesh := boxMesh("cube", 1, graphics.Vector{1, 1, 1, 1})
	for index := 0; index < mesh.VertexCount(); index++ {
		position := graphics.Vector{mesh.Positions[3*index], mesh.Positions[3*index+1], mesh.Positions[3*index+2], 0}
		normal := graphics.Vector{mesh.Normals[3*index], mesh.Normals[3*index+1], mesh.Normals[3*index+2], 0}
		if position.Dot(normal) <= 0 {
			test.Fatalf("vertex %d at %v has its normal %v pointing in", index, position, normal)
		}
	}
}

func TestHandCubesFollowTrackedHandsOnly(test *testing.T) {
	made := newWorld()
	grips := [2]graphics.Pose{
		{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{-0.2, 1, -0.3, 1}},
		{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0.2, 1, -0.3, 1}},
	}
	made.follow(camera.Rig{}, grips, [2]bool{false, true})
	if !made.hands[0].Hidden {
		test.Fatal("the untracked left hand's cube shows")
	}
	if made.hands[1].Hidden {
		test.Fatal("the tracked right hand's cube is hidden")
	}
	middle := made.hands[1].Transform.Apply(graphics.Vector{0, 0, 0, 1})
	if !nearlyEqual(middle[0], 0.2) || !nearlyEqual(middle[1], 1) || !nearlyEqual(middle[2], -0.3) {
		test.Fatalf("the right hand's cube is at %v, want 0.2, 1, -0.3", middle)
	}
}
