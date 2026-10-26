//go:build android

package app

import (
	"log"
	"time"

	"github.com/akrck02/go-renderer/android"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// Options tune the program's session.
type Options struct {
	// ApplicationName is the name the program gives the OpenXR runtime.
	ApplicationName string
	// Hidden is called when the program leaves the headset's view (another application, the
	// headset put to sleep): the system may end it after that without a word, so it is the moment
	// to keep what must survive.
	Hidden func()
	// Shown is called when the program comes back into view, and at its start.
	Shown func()
	// Passthrough shows the room behind the scene, when the headset can: draw the eyes with
	// gles.SceneRenderer.SeeThrough, and the room shows wherever nothing is drawn. Frame.ShowsRoom
	// tells whether it is shown.
	Passthrough bool
	// Panels are the flat pictures the program shows in the world (interfaces, signs), by their
	// size in pixels: they are made when the session opens and handed out in every Frame, in this
	// order. At most openxr.MostPanels.
	Panels []PanelSize
}

// The hands, as Frame.Hands holds them.
const (
	LeftHand  = 0
	RightHand = 1
)

// Frame is what a program gets every frame.
type Frame struct {
	// Input is the controllers as a game pad, the same as on a desktop: the left stick is Stick,
	// the right one RightStick, A, B, X and Y are the pad's buttons, the menu button is back.
	// Nothing is held while the program does not have the controls (Focused is false).
	Input *input.State
	// Head is where the head is and where it looks, in metres in the player's space: the floor
	// under where the player stood when the program started, -Z ahead. Until the headset knows
	// (HeadTracked) it is a guess, or at the origin looking ahead.
	Head graphics.Pose
	// HeadTracked tells whether the headset knows both where the head is and where it looks this
	// frame: wait for it before placing anything in the room by the head.
	HeadTracked bool
	// Hands are the controllers in the same space, Hands[LeftHand] and Hands[RightHand]: where they
	// are (Grip), where they point (Aim), the trigger and the grip button. A hand that is not
	// tracked says so (Tracked false).
	Hands [2]openxr.Hand
	// Controls is the controllers whole, as Hands, plus the menu button.
	Controls openxr.Controls
	// Seconds is the time since the previous frame, from the times the pictures are shown; 0 for
	// the first frame, and never more than a tenth of a second, even after a pause.
	Seconds float64
	// Focused tells whether the program has the controllers: false while the system's menu is up.
	// Frames still come; the program keeps drawing.
	Focused bool
	// ShouldRender tells whether the runtime will show this frame. When it is false, the eyes
	// have nowhere to draw: a program updates what it must and draws nothing.
	ShouldRender bool
	// Eyes are the two pictures to draw, left then right: each with the eye's pose, its field of
	// view (wider on the outer side), and a framebuffer with a depth buffer of Width × Height
	// pixels.
	Eyes [2]openxr.Eye
	// SRGB tells whether the eyes' framebuffers encode sRGB themselves: draw linear colour into them
	// (gles.SceneRenderer.LinearOutput), or the colours come out twice as light.
	SRGB bool
	// ShowsRoom tells whether the room is shown behind the scene (Options.Passthrough, on a
	// headset that has it): draw with gles.SceneRenderer.SeeThrough while it is true.
	ShowsRoom bool
	// Functions is OpenGL ES, on the thread that draws.
	Functions gles.Functions
	// Panels are the panels Options.Panels asked for, in its order, the same every frame. Set where
	// each is (Pose, its size, Shown) whenever it moves; draw into it (Target) only in a frame that
	// is drawn (ShouldRender) and only when its picture changes: the headset keeps showing the last.
	Panels []*openxr.Panel

	session *openxr.Session
}

// Vibrate shakes a hand's controller (LeftHand or RightHand), amplitude from 0 to 1, for a time: a
// touch the hand feels when what it points at answers. Nothing happens while the program does not
// have the controls.
func (frame Frame) Vibrate(hand int, amplitude float32, duration time.Duration) {
	if frame.session == nil {
		return
	}
	if err := frame.session.Vibrate(openxr.Side(hand), amplitude, duration); err != nil {
		log.Print(err)
	}
}

// Run opens the headset's session and calls step once a frame, at the headset's own pace (72
// frames a second, or 90 or 120), until step returns false or the system asks the program to end.
// When the program ends it (step returns false, or a frame fails), it asks the runtime to stop the
// session and waits for it, at most exitPatience, before closing. It must be called from main, and
// returns once the program is to end.
func Run(options Options, step func(frame Frame) bool) error {
	loop := &loop{options: options, step: step, state: input.NewState()}
	err := android.RunHeadset(android.HeadsetOptions{Hidden: options.Hidden, Shown: options.Shown}, loop.frame)
	if err == nil {
		err = loop.err
	}
	if err != nil {
		android.LogError(err.Error()) // the program usually exits next, before standard error reaches the log
	}
	return err
}

// loop is Run's state between the calls of the headset's drawing thread.
type loop struct {
	options Options
	step    func(frame Frame) bool
	session *openxr.Session
	state   *input.State
	clock   clock
	err     error // what ended the loop, when something went wrong
	// leavingSince is when the program asked the runtime to stop the session; zero while it has not.
	leavingSince time.Time
	panels       []*openxr.Panel
}

// frame is called again and again on the drawing thread; false ends the program.
func (loop *loop) frame(headset android.Headset, functions android.GL) bool {
	if loop.session == nil && !loop.open(headset) {
		return false
	}
	session := loop.session
	session.PollEvents()
	if loop.leaving() {
		return loop.keepLeaving()
	}
	switch nextMove(session.Exiting(), session.Running()) {
	case moveStop:
		loop.close()
		return false
	case moveWait:
		loop.clock.restart()
		time.Sleep(idlePause)
		return true
	}
	if !loop.makeFrame(functions) {
		return loop.leave()
	}
	return true
}

// open opens the session the first time a frame is asked for, when the headset's context exists.
func (loop *loop) open(headset android.Headset) bool {
	session, err := openxr.Open(headset, openxr.Options{ApplicationName: loop.options.ApplicationName, Passthrough: loop.options.Passthrough})
	if err != nil {
		loop.err = err
		return false
	}
	loop.session = session
	if !loop.makePanels() {
		loop.close() // the session has not begun: nothing to ask the runtime first
		return false
	}
	return true
}

// makePanels makes the panels the program asked for, once the session is open.
func (loop *loop) makePanels() bool {
	if err := checkPanels(loop.options.Panels); err != nil {
		loop.err = err
		return false
	}
	for _, size := range loop.options.Panels {
		panel, err := loop.session.NewPanel(size.Width, size.Height)
		if err != nil {
			loop.err = err
			return false
		}
		loop.panels = append(loop.panels, panel)
	}
	return true
}

// makeFrame waits for the runtime's turn, lets the program step and draw, and hands the frame back.
// It says whether the program goes on.
func (loop *loop) makeFrame(functions android.GL) bool {
	session := loop.session
	shown, err := session.BeginFrame()
	if err != nil {
		loop.err = err
		return false
	}
	frame := loop.describe(shown, functions)
	goOn := loop.step(frame)
	if err := session.EndFrame(shown); err != nil {
		loop.err = err
		return false
	}
	loop.state.Feed().EndFrame()
	return goOn
}

// describe makes the program's Frame from OpenXR's, reading the controllers when the program has
// them.
func (loop *loop) describe(shown openxr.Frame, functions android.GL) Frame {
	session := loop.session
	focused := session.Focused()
	var controls openxr.Controls // nothing held, nothing tracked
	if focused {
		controls = session.ReadControls(shown)
	}
	// An unfocused frame applies empty controls, so a button held as the menu came up is let go
	// rather than held until the program has the controls again.
	controls.Apply(loop.state)
	return Frame{
		Input:        loop.state,
		Head:         shown.Head,
		HeadTracked:  shown.HeadTracked,
		Hands:        [2]openxr.Hand{controls.Left, controls.Right},
		Controls:     controls,
		Seconds:      loop.clock.advance(shown.DisplayTime),
		Focused:      focused,
		ShouldRender: shown.ShouldRender,
		Eyes:         shown.Eyes,
		SRGB:         session.SRGB(),
		ShowsRoom:    session.ShowsRoom(),
		Functions:    functions,
		Panels:       loop.panels,
		session:      session,
	}
}

// leaving tells whether the program asked the runtime to stop the session.
func (loop *loop) leaving() bool { return !loop.leavingSince.IsZero() }

// leave starts ending a session the program is done with. A running session is not closed under
// the runtime: it is asked to stop, and the loop hands back empty frames until it has. A lost
// session, or one the runtime will not stop, has nothing to wait for and is closed at once.
func (loop *loop) leave() bool {
	if loop.session.Exiting() {
		loop.close()
		return false
	}
	if err := loop.session.RequestExit(); err != nil {
		log.Print(err)
		loop.close()
		return false
	}
	loop.leavingSince = time.Now()
	return true
}

// keepLeaving waits for the session the program asked to stop: an empty frame each turn, until the
// session is finished or the runtime has taken too long.
func (loop *loop) keepLeaving() bool {
	if leavingMove(loop.session.Finished(), time.Since(loop.leavingSince)) == moveStop {
		loop.close()
		return false
	}
	if !loop.makeEmptyFrame() {
		loop.close()
		return false
	}
	return true
}

// makeEmptyFrame begins a frame and ends it with nothing to show. It says whether that worked; a
// failure is only logged, the program is leaving anyway.
func (loop *loop) makeEmptyFrame() bool {
	session := loop.session
	shown, err := session.BeginFrame()
	if err != nil {
		log.Print(err)
		return false
	}
	if err := session.EndFrame(openxr.Frame{DisplayTime: shown.DisplayTime}); err != nil {
		log.Print(err)
		return false
	}
	return true
}

// close ends the session on the drawing thread, while its OpenGL ES context is still current.
func (loop *loop) close() {
	if loop.session == nil {
		return
	}
	loop.session.Close()
	loop.session = nil
}
