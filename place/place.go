// Package place works out where things stand in a headset. In the scene: the room the person
// carries (a camera.Rig) turned about their head, resized, moved so that a point of the scene is
// where they want it in the room, or set down before something. In the room: a panel ahead of the
// eyes, a label facing them, a strip under them that follows the head only once it looks away, a
// menu over a hand. All of it is arithmetic on poses, tested anywhere.
package place

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// HeadYaw is which way a head looks, turned about the vertical: 0 ahead (-Z), counter-clockwise
// from above. Looking straight up or down there is no way along the floor, and it is 0.
func HeadYaw(head graphics.Pose) float32 {
	forward := head.Orientation.Rotate(graphics.Vector{0, 0, -1, 0})
	if math.Hypot(float64(forward[0]), float64(forward[2])) < 1e-4 {
		return 0
	}
	return YawFacing(float64(forward[0]), float64(forward[2]))
}

// YawFacing is the yaw that looks along a direction on the floor.
func YawFacing(x, z float64) float32 {
	return float32(math.Atan2(-x, -z))
}

// AngleBetween is the turn from one yaw to another, between -π and π.
func AngleBetween(yaw, from float64) float64 {
	turn := math.Mod(yaw-from, 2*math.Pi)
	switch {
	case turn > math.Pi:
		turn -= 2 * math.Pi
	case turn < -math.Pi:
		turn += 2 * math.Pi
	}
	return turn
}

// Angles is how a pose is turned, in degrees: roll about where it looks (positive with its right
// side raised), pitch up from the floor, yaw from ahead (-Z), counter-clockwise from above.
type Angles struct {
	Roll, Pitch, Yaw float64
}

// AnglesOf is how a pose (a head, an eye) is turned: for telling it in a log.
func AnglesOf(pose graphics.Pose) Angles {
	forward := pose.Orientation.Rotate(graphics.Vector{0, 0, -1, 0})
	up := pose.Orientation.Rotate(graphics.Vector{0, 1, 0, 0})
	right := pose.Orientation.Rotate(graphics.Vector{1, 0, 0, 0})
	degrees := 180 / math.Pi
	return Angles{
		Roll:  math.Atan2(float64(right[1]), float64(up[1])) * degrees,
		Pitch: math.Asin(math.Max(-1, math.Min(1, float64(forward[1])))) * degrees,
		Yaw:   float64(YawFacing(float64(forward[0]), float64(forward[2]))) * degrees,
	}
}

// tiltedBack is a quaternion that turns about X by radians.
func tiltedBack(radians float64) graphics.Quaternion {
	half := radians / 2
	return graphics.Quaternion{X: float32(math.Sin(half)), W: float32(math.Cos(half))}
}
