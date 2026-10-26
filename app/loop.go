// Package app runs a program on the Meta Quest 3: it opens an OpenXR session on Go Renderer's
// headset activity, follows the session as the headset starts, pauses and ends it, reads the
// controllers into Go Renderer's input.State every frame, and hands the program each eye to draw.
//
// The loop itself (Run) is for Android only; the decisions it makes are plain Go, tested anywhere.
package app

import (
	"fmt"
	"time"

	"github.com/akrck02/go-renderer/openxr"
)

// move is what the loop does next, after reading the session's news.
type move int

const (
	// moveStop ends the program: the runtime asked it to leave.
	moveStop move = iota + 1
	// moveWait sleeps a little: the session is not running (the headset is off the head, or still
	// starting), so there are no frames to make and xrWaitFrame would not pace the loop.
	moveWait
	// moveFrame makes a frame.
	moveFrame
	// moveEmptyFrame hands back a frame with nothing in it: the program asked the runtime to stop
	// the session and waits for it to, and a running session must keep getting its frames.
	moveEmptyFrame
)

// idlePause is how long the loop sleeps while the session does not run: short enough to notice the
// session starting within a frame or so, long enough not to keep a core busy for nothing.
const idlePause = 10 * time.Millisecond

// exitPatience is the longest the loop waits for the runtime to stop a session the program asked
// to leave: a runtime that never answers must not keep the program from ending.
const exitPatience = 2 * time.Second

// nextMove decides what the loop does with the session as it now is. Leaving wins over running:
// the runtime may still say "running" in the frame where it asks the program to end.
func nextMove(exiting, running bool) move {
	if exiting {
		return moveStop
	}
	if !running {
		return moveWait
	}
	return moveFrame
}

// leavingMove decides what the loop does after the program asked to leave: stop once the session
// is finished or the runtime took longer than exitPatience to stop it, else hand back an empty frame.
func leavingMove(finished bool, waited time.Duration) move {
	if finished || waited >= exitPatience {
		return moveStop
	}
	return moveEmptyFrame
}

// PanelSize is a panel's picture in pixels (Options.Panels). A panel about a metre wide at arm's
// length reads well at around a thousand pixels across.
type PanelSize struct {
	Width, Height int
}

// CheckPanels says what is wrong with the panels asked for, before any is made: a session holds
// only so many, and a panel needs a picture.
func CheckPanels(sizes []PanelSize) error {
	if len(sizes) > openxr.MostPanels {
		return fmt.Errorf("app: %d panels asked for, a session has at most %d", len(sizes), openxr.MostPanels)
	}
	for index, size := range sizes {
		if size.Width <= 0 || size.Height <= 0 {
			return fmt.Errorf("app: panel %d is %d × %d pixels, it needs a size", index, size.Width, size.Height)
		}
	}
	return nil
}
