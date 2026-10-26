package pointer

import (
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// interfaceAt is an interface on a 1 × 1 m panel of 100 × 100 pixels, its middle at a point,
// facing +Z.
func interfaceAt(x, y, z float32) *Interface {
	panel := openxr.DetachedPanel(100, 100)
	panel.Pose = graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{x, y, z, 1}}
	panel.FitWidth(1)
	panel.Shown = true
	return &Interface{Panel: panel, Input: input.NewState()}
}

func TestTheNearestInterfaceHasTheHandAndTheOthersLetGo(test *testing.T) {
	far, close := interfaceAt(0, 1.5, -3), interfaceAt(0, 1.5, -1)
	interfaces := []*Interface{far, close}
	hand := openxr.Hand{Tracked: true, Aim: AimAt(graphics.Vector{0, 1.5, 0, 1}, graphics.Vector{0, 1.5, -5, 1})}
	hand.Trigger = 1
	hit, onPanel := PointAtNearest(interfaces, hand, 1.0/72)
	if !onPanel || !nearlyEqual(hit.Distance, 1) {
		test.Fatalf("the hand is on %+v (%v), want the near panel a metre away", hit, onPanel)
	}
	if !close.Input.Button(input.MouseLeft) || far.Input.Button(input.MouseLeft) {
		test.Fatal("want the trigger pressing the near interface's button only, not the one behind it")
	}
	hand.Trigger = 0
	close.Panel.Shown = false
	if hit, onPanel = PointAtNearest(interfaces, hand, 1.0/72); !onPanel || !nearlyEqual(hit.Distance, 3) {
		test.Fatalf("with the near one hidden the hand is on %+v (%v), want the far one", hit, onPanel)
	}
	allocations := testing.AllocsPerRun(100, func() { PointAtNearest(interfaces, hand, 1.0/72) })
	if allocations != 0 {
		test.Fatalf("pointing at several interfaces made %v allocations, want none", allocations)
	}
}

func TestAPixelOfAPanelIsWhereTheRayMeetsIt(test *testing.T) {
	panel := panelAhead()
	origin := graphics.Vector{0.3, 1.2, 0, 1}
	point := PanelPoint(panel, 250, 125)
	hit, found := HitPanel(FromAim(AimAt(origin, point)), panel)
	if !found || !nearlyEqual(hit.X, 250) || !nearlyEqual(hit.Y, 125) {
		test.Fatalf("aimed at pixel 250, 125 the ray meets %+v (%v)", hit, found)
	}
}

func TestTheRayInTheSceneGoesThroughTheRig(test *testing.T) {
	rig := camera.Rig{Position: graphics.Vector{10, 0, 20, 1}, Scale: 2}
	ray := SceneRay(rig, AimAt(graphics.Vector{0, 1, 0, 1}, graphics.Vector{0, 1, -1, 1}))
	if !nearlyEqual(ray.Origin[0], 10) || !nearlyEqual(ray.Origin[1], 2) || !nearlyEqual(ray.Origin[2], 20) {
		test.Fatalf("the ray starts at %v, want the hand through the rig (10, 2, 20)", ray.Origin)
	}
	if !nearlyEqual(ray.Direction[2], -1) {
		test.Fatalf("the ray goes along %v, want -Z", ray.Direction)
	}
}
