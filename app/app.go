//go:build android

package app

import (
	"log"
	"time"

	"github.com/akrck02/go-renderer/android"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// sessionHaptics shakes the controllers through the session.
type sessionHaptics struct{ session *openxr.Session }

func (haptics sessionHaptics) Vibrate(hand int, amplitude float32, duration time.Duration) {
	if err := haptics.session.Vibrate(openxr.Side(hand), amplitude, duration); err != nil {
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
	if err := CheckPanels(loop.options.Panels); err != nil {
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
		Haptics:      sessionHaptics{session},
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
