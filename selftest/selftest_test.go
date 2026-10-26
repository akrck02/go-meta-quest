package selftest

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/akrck02/go-meta-quest/place"
	"github.com/akrck02/go-meta-quest/pointer"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

var head = graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 1.6, 0, 1}}

// report is what a runner told, a line a step.
type report struct {
	names  []string
	errors []error
	totals [2]int
}

func newRunner(steps []Step, told *report) *Runner {
	runner := New(steps, 40*time.Millisecond)
	runner.Report = func(_, _ int, name string, err error) {
		told.names, told.errors = append(told.names, name), append(told.errors, err)
	}
	runner.Finished = func(passed, failed int) { told.totals = [2]int{passed, failed} }
	return runner
}

// run goes round frames until the runner is done, a second at most, calling frame between Drive
// and Check as a program's loop does.
func run(test *testing.T, runner *Runner, frame func(hands [2]openxr.Hand, pad *input.State, menu bool)) {
	deadline := time.Now().Add(time.Second)
	for !runner.Done() && time.Now().Before(deadline) {
		var hands [2]openxr.Hand
		menu := false
		runner.Drive(head, true, &hands, runner.Pad, &menu)
		if frame != nil {
			frame(hands, runner.Pad, menu)
		}
		runner.Pad.Feed().EndFrame()
		runner.Check()
		time.Sleep(2 * time.Millisecond)
	}
	if !runner.Done() {
		test.Fatalf("the runner did not end: at step %d of %d", runner.Current(), len(runner.Steps))
	}
}

func TestEveryStepIsCheckedAndTold(test *testing.T) {
	var told report
	steps := []Step{
		{Name: "passes", Check: func() error { return nil }},
		{Name: "fails", Check: func() error { return errors.New("wrong") }},
		{Name: "fails acting", Act: func() error { return errors.New("cannot") }, Check: func() error { return nil }},
	}
	run(test, newRunner(steps, &told), nil)
	if len(told.names) != 3 || told.errors[0] != nil || told.errors[1] == nil || told.errors[2] == nil {
		test.Fatalf("told %v with %v", told.names, told.errors)
	}
	if told.totals != [2]int{1, 2} {
		test.Fatalf("finished with %v, want 1 passed and 2 failed", told.totals)
	}
}

func TestAStepWaitsTillItIsReadyAndNoLonger(test *testing.T) {
	var told report
	ready := false
	steps := []Step{
		{Name: "waits", Ready: func() bool { return ready }, Act: func() error { return nil }},
		{Name: "never", Ready: func() bool { return false }},
	}
	runner := newRunner(steps, &told)
	runner.Patience = 100 * time.Millisecond
	frames := 0
	run(test, runner, func([2]openxr.Hand, *input.State, bool) {
		if frames++; frames == 5 {
			ready = true
		}
	})
	if told.errors[0] != nil || !errors.Is(told.errors[1], ErrNeverReady) {
		test.Fatalf("told %v, want the first passed and the second never ready", told.errors)
	}
}

func TestTheRightHandPointsAndPullsTheTriggerOnce(test *testing.T) {
	var told report
	target := graphics.Vector{0.5, 1.4, -2, 1}
	pulls, aimed := 0, true
	var runner *Runner
	steps := []Step{{Name: "points", Act: func() error { runner.PointAt(target, true); return nil }}}
	runner = newRunner(steps, &told)
	wasPulled := false
	run(test, runner, func(hands [2]openxr.Hand, _ *input.State, _ bool) {
		right := hands[rightHand]
		hit := pointer.FromAim(right.Aim)
		toTarget := graphics.Vector{target[0] - hit.Origin[0], target[1] - hit.Origin[1], target[2] - hit.Origin[2], 0}.Normalize()
		if !right.Tracked || math.Abs(float64(hit.Direction.Dot(toTarget))-1) > 1e-4 {
			aimed = false
		}
		if pulled := right.Trigger > 0; pulled && !wasPulled {
			pulls++
		}
		wasPulled = right.Trigger > 0
		if hands[leftHand].Tracked {
			aimed = false
		}
	})
	if !aimed || pulls != 1 {
		test.Fatalf("aimed at the target %v, pulled %d times, want aimed and once", aimed, pulls)
	}
}

func TestAButtonIsPressedForAFrameAndTheMenuToo(test *testing.T) {
	var told report
	var runner *Runner
	steps := []Step{{Name: "presses", Act: func() error { runner.Press(input.PadX); runner.PressMenu(); return nil }}}
	runner = newRunner(steps, &told)
	pressed, held, menus := 0, 0, 0
	run(test, runner, func(_ [2]openxr.Hand, pad *input.State, menu bool) {
		if pad.PadPressed(input.PadX) {
			pressed++
		}
		if pad.PadDown(input.PadX) {
			held++
		}
		if menu {
			menus++
		}
	})
	if pressed != 1 || held != 1 || menus != 1 {
		test.Fatalf("X pressed %d times and held %d frames, the menu %d times: want once each", pressed, held, menus)
	}
}

func TestThePalmHeldUpFacesTheHead(test *testing.T) {
	if facing := place.PalmFacing(PalmToFace(head).Grip, head); facing < 0.99 {
		test.Fatalf("the palm faces the head %v, want squarely", facing)
	}
	beside := HandBeside(head)
	if beside[0] <= 0 || beside[1] >= head.Position[1] || beside[2] >= 0 {
		test.Fatalf("the right hand is at %v, want low, right and a little ahead", beside)
	}
}

func TestNotAskedForOffTheHeadset(test *testing.T) {
	if AskedFor("selftest") {
		test.Fatal("asked for off the headset")
	}
}
