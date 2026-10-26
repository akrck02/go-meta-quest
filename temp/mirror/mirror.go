// Package mirror lets a finger on a picture of the headset's screen (its mirror, cast to a phone or
// a computer) point as the right controller does: a touch is a laser from the left eye through the
// point touched, a finger down is the trigger pulled. So a program's panels and scene answer a
// touch with what they already do for a controller, and someone without the headset on can drive
// it. Two fingers dragged turn the person, or carry a panel begun on; a mark (a dot at a point of
// the eye's picture) shows the one casting where each point of the picture is in theirs.
//
// The touches arrive as lines of text over a connection (Listen), all coordinates from 0 to 1 across
// the left eye's picture, from its top-left corner:
//
//	down 0.42 0.51        a finger down at a point
//	move 0.43 0.52        it moved (a mouse moves with no button held)
//	up 0.43 0.52          it was lifted
//	look 0.4 0.5 0.6 0.5  two fingers dragged from one point to another
//	mark 0.3 0.4          show the mark at a point ("mark off" hides it)
package mirror

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/akrck02/go-meta-quest/pointer"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/openxr"
)

// Lingers is how long the finger keeps the controller's place after its last touch: long enough for
// a hover from a mouse, short enough for the real controller to take over soon.
const Lingers = time.Second

// Touch is a finger on the left eye's picture: X from its left edge to its right, Y from its top to
// its bottom, both 0 to 1, and whether the finger is down.
type Touch struct {
	X, Y float32
	Down bool
}

// Drag is two fingers dragged over the picture, from one point to another.
type Drag struct {
	From, To Touch
}

// ParseTouch reads a down, move or up line. A move keeps the finger as it was, down or not.
func ParseTouch(line string, wasDown bool) (Touch, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return Touch{}, fmt.Errorf("mirror: not a touch: %q", line)
	}
	x, errX := strconv.ParseFloat(fields[1], 32)
	y, errY := strconv.ParseFloat(fields[2], 32)
	if errX != nil || errY != nil {
		return Touch{}, fmt.Errorf("mirror: not a touch: %q", line)
	}
	touch := Touch{X: float32(x), Y: float32(y)}
	switch fields[0] {
	case "down":
		touch.Down = true
	case "up":
	case "move":
		touch.Down = wasDown
	default:
		return Touch{}, fmt.Errorf("mirror: not a touch: %q", line)
	}
	return touch, nil
}

// Touches gathers the lines from the connection and hands them to the frames one change at a time:
// a quick tap, down and up between two frames, still presses and lets go. It is safe to add from
// one goroutine while the frames take from another.
type Touches struct {
	mutex   sync.Mutex
	pending []Touch
	down    bool // the finger as the last line read left it, for a move
	current Touch
	last    time.Time // when a frame last took a touch
	mark    Touch     // where the mark is; Down while it is shown
	drags   []Drag
}

// Add reads a line.
func (gathered *Touches) Add(line string) error {
	gathered.mutex.Lock()
	defer gathered.mutex.Unlock()
	switch {
	case strings.HasPrefix(line, "mark"):
		return gathered.setMark(line)
	case strings.HasPrefix(line, "look"):
		return gathered.addDrag(line)
	}
	touch, err := ParseTouch(line, gathered.down)
	if err != nil {
		return err
	}
	gathered.down = touch.Down
	gathered.pending = append(gathered.pending, touch)
	return nil
}

func (gathered *Touches) setMark(line string) error {
	if strings.TrimSpace(line) == "mark off" {
		gathered.mark.Down = false
		return nil
	}
	mark, err := ParseTouch(strings.Replace(line, "mark", "down", 1), false)
	if err != nil {
		return err
	}
	gathered.mark = mark
	return nil
}

func (gathered *Touches) addDrag(line string) error {
	fields := strings.Fields(line)
	if len(fields) != 5 {
		return fmt.Errorf("mirror: not a drag: %q", line)
	}
	var points [4]float32
	for index, field := range fields[1:] {
		value, err := strconv.ParseFloat(field, 32)
		if err != nil {
			return fmt.Errorf("mirror: not a drag: %q", line)
		}
		points[index] = float32(value)
	}
	gathered.drags = append(gathered.drags, Drag{From: Touch{X: points[0], Y: points[1]}, To: Touch{X: points[2], Y: points[3]}})
	return nil
}

// Next is where the finger is this frame, and whether it holds the controller's place: the moves
// that came together are one, a press or a release is never skipped, and a finger gone for Lingers
// gives the place back.
func (gathered *Touches) Next(now time.Time) (Touch, bool) {
	gathered.mutex.Lock()
	defer gathered.mutex.Unlock()
	if len(gathered.pending) > 0 {
		taken := 1
		for taken < len(gathered.pending) && gathered.pending[taken].Down == gathered.current.Down &&
			gathered.pending[0].Down == gathered.current.Down {
			taken++
		}
		gathered.current, gathered.last = gathered.pending[taken-1], now
		gathered.pending = gathered.pending[taken:]
	}
	if gathered.last.IsZero() {
		return Touch{}, false
	}
	return gathered.current, gathered.current.Down || now.Sub(gathered.last) < Lingers
}

// Marked is where the mark is to be shown, if it is.
func (gathered *Touches) Marked() (Touch, bool) {
	gathered.mutex.Lock()
	defer gathered.mutex.Unlock()
	return gathered.mark, gathered.mark.Down
}

// TakeDrags hands over the drags of two fingers since the last frame.
func (gathered *Touches) TakeDrags() []Drag {
	gathered.mutex.Lock()
	defer gathered.mutex.Unlock()
	drags := gathered.drags
	gathered.drags = nil
	return drags
}

// Listen reads the lines of whoever connects to a Unix socket at address, for as long as the
// program runs, one connection at a time (only one picture of the headset is touched). An address
// starting with @ is in Android's abstract namespace, which needs no permission and which adb
// reaches from the computer: adb forward tcp:8792 localabstract:<the address without @>. Lines that
// cannot be read are told to report, and the connection goes on; it returns only if it cannot
// listen or accept.
func Listen(address string, gathered *Touches, report func(error)) error {
	listener, err := net.Listen("unix", address)
	if err != nil {
		return err
	}
	for {
		connection, err := listener.Accept()
		if err != nil {
			return err
		}
		lines := bufio.NewScanner(connection)
		for lines.Scan() {
			if err := gathered.Add(lines.Text()); err != nil && report != nil {
				report(err)
			}
		}
		connection.Close()
	}
}

// Hand is the controller a finger is: aiming from the eye through the point touched, its trigger
// pulled while the finger is down.
func Hand(eye openxr.Eye, touch Touch) openxr.Hand {
	origin := eye.Pose.Position
	origin[3] = 1
	direction := Direction(eye, touch.X, touch.Y)
	target := graphics.Vector{origin[0] + direction[0], origin[1] + direction[1], origin[2] + direction[2], 1}
	hand := openxr.Hand{Tracked: true, Aim: pointer.AimAt(origin, target)}
	hand.Grip = hand.Aim
	if touch.Down {
		hand.Trigger = 1
	}
	return hand
}

// Direction is the way a point of the eye's picture is seen, in the room: the picture spans the
// eye's field of view, edge to edge.
func Direction(eye openxr.Eye, x, y float32) graphics.Vector {
	left, right := tangent(eye.Field.LeftRadians), tangent(eye.Field.RightRadians)
	up, down := tangent(eye.Field.UpRadians), tangent(eye.Field.DownRadians)
	local := graphics.Vector{left + x*(right-left), up + y*(down-up), -1, 0}
	return eye.Pose.Orientation.Rotate(local).Normalize()
}

// PicturePoint is where a point of the room is in the eye's picture, as Direction reads it; false
// behind the eye. A program tells where its panel's corners are with it, for a picture of the
// screen to be matched to the eye's.
func PicturePoint(eye openxr.Eye, point graphics.Vector) (float32, float32, bool) {
	offset := graphics.Vector{point[0] - eye.Pose.Position[0], point[1] - eye.Pose.Position[1], point[2] - eye.Pose.Position[2], 0}
	local := eye.Pose.Orientation.Conjugate().Rotate(offset)
	if local[2] >= 0 {
		return 0, 0, false
	}
	left, right := tangent(eye.Field.LeftRadians), tangent(eye.Field.RightRadians)
	up, down := tangent(eye.Field.UpRadians), tangent(eye.Field.DownRadians)
	across, upward := local[0]/-local[2], local[1]/-local[2]
	return (across - left) / (right - left), (up - upward) / (up - down), true
}

// MarkPose is where the mark floats in the room: metres ahead of the eye through its point of the
// picture, turned as the eye is.
func MarkPose(eye openxr.Eye, mark Touch, metres float32) graphics.Pose {
	direction := Direction(eye, mark.X, mark.Y)
	position := eye.Pose.Position
	return graphics.Pose{Orientation: eye.Pose.Orientation, Position: graphics.Vector{position[0] + direction[0]*metres,
		position[1] + direction[1]*metres, position[2] + direction[2]*metres, 1}}
}

// Carry carries out a drag of two fingers so that what was under them stays under them: one begun
// on a shown panel swings the panel about the eye, and the turn is 0; anywhere else the turn is
// how far the person should turn about the vertical (positive to the left; looking up and down is
// the head's own), for the program to turn them.
func Carry(drag Drag, eye openxr.Eye, panel *openxr.Panel) (turn float32) {
	from, to := Direction(eye, drag.From.X, drag.From.Y), Direction(eye, drag.To.X, drag.To.Y)
	origin := eye.Pose.Position
	origin[3] = 1
	if panel != nil && panel.Shown {
		if _, onPanel := pointer.HitPanel(pointer.Ray{Origin: origin, Direction: from}, panel); onPanel {
			panel.Pose = SwungAbout(panel.Pose, origin, from, to)
			return 0
		}
	}
	yaw := func(direction graphics.Vector) float64 {
		return math.Atan2(-float64(direction[0]), -float64(direction[2]))
	}
	return float32(math.Remainder(yaw(from)-yaw(to), 2*math.Pi))
}

// SwungAbout moves a pose about a point (the eye) by the turn from one direction to another, and
// faces it to the point again, upright.
func SwungAbout(pose graphics.Pose, centre, from, to graphics.Vector) graphics.Pose {
	axis := graphics.Vector{from[1]*to[2] - from[2]*to[1], from[2]*to[0] - from[0]*to[2], from[0]*to[1] - from[1]*to[0], 0}
	turn := graphics.Quaternion{X: axis[0], Y: axis[1], Z: axis[2], W: 1 + from[0]*to[0] + from[1]*to[1] + from[2]*to[2]}
	length := float32(math.Sqrt(float64(turn.X*turn.X + turn.Y*turn.Y + turn.Z*turn.Z + turn.W*turn.W)))
	if length < 1e-6 {
		return pose // the very opposite way: no single turn to choose
	}
	turn = graphics.Quaternion{X: turn.X / length, Y: turn.Y / length, Z: turn.Z / length, W: turn.W / length}
	offset := turn.Rotate(graphics.Vector{pose.Position[0] - centre[0], pose.Position[1] - centre[1], pose.Position[2] - centre[2], 0})
	position := graphics.Vector{centre[0] + offset[0], centre[1] + offset[1], centre[2] + offset[2], 1}
	// facing the centre: its -Z away from it, so its face, +Z, toward it
	return graphics.Pose{Orientation: pointer.AimAt(centre, position).Orientation, Position: position}
}

func tangent(radians float32) float32 { return float32(math.Tan(float64(radians))) }
