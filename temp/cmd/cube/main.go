//go:build android

// A lit cube that stands still in the room: an arm and a half ahead of where the player first
// looked, at eye height, the same size as a real one. Walk round it, lean in, look away and back.
// A small cube sits in each controller's grip. It is Go Renderer's scene renderer drawing both
// eyes from one plan of the frame.
//
//	tools/build-apk.sh ./cmd/cube && tools/install.sh build/cube.apk
package main

import (
	"log"

	"github.com/akrck02/go-meta-quest/app"
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// The nearest and farthest the eyes see, in metres: a hand held close to the face, and the far
// side of a large room.
const (
	near = 0.05
	far  = 100
)

func main() {
	made := newWorld()
	rig := camera.Rig{} // the player's space is the world: one metre a unit, no turn, no move
	var renderer *gles.SceneRenderer
	views := make([]gles.View, 2)
	poses := make([]graphics.Pose, 2)
	fields := make([]graphics.FieldOfView, 2)

	err := app.Run(app.Options{ApplicationName: "Cube"}, func(frame app.Frame) bool {
		made.follow(rig, [2]graphics.Pose{frame.Hands[app.LeftHand].Grip, frame.Hands[app.RightHand].Grip},
			[2]bool{frame.Hands[app.LeftHand].Tracked, frame.Hands[app.RightHand].Tracked})
		if !frame.ShouldRender {
			return true
		}
		made.place(rig, frame.Head, frame.HeadTracked)
		if renderer == nil {
			created, err := gles.NewSceneRenderer(frame.Functions, overlay.Fonts{})
			if err != nil {
				log.Print(err)
				return false
			}
			renderer = created
		}
		renderer.LinearOutput = frame.SRGB
		for index, eye := range frame.Eyes {
			poses[index], fields[index] = eye.Pose, eye.Field
			views[index] = gles.View{Target: eye.Target, Width: eye.Width, Height: eye.Height,
				View:       rig.EyeView(eye.Pose),
				Projection: graphics.PerspectiveFromFieldOfView(eye.Field, near, far),
				Eye:        rig.EyePosition(eye.Pose)}
		}
		covering, aspect := camera.CoveringLens(rig, poses, fields, near, far)
		renderer.DrawViews(made.scene, covering, aspect, views)
		return true
	})
	if err != nil {
		log.Fatal(err) // Android's log, under the tag Go
	}
}
