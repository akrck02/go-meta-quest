package pointer

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

func nearlyEqual(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-3 }

// panelAhead is a 2 × 1 m panel of 1000 × 500 pixels, its middle 2 m ahead of the origin at eye
// height, facing back toward the origin.
func panelAhead() *openxr.Panel {
	panel := openxr.DetachedPanel(1000, 500)
	panel.Pose = graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 1.5, -2, 1}}
	panel.FitWidth(2)
	panel.Shown = true
	return panel
}

// handAt is a tracked hand at eye height aiming at a point.
func handAt(x, y, z float32, at graphics.Vector) openxr.Hand {
	from := graphics.Vector{x, y, z, 1}
	direction := graphics.Vector{at[0] - x, at[1] - y, at[2] - z, 0}
	return openxr.Hand{Tracked: true, Aim: graphics.Pose{Orientation: aimingAlong(direction), Position: from}}
}

// aimingAlong turns -Z toward a direction (no roll).
func aimingAlong(direction graphics.Vector) graphics.Quaternion {
	yaw := math.Atan2(float64(-direction[0]), float64(-direction[2]))
	pitch := math.Atan2(float64(direction[1]), math.Hypot(float64(direction[0]), float64(direction[2])))
	return graphics.QuaternionAroundY(yaw).Multiply(aroundX(pitch))
}

func aroundX(radians float64) graphics.Quaternion {
	half := radians / 2
	return graphics.Quaternion{X: float32(math.Sin(half)), W: float32(math.Cos(half))}
}

func TestARayMeetsAPanelAtThePixelItPointsAt(test *testing.T) {
	panel := panelAhead()
	hit, found := HitPanel(FromAim(handAt(0, 1.5, 0, graphics.Vector{0, 1.5, -2, 1}).Aim), panel)
	if !found || !nearlyEqual(hit.X, 500) || !nearlyEqual(hit.Y, 250) || !nearlyEqual(hit.Distance, 2) {
		test.Fatalf("the middle is hit at %+v (%v), want 500, 250 two metres away", hit, found)
	}
	hit, found = HitPanel(FromAim(handAt(0, 1.5, 0, graphics.Vector{-0.5, 1.75, -2, 1}).Aim), panel)
	if !found || !nearlyEqual(hit.X, 250) || !nearlyEqual(hit.Y, 125) {
		test.Fatalf("up and to the left is hit at %+v (%v), want 250, 125", hit, found)
	}
}

func TestARayMissesBesideBehindAndAwayFromAPanel(test *testing.T) {
	panel := panelAhead()
	misses := map[string]openxr.Hand{
		"beside":         handAt(0, 1.5, 0, graphics.Vector{1.5, 1.5, -2, 1}),
		"away":           handAt(0, 1.5, 0, graphics.Vector{0, 1.5, 2, 1}),
		"from behind":    handAt(0, 1.5, -4, graphics.Vector{0, 1.5, -2, 1}),
		"parallel to it": handAt(0, 1.5, -2, graphics.Vector{1, 1.5, -2, 1}),
	}
	for name, hand := range misses {
		if hit, found := HitPanel(FromAim(hand.Aim), panel); found {
			test.Fatalf("a ray %s hit the panel at %+v", name, hit)
		}
	}
}

func TestATurnedPanelIsHitInItsOwnPixels(test *testing.T) {
	panel := panelAhead()
	// turned a quarter, at x -2 facing +x, toward the origin
	panel.Pose = graphics.Pose{Orientation: graphics.QuaternionAroundY(math.Pi / 2), Position: graphics.Vector{-2, 1.5, 0, 1}}
	hit, found := HitPanel(FromAim(handAt(0, 1.5, 0, graphics.Vector{-2, 1.5, 0.5, 1}).Aim), panel)
	if !found || !nearlyEqual(hit.X, 250) || !nearlyEqual(hit.Y, 250) {
		test.Fatalf("the turned panel is hit at %+v (%v), want 250, 250: its left half is toward +z", hit, found)
	}
}

func TestTheTriggerClicksWhereTheRayMeetsThePanel(test *testing.T) {
	panel, state, pointer := panelAhead(), input.NewState(), &Pointer{}
	hand := handAt(0, 1.5, 0, graphics.Vector{0, 1.5, -2, 1})
	if _, on := pointer.Update(state, hand, panel, 1.0/72); !on || state.CursorX != 500 || state.CursorY != 250 {
		test.Fatalf("the cursor is at %v, %v (on %v), want 500, 250", state.CursorX, state.CursorY, on)
	}
	if state.WindowWidth != 1000 || state.WindowHeight != 500 {
		test.Fatalf("the interface's window is %v × %v, want the panel's 1000 × 500", state.WindowWidth, state.WindowHeight)
	}
	hand.Trigger = 0.5
	pointer.Update(state, hand, panel, 1.0/72)
	if state.Button(input.MouseLeft) {
		test.Fatal("a trigger halfway pressed the button")
	}
	hand.Trigger = 0.9
	pointer.Update(state, hand, panel, 1.0/72)
	if !state.Button(input.MouseLeft) || !pointer.Pressed() {
		test.Fatal("a trigger pulled did not press the button")
	}
	hand.Trigger = 0.5
	pointer.Update(state, hand, panel, 1.0/72)
	if !state.Button(input.MouseLeft) {
		test.Fatal("a trigger eased halfway let go: it should hold until it is nearly out")
	}
	hand.Trigger = 0.1
	pointer.Update(state, hand, panel, 1.0/72)
	if state.Button(input.MouseLeft) {
		test.Fatal("a trigger let out still holds the button")
	}
}

func TestLeavingThePanelLetsGoOfTheButton(test *testing.T) {
	panel, state, pointer := panelAhead(), input.NewState(), &Pointer{}
	hand := handAt(0, 1.5, 0, graphics.Vector{0, 1.5, -2, 1})
	hand.Trigger = 1
	pointer.Update(state, hand, panel, 1.0/72)
	away := handAt(0, 1.5, 0, graphics.Vector{3, 1.5, -2, 1})
	away.Trigger = 1
	if _, on := pointer.Update(state, away, panel, 1.0/72); on || state.Button(input.MouseLeft) || pointer.Pressed() {
		test.Fatal("the button is still held after the ray left the panel")
	}
	hidden := hand
	panel.Shown = false
	if _, on := pointer.Update(state, hidden, panel, 1.0/72); on {
		test.Fatal("a hidden panel was pointed at")
	}
}

func TestTheStickScrollsOnlyWhilePointing(test *testing.T) {
	panel, state, pointer := panelAhead(), input.NewState(), &Pointer{}
	hand := handAt(0, 1.5, 0, graphics.Vector{0, 1.5, -2, 1})
	hand.StickY = 1
	pointer.Update(state, hand, panel, 0.5)
	if !nearlyEqual(float32(state.ScrollDelta), ScrollNotchesPerSecond/2) {
		test.Fatalf("half a second of the stick scrolled %v notches, want %v", state.ScrollDelta, ScrollNotchesPerSecond/2)
	}
	state.Feed().EndFrame()
	away := handAt(0, 1.5, 0, graphics.Vector{3, 1.5, -2, 1})
	away.StickY = 1
	pointer.Update(state, away, panel, 0.5)
	if state.ScrollDelta != 0 {
		test.Fatalf("the stick scrolled %v off the panel", state.ScrollDelta)
	}
}

func TestTheBeamReachesWhatTheHandPointsAt(test *testing.T) {
	beam := NewBeam(graphics.Vector{1, 1, 1, 1})
	if !beam.Node.Hidden {
		test.Fatal("a new beam shows before any hand holds it")
	}
	hand := handAt(0, 1.5, 0, graphics.Vector{0, 1.5, -2, 1})
	beam.Place(camera.Rig{Position: graphics.Vector{10, 0, 0, 1}, Scale: 1}, hand, 2)
	end := beam.Node.Transform.Apply(graphics.Vector{0, 0, -1, 1})
	if beam.Node.Hidden || !nearlyEqual(end[0], 10) || !nearlyEqual(end[1], 1.5) || !nearlyEqual(end[2], -2) {
		test.Fatalf("the beam ends at %v, want the panel's middle moved with the room to 10, 1.5, -2", end)
	}
	beam.Place(camera.Rig{Scale: 1}, openxr.Hand{}, 2)
	if !beam.Node.Hidden {
		test.Fatal("the beam shows on an untracked hand")
	}
}

func TestPointingMakesNothingNew(test *testing.T) {
	panel, state, pointer, beam := panelAhead(), input.NewState(), &Pointer{}, NewBeam(graphics.Vector{1, 1, 1, 1})
	hand := handAt(0, 1.5, 0, graphics.Vector{0, 1.5, -2, 1})
	allocations := testing.AllocsPerRun(100, func() {
		hit, _ := pointer.Update(state, hand, panel, 1.0/72)
		beam.Place(camera.Rig{Scale: 1}, hand, hit.Distance)
	})
	if allocations != 0 {
		test.Fatalf("a frame's pointing made %v allocations, want none", allocations)
	}
}
