//go:build android

// A board floating in front of you over Go Renderer's sample island: point at it with the right
// controller, a beam showing where, and pull the trigger to light a tile or put it out; the
// controller gives a short buzz for each. The board is a panel, a picture the headset places in
// the room itself, and stays with you as the sticks walk you around the island.
//
//	tools/build-apk.sh ./cmd/panel && tools/install.sh build/panel.apk
package main

import (
	"log"
	"time"

	"github.com/akrck02/go-meta-quest/app"
	"github.com/akrck02/go-meta-quest/explore"
	"github.com/akrck02/go-meta-quest/pointer"
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

// boardPose is where the board stands in the room: a step and a half ahead, its middle a little
// under a standing person's eyes, facing them.
var boardPose = graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 1.3, -1.5, 1}}

// demo is the program's state between frames.
type demo struct {
	explorer *explore.Explorer
	board    *board
	pointer  pointer.Pointer
	beam     *pointer.Beam
	panel    *input.State // the board's interface's own input: its cursor and button
	scene    *gles.SceneRenderer
	flat     *gles.FlatRenderer
	stale    bool // the board's picture changed since it was last drawn
}

func main() {
	world := sample.Island()
	made := &demo{board: newBoard(), beam: pointer.NewBeam(graphics.Vector{0.4, 0.8, 1, 1}), stale: true, panel: input.NewState()}
	made.explorer = explore.New(world, explore.Options{Start: startRig(world)})
	world.Nodes = append(world.Nodes, made.beam.Node)
	options := app.Options{ApplicationName: "Panel", Panels: []app.PanelSize{{Width: boardWidth, Height: boardHeight}}}
	if err := app.Run(options, made.frame); err != nil {
		log.Fatal(err) // Android's log, under the tag Go
	}
}

// startRig stands the room on the island's ground at its middle.
func startRig(world *scene.Scene) camera.Rig {
	rig := camera.Rig{Position: graphics.Vector{0, 0, 0, 1}, Scale: explore.PersonScale}
	if ground, found := world.GroundHeight(0, 0); found {
		rig.Position[1] = ground
	}
	return rig
}

func (made *demo) frame(frame app.Frame) bool {
	panel := frame.Panels[0]
	panel.Pose = boardPose
	panel.FitWidth(boardMetres)
	panel.Shown = true
	made.explorer.Steer(frame.Input, frame.Seconds, frame.Head, frame.HeadTracked)
	made.point(frame)
	made.explorer.Follow([2]graphics.Pose{frame.Hands[app.LeftHand].Grip, frame.Hands[app.RightHand].Grip},
		[2]bool{frame.Hands[app.LeftHand].Tracked, frame.Hands[app.RightHand].Tracked})
	if !frame.ShouldRender {
		return true
	}
	if !made.prepare(frame) {
		return false
	}
	made.drawBoard(frame)
	made.scene.LinearOutput = frame.SRGB
	views, covering, aspect := made.explorer.Views(frame.Eyes)
	made.scene.DrawViews(made.explorer.World(), covering, aspect, views)
	return true
}

// point aims the right hand at the board: the beam reaches it, the tile under it lights its border,
// and a pull of the trigger lights the tile with a buzz.
func (made *demo) point(frame app.Frame) {
	hand := frame.Hands[app.RightHand]
	hit, pointing := made.pointer.Update(made.panel, hand, frame.Panels[0], frame.Seconds)
	made.beam.Place(made.explorer.Rig(), hand, hit.Distance)
	if made.board.hover(hit.X, hit.Y, pointing) {
		made.stale = true
		if made.board.hovered >= 0 {
			frame.Vibrate(app.RightHand, 0.15, 10*time.Millisecond)
		}
	}
	if made.panel.ButtonPressed(input.MouseLeft) && made.board.click() {
		made.stale = true
		frame.Vibrate(app.RightHand, 0.6, 40*time.Millisecond)
	}
	made.panel.Feed().EndFrame()
}

// prepare makes the renderers the first time a frame is drawn, where OpenGL ES is current.
func (made *demo) prepare(frame app.Frame) bool {
	if made.scene != nil {
		return true
	}
	scene, err := gles.NewSceneRenderer(frame.Functions, overlay.Fonts{})
	if err != nil {
		log.Print(err)
		return false
	}
	flat, err := gles.NewFlatRenderer(frame.Functions, overlay.Fonts{})
	if err != nil {
		log.Print(err)
		return false
	}
	made.scene, made.flat = scene, flat
	return true
}

// drawBoard draws the board's picture again when it changed; the headset keeps showing the last.
func (made *demo) drawBoard(frame app.Frame) {
	if !made.stale {
		return
	}
	target, err := frame.Panels[0].Target()
	if err != nil {
		log.Print(err)
		return
	}
	made.flat.Target, made.flat.LinearOutput = target, frame.SRGB
	made.flat.Draw(made.board.draw(), boardColour, boardWidth, boardHeight, 1)
	made.stale = false
}
