// Package pointer points at a headset's panels with a controller, as a laser: a ray from where the
// controller aims meets a panel at one of its pixels, and that pixel becomes the cursor of the
// interface drawn on the panel, the trigger its mouse button and the thumbstick its scroll wheel.
// A beam shows where the ray goes. Nothing here makes anything new in a frame.
package pointer

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// How far the trigger goes in before it presses, and how far it comes back out before it lets go:
// apart, so a finger resting halfway does not click again and again.
const (
	triggerPress   = 0.7
	triggerRelease = 0.3
)

// ScrollNotchesPerSecond is how fast a stick pushed all the way scrolls, in a mouse wheel's
// notches a second.
const ScrollNotchesPerSecond = 12

// stickRest is how far a stick moves before it scrolls: one let go rarely reads exactly 0.
const stickRest = 0.15

// Ray is a line from a point, the way a controller aims, in the headset's space (metres).
type Ray struct {
	Origin, Direction graphics.Vector // Direction is a unit vector
}

// FromAim is the ray of a controller's aim pose, which points along its -Z.
func FromAim(aim graphics.Pose) Ray {
	origin := aim.Position
	origin[3] = 1
	return Ray{Origin: origin, Direction: aim.Orientation.Rotate(graphics.Vector{0, 0, -1, 0})}
}

// At is the point a distance along the ray.
func (ray Ray) At(distance float32) graphics.Vector {
	return graphics.Vector{ray.Origin[0] + ray.Direction[0]*distance, ray.Origin[1] + ray.Direction[1]*distance,
		ray.Origin[2] + ray.Direction[2]*distance, 1}
}

// Hit is where a ray meets a panel: the pixel from the panel's top-left corner, and how far along
// the ray, in metres.
type Hit struct {
	X, Y     float32
	Distance float32
}

// HitPanel finds where a ray meets a panel, as openxr.Panel places it: its middle at Pose, facing
// +Z, WidthMetres × HeightMetres, a picture of Width × Height pixels. A ray from behind the panel,
// parallel to it, or passing beside it misses.
func HitPanel(ray Ray, panel *openxr.Panel) (Hit, bool) {
	if panel.WidthMetres <= 0 || panel.HeightMetres <= 0 {
		return Hit{}, false
	}
	toPanel := panel.Pose.Orientation.Conjugate()
	offset := graphics.Vector{ray.Origin[0] - panel.Pose.Position[0], ray.Origin[1] - panel.Pose.Position[1],
		ray.Origin[2] - panel.Pose.Position[2], 0}
	origin, direction := toPanel.Rotate(offset), toPanel.Rotate(ray.Direction)
	// in the panel's own space the panel is the plane z = 0, seen from +z
	if origin[2] <= 0 || direction[2] >= -1e-6 {
		return Hit{}, false
	}
	distance := -origin[2] / direction[2]
	x, y := origin[0]+direction[0]*distance, origin[1]+direction[1]*distance
	across, up := x/panel.WidthMetres+0.5, 0.5-y/panel.HeightMetres
	if across < 0 || across > 1 || up < 0 || up > 1 {
		return Hit{}, false
	}
	return Hit{X: across * float32(panel.Width()), Y: up * float32(panel.Height()), Distance: distance}, true
}

// Pointer is one controller pointing at a panel, and what it does to the panel's interface.
type Pointer struct {
	pressed  bool // the trigger holds the button down
	pointing bool // the ray was on the panel last frame
	hit      Hit
}

// Update aims the hand at the panel and feeds the interface's input state (one of its own, not the
// controllers' pad): the cursor goes to the pixel the ray meets and the trigger presses the left
// button there; the stick, up and down, scrolls. A press begun on the panel is let go when the ray
// leaves it, so no button is ever left held; while the ray is off the panel the cursor leaves the
// interface, and the stick and the trigger are the program's again. The state's window is the
// panel's picture. It says whether the ray is on the panel, and where.
func (pointer *Pointer) Update(state *input.State, hand openxr.Hand, panel *openxr.Panel, seconds float64) (Hit, bool) {
	state.WindowWidth, state.WindowHeight = panel.Width(), panel.Height()
	state.FramebufferWidth, state.FramebufferHeight = panel.Width(), panel.Height()
	hit, onPanel := Hit{}, false
	if hand.Tracked && panel.Shown {
		hit, onPanel = HitPanel(FromAim(hand.Aim), panel)
	}
	if !onPanel {
		pointer.leave(state)
		return Hit{}, false
	}
	feed := state.Feed()
	feed.MoveCursor(float64(hit.X), float64(hit.Y))
	pointer.pointing, pointer.hit = true, hit
	pointer.followTrigger(state, hand.Trigger)
	if math.Abs(float64(hand.StickY)) > stickRest {
		feed.AddScroll(float64(hand.StickY) * ScrollNotchesPerSecond * seconds)
	}
	return hit, true
}

// Pressed tells whether the trigger holds the interface's button down.
func (pointer *Pointer) Pressed() bool { return pointer.pressed }

// followTrigger presses and lets go the left button with the trigger, apart enough not to flicker.
func (pointer *Pointer) followTrigger(state *input.State, trigger float32) {
	if !pointer.pressed && trigger >= triggerPress {
		pointer.pressed = true
		state.Feed().SetButton(input.MouseLeft, true)
	}
	if pointer.pressed && trigger <= triggerRelease {
		pointer.pressed = false
		state.Feed().SetButton(input.MouseLeft, false)
	}
}

// leave lets go of a held button where the ray last was, and takes the cursor off the interface.
func (pointer *Pointer) leave(state *input.State) {
	if !pointer.pointing {
		return
	}
	pointer.pointing = false
	feed := state.Feed()
	if pointer.pressed {
		pointer.pressed = false
		feed.SetButton(input.MouseLeft, false)
	}
	feed.PointerLeaves()
}
