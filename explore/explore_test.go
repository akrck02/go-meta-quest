package explore

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

func nearlyEqual(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-3 }

// headAt is a standing head somewhere in the room, looking a little to the left.
func headAt(x, z float32) graphics.Pose {
	return graphics.Pose{Orientation: graphics.QuaternionAroundY(0.4), Position: graphics.Vector{x, 1.6, z, 1}}
}

// flatGround is ground everywhere at one height.
func flatGround(height float32) camera.GroundFunc {
	return func(x, z float32) (float32, bool) { return height, true }
}

func TestTheOptionsDefaultsAreAPersonAGiantAndTheFarDistance(test *testing.T) {
	made := New(scene.New(), Options{})
	if made.Rig().Scale != PersonScale || made.giantScale != DefaultGiantScale || made.far != DefaultFar {
		test.Fatalf("the defaults are scale %v, giant %v, far %v", made.Rig().Scale, made.giantScale, made.far)
	}
	start := camera.Rig{Position: graphics.Vector{3, 1, 4, 1}, YawRadians: 1, Scale: 2}
	made = New(scene.New(), Options{Start: start, GiantScale: 50, Far: 90})
	if made.Rig() != start || made.giantScale != 50 || made.far != 90 {
		test.Fatalf("the options were not kept: rig %+v, giant %v, far %v", made.Rig(), made.giantScale, made.far)
	}
}

func TestBecomingAGiantKeepsTheHeadOverTheSamePoint(test *testing.T) {
	rig := camera.Rig{Position: graphics.Vector{10, 2, -5, 1}, YawRadians: 0.7, Scale: PersonScale}
	head := headAt(0.8, -0.6) // the player walked away from the room's origin
	before := rig.EyePosition(head)
	giant := rescale(rig, head, DefaultGiantScale, flatGround(3))
	after := giant.EyePosition(head)
	if !nearlyEqual(after[0], before[0]) || !nearlyEqual(after[2], before[2]) {
		test.Fatalf("the head moved from %v to %v", before, after)
	}
	if giant.Scale != DefaultGiantScale || giant.YawRadians != rig.YawRadians {
		test.Fatalf("the giant's rig is %+v, want the same turn at scale %v", giant, DefaultGiantScale)
	}
	if giant.Position[1] != 3 || !nearlyEqual(after[1], 3+1.6*DefaultGiantScale) {
		test.Fatalf("the giant's floor is at %v and head at %v, want the floor on the ground at 3", giant.Position[1], after[1])
	}
	person := rescale(giant, head, PersonScale, flatGround(3))
	if back := person.EyePosition(head); !nearlyEqual(back[0], before[0]) || !nearlyEqual(back[2], before[2]) {
		test.Fatalf("back to a person, the head is at %v, want over %v", back, before)
	}
}

func TestRescalingOverTheVoidKeepsTheFloorsHeight(test *testing.T) {
	rig := camera.Rig{Position: graphics.Vector{0, 4, 0, 1}, Scale: PersonScale}
	nowhere := func(x, z float32) (float32, bool) { return 0, false }
	if giant := rescale(rig, headAt(0, 0), DefaultGiantScale, nowhere); giant.Position[1] != 4 {
		test.Fatalf("the floor went to %v with no ground under it, want it to stay at 4", giant.Position[1])
	}
	if giant := rescale(rig, headAt(0, 0), DefaultGiantScale, nil); giant.Position[1] != 4 {
		test.Fatalf("the floor went to %v without a ground, want it to stay at 4", giant.Position[1])
	}
}

func TestTheScaleSwitchesBetweenAPersonAndAGiant(test *testing.T) {
	made := New(scene.New(), Options{GiantScale: 30})
	for scale, want := range map[float32]float32{0: 30, PersonScale: 30, 30: PersonScale, 7: PersonScale} {
		if got := made.otherScale(scale); got != want {
			test.Fatalf("from %v the other scale is %v, want %v", scale, got, want)
		}
	}
}

func TestYMakesAGiantOnlyWithATrackedHead(test *testing.T) {
	made := New(sample.Island(), Options{})
	state := input.NewState()
	state.Feed().SetPadButton(input.PadY, true)
	made.Steer(state, 0, headAt(0, 0), false)
	if made.Rig().Scale != PersonScale {
		test.Fatal("an untracked head switched the scale")
	}
	made.Steer(state, 0, headAt(0, 0), true)
	if made.Rig().Scale != DefaultGiantScale {
		test.Fatalf("the scale is %v after Y, want %v", made.Rig().Scale, DefaultGiantScale)
	}
	state.Feed().EndFrame()
	made.Steer(state, 0, headAt(0, 0), true)
	if made.Rig().Scale != DefaultGiantScale {
		test.Fatal("holding Y switched the scale again")
	}
}

func TestTheStickWalksOnTheGround(test *testing.T) {
	world := sample.Island()
	made := New(world, Options{Start: camera.Rig{Position: graphics.Vector{0, 0, 0, 1}}})
	start := made.Rig().Position
	state := input.NewState()
	state.Feed().SetStick(0, 1)
	for range 60 {
		made.Steer(state, 1.0/60, graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 1.6, 0, 1}}, true)
	}
	if !nearlyEqual(made.Rig().Position[2], start[2]-camera.DefaultRigWalkSpeedMetres) {
		test.Fatalf("a second of a full stick moved the room from %v to %v, want %v metres north", start, made.Rig().Position, camera.DefaultRigWalkSpeedMetres)
	}
	if ground, found := world.GroundHeight(made.Rig().Position[0], made.Rig().Position[2]); found && !nearlyEqual(made.Rig().Position[1], ground) {
		test.Fatalf("the floor is at %v, want on the ground at %v", made.Rig().Position[1], ground)
	}
}

func TestHandCubesFollowTrackedHandsOnly(test *testing.T) {
	world := scene.New()
	made := New(world, Options{})
	if len(world.Nodes) != 2 {
		test.Fatalf("the scene has %d nodes, want the two hands' cubes", len(world.Nodes))
	}
	grips := [2]graphics.Pose{
		{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{-0.2, 1, -0.3, 1}},
		{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0.2, 1, -0.3, 1}},
	}
	made.Follow(grips, [2]bool{false, true})
	if !made.hands[0].Hidden || made.hands[1].Hidden {
		test.Fatal("want the untracked left hand's cube hidden and the tracked right one shown")
	}
	middle := made.hands[1].Transform.Apply(graphics.Vector{0, 0, 0, 1})
	if !nearlyEqual(middle[0], 0.2) || !nearlyEqual(middle[1], 1) || !nearlyEqual(middle[2], -0.3) {
		test.Fatalf("the right hand's cube is at %v, want 0.2, 1, -0.3", middle)
	}
}

func TestTheNearPlaneGrowsWithTheGiant(test *testing.T) {
	made := New(scene.New(), Options{})
	if made.near() != NearMetres {
		test.Fatalf("a person's near plane is %v, want %v", made.near(), NearMetres)
	}
	made.rig.Scale = DefaultGiantScale
	if !nearlyEqual(made.near(), NearMetres*DefaultGiantScale) {
		test.Fatalf("a giant's near plane is %v, want %v", made.near(), NearMetres*DefaultGiantScale)
	}
}

func TestTheViewsAreTheEyesSeenFromTheRig(test *testing.T) {
	made := New(sample.Island(), Options{Far: 800})
	field := graphics.FieldOfView{LeftRadians: -0.8, RightRadians: 0.8, UpRadians: 0.8, DownRadians: -0.8}
	var eyes [2]openxr.Eye
	for index, side := range []float32{-0.032, 0.032} {
		eyes[index] = openxr.Eye{Pose: graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{side, 1.6, 0, 1}},
			Field: field, Target: 7, Width: 100, Height: 80}
	}
	views, covering, aspect := made.Views(eyes)
	for index, view := range views {
		if want := made.Rig().EyePosition(eyes[index].Pose); view.Eye != want || view.Target != 7 || view.Width != 100 {
			test.Fatalf("view %d is %+v, want the eye at %v", index, view, want)
		}
	}
	if aspect <= 0 || covering.Near > made.near() || covering.Far < 800 {
		test.Fatalf("the covering lens %+v (aspect %v) does not cover the eyes' %v to %v", covering, aspect, made.near(), 800)
	}
}

func TestSteeringMakesNothingNew(test *testing.T) {
	made := New(sample.Island(), Options{})
	state := input.NewState()
	state.Feed().SetStick(0.5, 1)
	head := headAt(0.2, -0.1)
	grips := [2]graphics.Pose{head, head}
	made.Steer(state, 1.0/72, head, true) // the ground is indexed on the first call
	allocations := testing.AllocsPerRun(100, func() {
		made.Steer(state, 1.0/72, head, true)
		made.Follow(grips, [2]bool{true, true})
	})
	if allocations != 0 {
		test.Fatalf("a frame's steering made %v allocations, want none", allocations)
	}
}

func TestShowingAnotherSceneTakesTheHandsAndStartsThere(test *testing.T) {
	first, second := scene.New(), scene.New()
	made := New(first, Options{})
	start := camera.Rig{Position: graphics.Vector{40, 0, 20, 1}, YawRadians: 0.5}
	made.Show(second, start)
	if made.World() != second || len(second.Nodes) != 2 || second.Nodes[0] != made.hands[0] {
		test.Fatalf("the second scene has %d nodes, want the two hands' cubes", len(second.Nodes))
	}
	if made.Rig().Position != start.Position || made.Rig().YawRadians != 0.5 || made.Rig().Scale != PersonScale {
		test.Fatalf("the rig is %+v, want the start as a person", made.Rig())
	}
	ground := scene.New()
	ground.Nodes = []*scene.Node{{Mesh: &scene.Mesh{Ground: true, Positions: []float32{-50, 3, -50, 50, 3, -50, 0, 3, 50}}}}
	ground.Nodes[0].Mesh.ComputeNormals()
	made.Show(ground, camera.Rig{Position: graphics.Vector{0, 0, 0, 1}})
	made.Steer(input.NewState(), 0, headAt(0, 0), true)
	if !nearlyEqual(made.Rig().Position[1], 3) {
		test.Fatalf("the floor is at %v, want on the new scene's ground at 3", made.Rig().Position[1])
	}
}

func TestMovingKeepsTheSceneAndTheHands(test *testing.T) {
	world := scene.New()
	made := New(world, Options{})
	made.Move(camera.Rig{Position: graphics.Vector{12, 0, -30, 1}, YawRadians: 1.2})
	if made.World() != world || len(world.Nodes) != 2 {
		test.Fatalf("moving changed the scene: %d nodes, want the two hands' cubes once", len(world.Nodes))
	}
	if made.Rig().Position != (graphics.Vector{12, 0, -30, 1}) || made.Rig().YawRadians != 1.2 || made.Rig().Scale != PersonScale {
		test.Fatalf("the rig is %+v, want the place asked as a person", made.Rig())
	}
	made.Move(camera.Rig{Position: graphics.Vector{0, 0, 0, 1}, Scale: 55})
	if made.Rig().Scale != 55 {
		test.Fatalf("the scale is %v, want the 55 asked", made.Rig().Scale)
	}
}
