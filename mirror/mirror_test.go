package mirror

import (
	"math"
	"testing"
	"time"

	"github.com/akrck02/go-meta-quest/place"
	"github.com/akrck02/go-meta-quest/pointer"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/openxr"
)

func near(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-3 }

// questEye is a left eye as a Quest 3 gives it: a field wider on the outer side, the head turned.
func questEye() openxr.Eye {
	return openxr.Eye{
		Pose:  graphics.Pose{Orientation: graphics.QuaternionAroundY(0.6), Position: graphics.Vector{0.3, 1.6, -0.2, 1}},
		Field: graphics.FieldOfView{LeftRadians: -0.94, RightRadians: 0.79, UpRadians: 0.87, DownRadians: -0.91},
	}
}

// panelBefore is a 1200 × 750 panel 1.1 m wide a metre before an eye.
func panelBefore(eye openxr.Eye) *openxr.Panel {
	panel := openxr.DetachedPanel(1200, 750)
	panel.Pose = place.Ahead(eye.Pose, 1, 0.2)
	panel.FitWidth(1.1)
	panel.Shown = true
	return panel
}

func TestAPointOfTheEyesPictureIsSeenWhereItIs(test *testing.T) {
	eye := questEye()
	point := graphics.Vector{-1.5, 2.1, -3, 1}
	x, y, seen := PicturePoint(eye, point)
	if !seen || x < 0 || x > 1 || y < 0 || y > 1 {
		test.Fatalf("the point is at (%.3f, %.3f), seen %v: it should be in the picture", x, y, seen)
	}
	direction := Direction(eye, x, y)
	toPoint := graphics.Vector{point[0] - eye.Pose.Position[0], point[1] - eye.Pose.Position[1], point[2] - eye.Pose.Position[2], 0}.Normalize()
	if !near(direction[0], toPoint[0]) || !near(direction[1], toPoint[1]) || !near(direction[2], toPoint[2]) {
		test.Fatalf("the picture's point looks along %v, the point is along %v", direction, toPoint)
	}
	if _, _, seen := PicturePoint(eye, graphics.Vector{2, 1.6, 2, 1}); seen {
		test.Fatal("a point behind the eye is seen")
	}
}

func TestTheMiddleOfThePictureIsWhereTheFieldIsCentred(test *testing.T) {
	eye := openxr.Eye{Pose: graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 0, 0, 1}},
		Field: graphics.FieldOfView{LeftRadians: -0.8, RightRadians: 0.8, UpRadians: 0.8, DownRadians: -0.8}}
	if direction := Direction(eye, 0.5, 0.5); !near(direction[0], 0) || !near(direction[1], 0) || !near(direction[2], -1) {
		test.Fatalf("the middle looks along %v, not straight ahead", direction)
	}
	if up := Direction(eye, 0.5, 0); up[1] <= 0 {
		test.Fatalf("the picture's top looks along %v, not up", up)
	}
}

func TestATouchOnAPanelPointsAtTheSamePixel(test *testing.T) {
	eye := questEye()
	panel := panelBefore(eye)
	x, y, _ := PicturePoint(eye, pointer.PanelPoint(panel, 300, 200))
	hit, onPanel := pointer.HitPanel(pointer.FromAim(Hand(eye, Touch{X: x, Y: y, Down: true}).Aim), panel)
	if !onPanel || math.Abs(float64(hit.X-300)) > 1 || math.Abs(float64(hit.Y-200)) > 1 {
		test.Fatalf("the touch reached pixel (%.1f, %.1f), on the panel %v: it should be (300, 200)", hit.X, hit.Y, onPanel)
	}
}

func TestAFingerDownPullsTheTrigger(test *testing.T) {
	if Hand(questEye(), Touch{Down: true}).Trigger != 1 || Hand(questEye(), Touch{}).Trigger != 0 {
		test.Fatal("the trigger should be pulled while the finger is down, and only then")
	}
}

func TestTouchLinesAreRead(test *testing.T) {
	cases := []struct {
		line    string
		wasDown bool
		want    Touch
	}{
		{"down 0.25 0.5", false, Touch{X: 0.25, Y: 0.5, Down: true}},
		{"move 0.3 0.6", true, Touch{X: 0.3, Y: 0.6, Down: true}},
		{"move 0.3 0.6", false, Touch{X: 0.3, Y: 0.6}},
		{"up 0.3 0.6", true, Touch{X: 0.3, Y: 0.6}},
	}
	for _, each := range cases {
		got, err := ParseTouch(each.line, each.wasDown)
		if err != nil || got != each.want {
			test.Errorf("%q read as %+v (%v), want %+v", each.line, got, err, each.want)
		}
	}
	for _, wrong := range []string{"", "down", "tap 0.1 0.2", "down x 0.2"} {
		if _, err := ParseTouch(wrong, false); err == nil {
			test.Errorf("%q should not be read", wrong)
		}
	}
}

func TestAQuickTapStillPressesAndLetsGo(test *testing.T) {
	var gathered Touches
	for _, line := range []string{"down 0.5 0.5", "up 0.5 0.5"} {
		if err := gathered.Add(line); err != nil {
			test.Fatal(err)
		}
	}
	now := time.Now()
	if first, touching := gathered.Next(now); !touching || !first.Down {
		test.Fatalf("the first frame should press: %+v, %v", first, touching)
	}
	if second, touching := gathered.Next(now); !touching || second.Down {
		test.Fatalf("the second frame should let go: %+v, %v", second, touching)
	}
	if _, touching := gathered.Next(now.Add(2 * Lingers)); touching {
		test.Fatal("a finger gone a while should leave the controller its place")
	}
}

func TestMovesThatCameTogetherAreOne(test *testing.T) {
	var gathered Touches
	for _, line := range []string{"down 0.1 0.1", "move 0.2 0.2", "move 0.3 0.3", "up 0.4 0.4"} {
		if err := gathered.Add(line); err != nil {
			test.Fatal(err)
		}
	}
	now := time.Now()
	for frame, want := range []Touch{{X: 0.1, Y: 0.1, Down: true}, {X: 0.3, Y: 0.3, Down: true}, {X: 0.4, Y: 0.4}} {
		if got, _ := gathered.Next(now); got != want {
			test.Fatalf("frame %d: %+v, want %+v", frame, got, want)
		}
	}
}

func TestNoTouchLeavesTheController(test *testing.T) {
	var gathered Touches
	if _, touching := gathered.Next(time.Now()); touching {
		test.Fatal("with no touch yet, the controller should keep its place")
	}
}

func TestAMarkIsShownAndHidden(test *testing.T) {
	var gathered Touches
	if err := gathered.Add("mark 0.3 0.4"); err != nil {
		test.Fatal(err)
	}
	if mark, shown := gathered.Marked(); !shown || mark.X != 0.3 || mark.Y != 0.4 {
		test.Fatalf("the mark is %+v, shown %v", mark, shown)
	}
	if _, touching := gathered.Next(time.Now()); touching {
		test.Fatal("a mark is not a touch")
	}
	if err := gathered.Add("mark off"); err != nil {
		test.Fatal(err)
	}
	if _, shown := gathered.Marked(); shown {
		test.Fatal("the mark should be hidden")
	}
}

func TestTheMarkFloatsWhereThePictureSaysAheadOfTheEye(test *testing.T) {
	eye := questEye()
	x, y, seen := PicturePoint(eye, MarkPose(eye, Touch{X: 0.2, Y: 0.7}, 1.5).Position)
	if !seen || !near(x, 0.2) || !near(y, 0.7) {
		test.Fatalf("the mark is seen at (%.3f, %.3f), seen %v", x, y, seen)
	}
}

func TestTwoFingersDraggedRightOffThePanelTurnThePersonLeft(test *testing.T) {
	eye := questEye()
	panel := panelBefore(eye)
	panel.Shown = false
	var gathered Touches
	if err := gathered.Add("look 0.4 0.5 0.6 0.5"); err != nil {
		test.Fatal(err)
	}
	drags := gathered.TakeDrags()
	if len(drags) != 1 || len(gathered.TakeDrags()) != 0 {
		test.Fatalf("one drag is handed over once, got %d", len(drags))
	}
	from, to := Direction(eye, 0.4, 0.5), Direction(eye, 0.6, 0.5)
	wanted := place.YawFacing(float64(from[0]), float64(from[2])) - place.YawFacing(float64(to[0]), float64(to[2]))
	if turn := Carry(drags[0], eye, panel); wanted <= 0 || !near(turn, wanted) {
		test.Fatalf("the drag turns %.3f rad, want %.3f to the left", turn, wanted)
	}
	if err := gathered.Add("look 0.4 0.5"); err == nil {
		test.Fatal("half a drag should not be read")
	}
}

func TestTwoFingersOnAPanelCarryIt(test *testing.T) {
	eye := questEye()
	panel := panelBefore(eye)
	x, y, _ := PicturePoint(eye, pointer.PanelPoint(panel, 600, 375))
	if turn := Carry(Drag{From: Touch{X: x, Y: y}, To: Touch{X: x - 0.2, Y: y - 0.1}}, eye, panel); turn != 0 {
		test.Fatalf("dragging the panel turns the person %v", turn)
	}
	movedX, movedY, seen := PicturePoint(eye, pointer.PanelPoint(panel, 600, 375))
	if !seen || !near(movedX, x-0.2) || !near(movedY, y-0.1) {
		test.Fatalf("the panel's middle is at (%.3f, %.3f), want (%.3f, %.3f)", movedX, movedY, x-0.2, y-0.1)
	}
	if facing := panel.Pose.Orientation.Rotate(graphics.Vector{0, 0, 1, 0}); facing[0]*(eye.Pose.Position[0]-panel.Pose.Position[0])+
		facing[2]*(eye.Pose.Position[2]-panel.Pose.Position[2]) <= 0 {
		test.Fatal("the panel should still face the eye")
	}
	if right := panel.Pose.Orientation.Rotate(graphics.Vector{1, 0, 0, 0}); !near(right[1], 0) {
		test.Fatalf("the panel should stay upright, its right side is at %v", right)
	}
}
