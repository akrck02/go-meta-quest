// Package selftest goes through a program in the headset as a person would, with controllers of
// its own: a right hand that points at things and pulls the trigger, a left hand that turns its
// palm to the face, buttons pressed for a frame. Each step acts, lets the program settle, and checks
// that it did what it should. The same steps run in go test, with frames a test makes, and in the
// headset itself, where each step is told in the log and stays a moment on the screen.
//
// A program runs a Runner in its loop: Drive before its own frame (the runner puts its hands and
// presses into the frame's input), Check after it. In the headset it reads the runner's Pad
// instead of the real controllers, so a person wearing it cannot press over a step.
package selftest

import (
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akrck02/go-meta-quest/place"
	"github.com/akrck02/go-meta-quest/pointer"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// DefaultPatience is the longest a step waits to be ready (a scene read, a button shown) before it
// fails.
const DefaultPatience = 20 * time.Second

// PullTime is how long the trigger is held in: a few frames, as a quick pull. It goes in a quarter
// of the way into the step's settling.
const PullTime = 120 * time.Millisecond

// The hands, in the order a frame holds them (as go-meta-quest's app.Frame does).
const (
	leftHand  = 0
	rightHand = 1
)

// ErrNeverReady is what a step that waited longer than the runner's patience failed with.
var ErrNeverReady = errors.New("selftest: the step never got ready")

// Step is one thing the self-test does and what it checks after.
type Step struct {
	Name string
	// Ready says the step can start; nil when it always can.
	Ready func() bool
	// Act does the step: it points the right hand (Runner.PointAt), presses buttons, or does
	// something else to the program. An error fails the step.
	Act func() error
	// Check says what went wrong, once the step has settled; nil when it did what it should.
	Check func() error
}

// Runner runs the steps, frame by frame.
type Runner struct {
	Steps []Step
	// Settle is how long each step settles before it is checked: in the headset long enough to be
	// seen on the mirror, in a test only as long as the program needs.
	Settle time.Duration
	// Patience is how long a step waits to be ready (0 = DefaultPatience).
	Patience time.Duration
	// Pad is the controllers' buttons and sticks as the self-test presses them.
	Pad *input.State
	// BeforeEach is called as each step starts, before it acts: to note how things were.
	BeforeEach func()
	// Report tells how a step went (err nil when it passed), and Finished how they all went; by
	// default both write to the log, a line each.
	Report   func(index, total int, name string, err error)
	Finished func(passed, failed int)

	current  int
	started  time.Time // when the current step started acting (zero while it waits to be ready)
	waiting  time.Time // when the current step began waiting to be ready
	failure  error     // what the act said went wrong
	aiming   bool
	target   graphics.Vector // where the right hand points, in the room
	pull     bool
	toPress  []input.PadButton // pressed on the next frame
	held     []input.PadButton // pressed on the last frame, let go on this one
	menu     bool
	palmUp   bool
	passed   int
	failed   int
	finished bool
}

// New prepares a runner for steps, each settling for settle.
func New(steps []Step, settle time.Duration) *Runner {
	return &Runner{Steps: steps, Settle: settle, Pad: input.NewState()}
}

// PointAt points the right hand at a point of the room from beside the head, pulling the trigger
// on it when pull says so.
func (runner *Runner) PointAt(target graphics.Vector, pull bool) {
	runner.target, runner.aiming, runner.pull = target, true, pull
}

// Press presses a button for a frame.
func (runner *Runner) Press(button input.PadButton) {
	runner.toPress = append(runner.toPress, button)
}

// PressMenu presses the menu button for a frame.
func (runner *Runner) PressMenu() { runner.menu = true }

// HoldPalmUp holds the left hand's palm to the face, or lowers it, until a step says otherwise.
func (runner *Runner) HoldPalmUp(up bool) { runner.palmUp = up }

// Done tells whether every step has run.
func (runner *Runner) Done() bool { return runner.finished }

// Current is the index of the step running now (len(Steps) once done).
func (runner *Runner) Current() int { return runner.current }

// Passed and Failed count the steps so far.
func (runner *Runner) Passed() int { return runner.passed }
func (runner *Runner) Failed() int { return runner.failed }

// Drive puts this frame's hands and presses into a program's frame: the right hand pointing as the
// step wants (untracked otherwise), the left hand down unless its palm is held up, the buttons into
// pad, the menu into menu. While a step waits to be ready the hands are left as they are.
func (runner *Runner) Drive(head graphics.Pose, headTracked bool, hands *[2]openxr.Hand, pad *input.State, menu *bool) {
	if runner.finished {
		return
	}
	step := &runner.Steps[runner.current]
	if runner.started.IsZero() {
		if runner.waiting.IsZero() {
			runner.waiting = time.Now()
		}
		if !headTracked || (step.Ready != nil && !step.Ready()) {
			if time.Since(runner.waiting) > runner.patience() {
				runner.finish(ErrNeverReady)
			}
			return
		}
		runner.start(step)
	}
	hand := openxr.Hand{Tracked: true}
	if runner.aiming {
		hand.Aim = pointer.AimAt(HandBeside(head), runner.target)
		hand.Grip = hand.Aim
		since := time.Since(runner.started)
		if runner.pull && since > runner.Settle/4 && since < runner.Settle/4+PullTime {
			hand.Trigger = 1
		}
	}
	hands[rightHand] = hand
	hands[leftHand] = openxr.Hand{} // the real left hand could call up something over a step
	if runner.palmUp {
		hands[leftHand] = PalmToFace(head)
	}
	for _, button := range runner.held {
		pad.Feed().SetPadButton(button, false)
	}
	runner.held = append(runner.held[:0], runner.toPress...)
	runner.toPress = runner.toPress[:0]
	for _, button := range runner.held {
		pad.Feed().SetPadButton(button, true)
	}
	if runner.menu && menu != nil {
		*menu, runner.menu = true, false
	}
}

// Check checks the current step once it has settled, and moves on to the next.
func (runner *Runner) Check() {
	if runner.finished || runner.started.IsZero() || time.Since(runner.started) < runner.Settle {
		return
	}
	err := runner.failure
	if step := runner.Steps[runner.current]; err == nil && step.Check != nil {
		err = step.Check()
	}
	runner.finish(err)
}

// start runs a step's act, once it is ready.
func (runner *Runner) start(step *Step) {
	runner.started = time.Now()
	runner.aiming, runner.pull, runner.menu = false, false, false
	runner.toPress = runner.toPress[:0]
	if runner.BeforeEach != nil {
		runner.BeforeEach()
	}
	if step.Act != nil {
		runner.failure = step.Act()
	}
}

// finish tells how the current step went and moves on.
func (runner *Runner) finish(err error) {
	if err != nil {
		runner.failed++
	} else {
		runner.passed++
	}
	report := runner.Report
	if report == nil {
		report = logReport
	}
	report(runner.current, len(runner.Steps), runner.Steps[runner.current].Name, err)
	runner.current++
	runner.started, runner.waiting, runner.failure = time.Time{}, time.Time{}, nil
	runner.aiming, runner.pull = false, false
	runner.toPress = runner.toPress[:0]
	if runner.current == len(runner.Steps) {
		runner.finished = true
		finished := runner.Finished
		if finished == nil {
			finished = func(passed, failed int) { log.Printf("SELFTEST: %d passed, %d failed", passed, failed) }
		}
		finished(runner.passed, runner.failed)
	}
}

func (runner *Runner) patience() time.Duration {
	if runner.Patience == 0 {
		return DefaultPatience
	}
	return runner.Patience
}

func logReport(index, total int, name string, err error) {
	if err != nil {
		log.Printf("SELFTEST %d/%d FAILED: %s: %v", index+1, total, name, err)
		return
	}
	log.Printf("SELFTEST %d/%d ok: %s", index+1, total, name)
}

// HandBeside is where the self-test's right hand is: low and to the right of the head, as a hand
// held at rest pointing ahead.
func HandBeside(head graphics.Pose) graphics.Vector {
	offset := graphics.QuaternionAroundY(float64(place.HeadYaw(head))).Rotate(graphics.Vector{0.18, -0.35, -0.2, 0})
	return graphics.Vector{head.Position[0] + offset[0], head.Position[1] + offset[1], head.Position[2] + offset[2], 1}
}

// PalmToFace is a left hand low and to the left of the head, its palm (the controller's +X) turned
// to the face.
func PalmToFace(head graphics.Pose) openxr.Hand {
	offset := graphics.QuaternionAroundY(float64(place.HeadYaw(head))).Rotate(graphics.Vector{-0.15, -0.3, -0.3, 0})
	origin := graphics.Vector{head.Position[0] + offset[0], head.Position[1] + offset[1], head.Position[2] + offset[2], 1}
	grip := pointer.AimAt(origin, head.Position)
	grip.Orientation = grip.Orientation.Multiply(graphics.QuaternionAroundY(math.Pi / 2)) // +X where -Z looked: at the head
	return openxr.Hand{Tracked: true, Grip: grip, Aim: grip}
}

// AskedFor tells whether a file of that name is in the program's own folder of the headset's
// shared storage, named after its package (the process's name): a program runs its self-test
// when one is put there with adb before it starts,
//
//	adb shell touch /sdcard/Android/data/<package>/files/<name>
//
// as Android lets an application read only a few system properties of its own.
func AskedFor(name string) bool {
	command, err := os.ReadFile("/proc/self/cmdline")
	if err != nil {
		return false
	}
	pkg := strings.TrimRight(string(command), "\x00")
	_, err = os.Stat(filepath.Join("/sdcard/Android/data", pkg, "files", name))
	return err == nil
}
