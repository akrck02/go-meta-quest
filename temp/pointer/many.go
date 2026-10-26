package pointer

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// Interface is a panel with an interface of its own drawn on it: the input its frames read and the
// pointer that feeds it. A program with several (a main panel, a card beside a thing, a menu over a
// hand) points at them together with PointAtNearest.
type Interface struct {
	Panel   *openxr.Panel
	Input   *input.State
	Pointer Pointer
}

// PointAtNearest points a hand at whichever shown interface its ray meets first, as one laser can
// only be on one; the others let go of the hand (their cursors leave, a held button is let go).
// Pointing at none of them, the first has the hand. It returns where the ray meets the interface it
// is on, and whether it is on one. Nothing is made.
func PointAtNearest(interfaces []*Interface, hand openxr.Hand, seconds float64) (Hit, bool) {
	chosen, nearest := 0, float32(math.MaxFloat32)
	ray := FromAim(hand.Aim)
	for index, each := range interfaces {
		if !hand.Tracked || each.Panel == nil || !each.Panel.Shown {
			continue
		}
		if hit, hits := HitPanel(ray, each.Panel); hits && hit.Distance < nearest {
			chosen, nearest = index, hit.Distance
		}
	}
	var hit Hit
	var onPanel bool
	for index, each := range interfaces {
		if each.Panel == nil {
			continue
		}
		if index == chosen {
			hit, onPanel = each.Pointer.Update(each.Input, hand, each.Panel, seconds)
		} else {
			each.Pointer.Update(each.Input, openxr.Hand{}, each.Panel, seconds)
		}
	}
	return hit, onPanel
}

// SceneRay is the ray a controller points along in the scene, through the rig: from the hand, down
// its aim, in the scene's units. It is what a program picks things in its scene with.
func SceneRay(rig camera.Rig, aim graphics.Pose) camera.Ray {
	place := rig.Place(aim)
	return camera.Ray{Origin: place.Apply(graphics.Vector{0, 0, 0, 1}), Direction: place.Apply(graphics.Vector{0, 0, -1, 0}).Normalize()}
}

// AimAt is a controller's aim pose at a point, pointing (its -Z) at a target, with no roll: how a
// program (or a test) points a hand of its own.
func AimAt(origin, target graphics.Vector) graphics.Pose {
	toX, toY, toZ := float64(target[0]-origin[0]), float64(target[1]-origin[1]), float64(target[2]-origin[2])
	yaw := math.Atan2(-toX, -toZ)
	pitch := math.Atan2(toY, math.Hypot(toX, toZ))
	tilt := graphics.Quaternion{X: float32(math.Sin(pitch / 2)), W: float32(math.Cos(pitch / 2))}
	origin[3] = 1
	return graphics.Pose{Orientation: graphics.QuaternionAroundY(yaw).Multiply(tilt), Position: origin}
}

// PanelPoint is a pixel of a panel (from its top-left corner) in the headset's space: the way back
// from HitPanel, to point at a button.
func PanelPoint(panel *openxr.Panel, x, y float32) graphics.Vector {
	across := (x/float32(panel.Width()) - 0.5) * panel.WidthMetres
	up := (0.5 - y/float32(panel.Height())) * panel.HeightMetres
	offset := panel.Pose.Orientation.Rotate(graphics.Vector{across, up, 0, 0})
	return graphics.Vector{panel.Pose.Position[0] + offset[0], panel.Pose.Position[1] + offset[1], panel.Pose.Position[2] + offset[2], 1}
}
