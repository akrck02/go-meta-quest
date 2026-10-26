package place

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
)

func near(first, second float64) bool { return math.Abs(first-second) < 1e-3 }

// headAt is a head at a point of the room looking yaw radians to the left of -Z.
func headAt(x, y, z float32, yaw float64) graphics.Pose {
	return graphics.Pose{Orientation: graphics.QuaternionAroundY(yaw), Position: graphics.Vector{x, y, z, 1}}
}

func samePoint(first, second graphics.Vector) bool {
	return near(float64(first[0]), float64(second[0])) && near(float64(first[1]), float64(second[1])) && near(float64(first[2]), float64(second[2]))
}

func TestTheHeadsYawIsWhereItLooksAlongTheFloor(test *testing.T) {
	if yaw := HeadYaw(headAt(0, 1.6, 0, 0.5)); !near(float64(yaw), 0.5) {
		test.Fatalf("a head turned 0.5 rad looks along %v", yaw)
	}
	straightDown := graphics.Pose{Orientation: graphics.Quaternion{X: float32(-math.Sqrt2 / 2), W: float32(math.Sqrt2 / 2)}}
	if yaw := HeadYaw(straightDown); yaw != 0 {
		test.Fatalf("a head looking straight down has a yaw of %v, want 0", yaw)
	}
	if yaw := YawFacing(-1, 0); !near(float64(yaw), math.Pi/2) {
		test.Fatalf("facing -X is %v, want π/2 (counter-clockwise from -Z)", yaw)
	}
}

func TestTheAngleBetweenTwoYawsIsTheShortTurn(test *testing.T) {
	for _, each := range []struct{ yaw, from, want float64 }{{0.2, 0.1, 0.1}, {-3, 3, 2*math.Pi - 6}, {3, -3, 6 - 2*math.Pi}} {
		if turn := AngleBetween(each.yaw, each.from); !near(turn, each.want) {
			test.Errorf("from %v to %v turns %v, want %v", each.from, each.yaw, turn, each.want)
		}
	}
}

func TestAnglesTellHowAPoseIsTurned(test *testing.T) {
	angles := AnglesOf(headAt(0, 0, 0, math.Pi/2))
	if !near(angles.Yaw, 90) || !near(angles.Pitch, 0) || !near(angles.Roll, 0) {
		test.Fatalf("a head turned left a quarter turn reads %+v", angles)
	}
}

func TestARigWithTheHeadAtAPointPutsTheEyesOverIt(test *testing.T) {
	head := headAt(0.4, 1.6, -0.3, 0.2)
	for _, scale := range []float32{1, 20} {
		rig := RigWithHeadAt(graphics.Vector{10, 2, -5, 1}, 1, head, scale)
		eye := rig.EyePosition(head)
		if !near(float64(eye[0]), 10) || !near(float64(eye[2]), -5) || rig.Position[1] != 2 {
			test.Fatalf("at scale %v the eyes are over %v, the floor at %v", scale, eye, rig.Position[1])
		}
		if yaw := float64(rig.YawRadians + HeadYaw(head)); !near(yaw, 1) {
			test.Fatalf("at scale %v the head looks along %v in the scene, want 1", scale, yaw)
		}
	}
}

func TestTurningAroundTheHeadKeepsItOverTheSamePointAndTheFloorsHeight(test *testing.T) {
	head := headAt(0.3, 1.6, 0.2, 0)
	rig := camera.Rig{Position: graphics.Vector{5, 0.8, 7, 1}, YawRadians: 0.3, Scale: 2}
	before := rig.EyePosition(head)
	turned := TurnedAround(rig, head, 0.5)
	after := turned.EyePosition(head)
	if !near(float64(before[0]), float64(after[0])) || !near(float64(before[2]), float64(after[2])) {
		test.Fatalf("the head moved from %v to %v", before, after)
	}
	if !near(float64(turned.YawRadians), 0.8) || turned.Position[1] != 0.8 || turned.Scale != 2 {
		test.Fatalf("turned is %+v, want a yaw of 0.8, the floor at 0.8 and the scale kept", turned)
	}
}

func TestRescalingAroundTheHeadKeepsItOverTheSamePoint(test *testing.T) {
	head := headAt(0.3, 1.6, 0.2, 0.4)
	rig := camera.Rig{Position: graphics.Vector{5, 0, 7, 1}, YawRadians: 0.3, Scale: 1}
	before := rig.EyePosition(head)
	giant := RescaledAround(rig, head, 20)
	after := giant.EyePosition(head)
	if giant.Scale != 20 || !near(float64(before[0]), float64(after[0])) || !near(float64(before[2]), float64(after[2])) {
		test.Fatalf("the giant's head is over %v, want over %v", after, before)
	}
}

func TestAPointOfTheSceneIsFoundInTheRoomAndKeptThere(test *testing.T) {
	rig := camera.Rig{Position: graphics.Vector{5, 1, 7, 1}, YawRadians: 0.7, Scale: 30}
	scenePoint := graphics.Vector{40, 3, -20, 1}
	roomPoint := RoomPoint(rig, scenePoint)
	if back := rig.Matrix().Apply(roomPoint); !samePoint(back, scenePoint) {
		test.Fatalf("the room's point %v is %v in the scene, want %v", roomPoint, back, scenePoint)
	}
	moved := Keeping(rig, scenePoint, graphics.Vector{0, 0.75, -1, 1})
	if inRoom := RoomPoint(moved, scenePoint); !samePoint(inRoom, graphics.Vector{0, 0.75, -1, 1}) {
		test.Fatalf("the kept point is at %v in the room", inRoom)
	}
}

func TestAModelTurnsAndResizesAboutItsMiddle(test *testing.T) {
	rig := camera.Rig{Position: graphics.Vector{5, 0, 7, 1}, YawRadians: 0.2, Scale: 40}
	middle := graphics.Vector{100, 0, 60, 1}
	before := RoomPoint(rig, middle)
	turned := TurnedAbout(rig, middle, math.Pi/4)
	if !samePoint(RoomPoint(turned, middle), before) || !near(float64(turned.YawRadians), 0.2-math.Pi/4) {
		test.Fatalf("turned, the middle moved to %v from %v (yaw %v)", RoomPoint(turned, middle), before, turned.YawRadians)
	}
	resized := ResizedAbout(rig, middle, 25)
	if !samePoint(RoomPoint(resized, middle), before) || resized.Scale != 25 {
		test.Fatalf("resized, the middle moved to %v from %v", RoomPoint(resized, middle), before)
	}
}

func TestAModelAheadHasItsMiddleOnTheTableBeforeTheHead(test *testing.T) {
	head := headAt(1, 1.6, 2, math.Pi/2) // looking along -X
	rig := ModelAhead(graphics.Vector{100, 0, 60, 1}, 50, head, 0.9, 0.75)
	if table := RoomPoint(rig, graphics.Vector{100, 0, 60, 1}); !samePoint(table, graphics.Vector{0.1, 0.75, 2, 1}) {
		test.Fatalf("the model's middle is at %v in the room, want 0.9 m along -X at 0.75 m", table)
	}
	north := RoomPoint(rig, graphics.Vector{100, 0, 10, 1}) // 50 units north of the middle: a metre away from the person
	if !samePoint(north, graphics.Vector{-0.9, 0.75, 2, 1}) {
		test.Fatalf("the scene's north is at %v in the room, want a metre further along where the head looks", north)
	}
}

func TestStandingBeforeAPointFacesItFromWhereThePersonIs(test *testing.T) {
	head := headAt(0, 1.6, 0, 0)
	rig := camera.Rig{Position: graphics.Vector{0, 0, 30, 1}, Scale: 1} // the person 30 m south of the origin
	stood := StandingBefore(rig, head, graphics.Vector{0, 0, 0, 1}, 6, 1)
	eye := stood.EyePosition(head)
	if !near(float64(eye[0]), 0) || !near(float64(eye[2]), 6) {
		test.Fatalf("the person stands at %v, want 6 m south of the point", eye)
	}
	if yaw := stood.YawRadians + HeadYaw(head); !near(float64(yaw), 0) {
		test.Fatalf("the person faces %v, want north (0)", yaw)
	}
	if distance := FloorDistance(stood, head, graphics.Vector{0, 0, 0, 1}); !near(distance, 6) {
		test.Fatalf("the point is %v m away along the floor, want 6", distance)
	}
}

func TestAPanelAheadFloatsBeforeTheEyesFacingThem(test *testing.T) {
	pose := Ahead(headAt(0, 1.6, 0, math.Pi/2), 1, 0.2)
	if !samePoint(pose.Position, graphics.Vector{-1, 1.4, 0, 1}) {
		test.Fatalf("the panel is at %v, want a metre along -X and 0.2 m under the eyes", pose.Position)
	}
	if facing := pose.Orientation.Rotate(graphics.Vector{0, 0, 1, 0}); !samePoint(facing, graphics.Vector{1, 0, 0, 0}) {
		test.Fatalf("the panel faces %v, want back toward the head", facing)
	}
}

func TestALabelFacesTheHeadUpright(test *testing.T) {
	pose := FacingHead(graphics.Vector{3, 2, 0, 1}, headAt(0, 1.6, 0, 0))
	if facing := pose.Orientation.Rotate(graphics.Vector{0, 0, 1, 0}); !samePoint(facing, graphics.Vector{-1, 0, 0, 0}) {
		test.Fatalf("the label faces %v, want toward the head (-X)", facing)
	}
	if up := pose.Orientation.Rotate(graphics.Vector{0, 1, 0, 0}); !samePoint(up, graphics.Vector{0, 1, 0, 0}) {
		test.Fatalf("the label's up is %v", up)
	}
}

func TestAStripUnderTheEyesLeansBackHalfTheWayToThem(test *testing.T) {
	pose := UnderEyes(headAt(0, 1.6, 0, 0), 0, 0.75, 0.5)
	if !samePoint(pose.Position, graphics.Vector{0, 1.1, -0.75, 1}) {
		test.Fatalf("the strip is at %v", pose.Position)
	}
	facing := pose.Orientation.Rotate(graphics.Vector{0, 0, 1, 0})
	upward := math.Atan2(float64(facing[1]), float64(facing[2]))
	if want := math.Atan2(0.5, 0.75) / 2; !near(upward, want) {
		test.Fatalf("the strip's face is turned %v up, want half the eyes' %v", upward, want)
	}
}

func TestSomethingFollowingTheHeadWaitsTillItLooksWellAway(test *testing.T) {
	if yaw := Following(0, headAt(0, 1.6, 0, 0.3), 30); yaw != 0 {
		test.Fatalf("a glance of 17° moved it to %v", yaw)
	}
	if yaw := Following(0, headAt(0, 1.6, 0, 0.8), 30); !near(float64(yaw), 0.8) {
		test.Fatalf("looking 46° away left it at %v", yaw)
	}
}

func TestAThingLooksAsTallAsItsAngle(test *testing.T) {
	if height := HeightForAngle(10, 45); !near(height, 10) {
		test.Fatalf("45° ten metres away is %v m tall", height)
	}
}

func TestTheMenuShowsOverAPalmTurnedToTheFace(test *testing.T) {
	head := headAt(0, 1.6, 0, 0)
	toward := graphics.Pose{Orientation: graphics.QuaternionAroundY(-math.Pi / 2), Position: graphics.Vector{0, 1.6, -0.4, 1}} // its +X toward +Z, the head
	if facing := PalmFacing(toward, head); !near(facing, 1) {
		test.Fatalf("a palm turned to the face faces it %v", facing)
	}
	away := graphics.Pose{Orientation: graphics.QuaternionAroundY(math.Pi / 2), Position: toward.Position}
	if facing := PalmFacing(away, head); !near(facing, -1) {
		test.Fatalf("a palm turned away faces the head %v", facing)
	}
	pose := OverPalm(toward, head, 0.14, 0.04)
	if !samePoint(pose.Position, graphics.Vector{0, 1.74, -0.36, 1}) {
		test.Fatalf("the menu is at %v, want 14 cm over the hand and 4 cm toward the head", pose.Position)
	}
}
