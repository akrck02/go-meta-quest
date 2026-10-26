package explore

import (
	"log"

	"github.com/akrck02/go-meta-quest/app"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// Run explores a scene in the headset until the system ends the program (app.Run). It must be
// called from main.
func Run(world *scene.Scene, options Options) error {
	made := New(world, options)
	var renderer *gles.SceneRenderer
	return app.Run(app.Options{ApplicationName: options.ApplicationName}, func(frame app.Frame) bool {
		made.Steer(frame.Input, frame.Seconds, frame.Head, frame.HeadTracked)
		if options.Update != nil {
			options.Update(made, frame.Input)
		}
		made.Follow([2]graphics.Pose{frame.Hands[app.LeftHand].Grip, frame.Hands[app.RightHand].Grip},
			[2]bool{frame.Hands[app.LeftHand].Tracked, frame.Hands[app.RightHand].Tracked})
		if !frame.ShouldRender {
			return true
		}
		if renderer == nil {
			created, err := gles.NewSceneRenderer(frame.Functions, overlay.Fonts{})
			if err != nil {
				log.Print(err)
				return false
			}
			renderer = created
		}
		renderer.LinearOutput = frame.SRGB
		views, covering, aspect := made.Views(frame.Eyes)
		renderer.DrawViews(made.world, covering, aspect, views)
		return true
	})
}
