package app

import (
	"time"

	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/openxr"
)

// What a program gets from the headset, on any platform: Run hands these out on a Meta Quest, and a
// stand-in for a headset on a computer can hand out the same, so one program runs on both.

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
	// Haptics shakes the controllers: the session's on a headset; nil where there is nothing to
	// shake.
	Haptics Haptics
}

// Haptics is what shakes the controllers: the headset's session, or whatever stands in for it.
type Haptics interface {
	Vibrate(hand int, amplitude float32, duration time.Duration)
}

// Vibrate shakes a hand's controller (LeftHand or RightHand), amplitude from 0 to 1, for a time: a
// touch the hand feels when what it points at answers. Nothing happens while the program does not
// have the controls, nor where nothing can shake.
func (frame Frame) Vibrate(hand int, amplitude float32, duration time.Duration) {
	if frame.Haptics == nil {
		return
	}
	frame.Haptics.Vibrate(hand, amplitude, duration)
}
