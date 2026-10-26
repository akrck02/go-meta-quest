package place

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
)

// The room in the scene. A rig's Position is where the room's floor, under its origin, is in the
// scene; its yaw turns the room and its scale makes the person bigger (a giant over a model) or
// smaller. These move it without the person moving their own head.

// RigWithHeadAt is the rig that puts the head over a point of the scene's floor, looking the way
// yaw says (0 is -Z, counter-clockwise from above), at a scale. The floor's height is the point's.
func RigWithHeadAt(floor graphics.Vector, yaw float32, head graphics.Pose, scale float32) camera.Rig {
	rigYaw := yaw - HeadYaw(head)
	turned := graphics.RotateY(float64(rigYaw)).Apply(graphics.Vector{head.Position[0] * scale, 0, head.Position[2] * scale, 0})
	return camera.Rig{Position: graphics.Vector{floor[0] - turned[0], floor[1], floor[2] - turned[2], 1}, YawRadians: rigYaw, Scale: scale}
}

// TurnedAround turns the person about the vertical through their head, which stays over the same
// point of the floor: positive radians to the left. The floor stays at its height.
func TurnedAround(rig camera.Rig, head graphics.Pose, radians float32) camera.Rig {
	eye := rig.EyePosition(head)
	return RigWithHeadAt(graphics.Vector{eye[0], rig.Position[1], eye[2], 1}, rig.YawRadians+HeadYaw(head)+radians, head, scaleOf(rig))
}

// RescaledAround changes the person's size keeping their head over the same point of the floor,
// which stays at its height.
func RescaledAround(rig camera.Rig, head graphics.Pose, scale float32) camera.Rig {
	eye := rig.EyePosition(head)
	return RigWithHeadAt(graphics.Vector{eye[0], rig.Position[1], eye[2], 1}, rig.YawRadians+HeadYaw(head), head, scale)
}

// RoomPoint is where a point of the scene is in the room, through the rig: where to reach for it,
// or point at it from.
func RoomPoint(rig camera.Rig, scenePoint graphics.Vector) graphics.Vector {
	scale := scaleOf(rig)
	back := graphics.RotateY(float64(-rig.YawRadians)).Apply(graphics.Vector{scenePoint[0] - rig.Position[0],
		scenePoint[1] - rig.Position[1], scenePoint[2] - rig.Position[2], 0})
	return graphics.Vector{back[0] / scale, back[1] / scale, back[2] / scale, 1}
}

// Keeping moves a rig, turned and sized as it is, so that a point of the scene is at a point of
// the room.
func Keeping(rig camera.Rig, scenePoint, roomPoint graphics.Vector) camera.Rig {
	rig.Position = graphics.Vector{0, 0, 0, 1}
	moved := rig.Matrix().Apply(graphics.Vector{roomPoint[0], roomPoint[1], roomPoint[2], 1})
	rig.Position = graphics.Vector{scenePoint[0] - moved[0], scenePoint[1] - moved[1], scenePoint[2] - moved[2], 1}
	return rig
}

// TurnedAbout turns the scene about the vertical through one of its points, which stays where it is
// in the room: positive radians counter-clockwise from above. A model on a table turns in place.
func TurnedAbout(rig camera.Rig, scenePoint graphics.Vector, radians float32) camera.Rig {
	inRoom := RoomPoint(rig, scenePoint)
	rig.YawRadians -= radians // the room turns the other way under the scene
	return Keeping(rig, scenePoint, inRoom)
}

// ResizedAbout gives the rig another scale about a point of the scene, which stays where it is in
// the room: a model on a table grows or shrinks about its middle.
func ResizedAbout(rig camera.Rig, scenePoint graphics.Vector, scale float32) camera.Rig {
	inRoom := RoomPoint(rig, scenePoint)
	rig.Scale = scale
	return Keeping(rig, scenePoint, inRoom)
}

// ModelAhead makes the scene a model at a scale with one of its points (a model's middle) ahead of
// the head along the floor, aheadMetres away and heightMetres over the room's floor: on a table.
// The scene's -Z is the way the head looks.
func ModelAhead(scenePoint graphics.Vector, scale float32, head graphics.Pose, aheadMetres, heightMetres float32) camera.Rig {
	yaw := HeadYaw(head)
	ahead := graphics.QuaternionAroundY(float64(yaw)).Rotate(graphics.Vector{0, 0, -1, 0})
	table := graphics.Vector{head.Position[0] + ahead[0]*aheadMetres, heightMetres, head.Position[2] + ahead[2]*aheadMetres, 1}
	return Keeping(camera.Rig{YawRadians: -yaw, Scale: scale}, scenePoint, table)
}

// StandingBefore puts the person on the floor awayMetres (in the scene's units) from a point of
// the scene, on the side they are now, facing it, at a scale. The floor stays at its height. Right
// over the point they face -Z.
func StandingBefore(rig camera.Rig, head graphics.Pose, scenePoint graphics.Vector, away float64, scale float32) camera.Rig {
	eye := rig.EyePosition(head)
	towardX, towardZ := float64(scenePoint[0]-eye[0]), float64(scenePoint[2]-eye[2])
	length := math.Hypot(towardX, towardZ)
	if length < 1e-3 {
		towardX, towardZ, length = 0, -1, 1
	}
	towardX, towardZ = towardX/length, towardZ/length
	stand := graphics.Vector{scenePoint[0] - float32(towardX*away), rig.Position[1], scenePoint[2] - float32(towardZ*away), 1}
	return RigWithHeadAt(stand, YawFacing(towardX, towardZ), head, scale)
}

// FloorDistance is how far a point of the scene is from the person's head along the floor, in the
// person's own metres.
func FloorDistance(rig camera.Rig, head graphics.Pose, scenePoint graphics.Vector) float64 {
	eye := rig.EyePosition(head)
	return math.Hypot(float64(scenePoint[0]-eye[0]), float64(scenePoint[2]-eye[2])) / float64(scaleOf(rig))
}

// scaleOf is a rig's scale, a person's for 0.
func scaleOf(rig camera.Rig) float32 {
	if rig.Scale <= 0 {
		return 1
	}
	return rig.Scale
}
