package place

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// Things in the room, placed for the head that sees them. Panels face +Z, so a pose facing the head
// has its -Z pointing away from it.

// Ahead is where a panel floats when it is called up: aheadMetres ahead of the head the way it
// looks along the floor, belowMetres under the eyes, upright and facing the person.
func Ahead(head graphics.Pose, aheadMetres, belowMetres float32) graphics.Pose {
	turn := graphics.QuaternionAroundY(float64(HeadYaw(head)))
	ahead := turn.Rotate(graphics.Vector{0, 0, -1, 0})
	return graphics.Pose{Orientation: turn, Position: graphics.Vector{head.Position[0] + ahead[0]*aheadMetres,
		head.Position[1] - belowMetres, head.Position[2] + ahead[2]*aheadMetres, 1}}
}

// FacingHead is a label at a point of the room, upright and turned toward the head.
func FacingHead(point graphics.Vector, head graphics.Pose) graphics.Pose {
	yaw := YawFacing(float64(point[0]-head.Position[0]), float64(point[2]-head.Position[2]))
	return graphics.Pose{Orientation: graphics.QuaternionAroundY(float64(yaw)), Position: point}
}

// LeaningBack is a panel at a point under the eyes turned toward the head and leaned back half the
// way to face them: a strip low ahead, or a bar lying on a table, read without bending the neck.
func LeaningBack(point graphics.Vector, head graphics.Pose) graphics.Pose {
	pose := FacingHead(point, head)
	along := math.Max(math.Hypot(float64(point[0]-head.Position[0]), float64(point[2]-head.Position[2])), 0.05)
	pose.Orientation = pose.Orientation.Multiply(tiltedBack(-math.Atan2(float64(head.Position[1]-point[1]), along) / 2))
	return pose
}

// UnderEyes is where a strip floats for a head when it faces yaw: aheadMetres ahead along the
// floor, belowMetres under the eyes, leaned back toward them.
func UnderEyes(head graphics.Pose, yaw float32, aheadMetres, belowMetres float32) graphics.Pose {
	ahead := graphics.QuaternionAroundY(float64(yaw)).Rotate(graphics.Vector{0, 0, -1, 0})
	point := graphics.Vector{head.Position[0] + ahead[0]*aheadMetres, head.Position[1] - belowMetres,
		head.Position[2] + ahead[2]*aheadMetres, 1}
	return LeaningBack(point, head)
}

// Following is which way something that follows the head lazily faces now: where it faced, until
// the head looks more than degrees away from it, then where the head looks. A glance aside finds it
// where it was.
func Following(faced float32, head graphics.Pose, degrees float64) float32 {
	looking := HeadYaw(head)
	if math.Abs(AngleBetween(float64(looking), float64(faced))) > degrees*math.Pi/180 {
		return looking
	}
	return faced
}

// HeightForAngle is how tall a thing distance metres away is when it looks degrees tall.
func HeightForAngle(distance, degrees float64) float64 {
	return distance * math.Tan(degrees*math.Pi/180)
}

// PalmFacing is how squarely a hand's palm faces the head, from -1 (away) to 1: holding a Quest's
// left controller the palm is on its right side, the grip's +X.
func PalmFacing(grip, head graphics.Pose) float64 {
	palm := grip.Orientation.Rotate(graphics.Vector{1, 0, 0, 0})
	toward := graphics.Vector{head.Position[0] - grip.Position[0], head.Position[1] - grip.Position[1], head.Position[2] - grip.Position[2], 0}
	length := math.Sqrt(float64(toward.Dot(toward)))
	if length < 1e-3 {
		return 0
	}
	return float64(palm.Dot(toward)) / length
}

// OverPalm is where a menu floats for a hand: aboveMetres over the grip and towardMetres out of the
// palm (the grip's +X), upright and facing the head.
func OverPalm(grip, head graphics.Pose, aboveMetres, towardMetres float32) graphics.Pose {
	palm := grip.Orientation.Rotate(graphics.Vector{1, 0, 0, 0})
	point := graphics.Vector{grip.Position[0] + palm[0]*towardMetres, grip.Position[1] + aboveMetres + palm[1]*towardMetres,
		grip.Position[2] + palm[2]*towardMetres, 1}
	return FacingHead(point, head)
}
