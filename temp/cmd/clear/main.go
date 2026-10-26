//go:build android

// The first step on the headset: each eye sees a colour of its own, changing slowly. No renderer,
// only OpenGL ES clearing each eye's picture. It checks the whole road to the headset (the
// package, the OpenXR loader and runtime, the session, the swapchains), with nothing else in the
// way.
//
//	tools/build-apk.sh ./cmd/clear && tools/install.sh build/clear.apk
package main

import (
	"log"

	"github.com/akrck02/go-meta-quest/app"
	"github.com/akrck02/go-renderer/gles"
)

func main() {
	seconds := 0.0
	err := app.Run(app.Options{ApplicationName: "Clear"}, func(frame app.Frame) bool {
		seconds += frame.Seconds
		if !frame.ShouldRender {
			return true
		}
		for index, eye := range frame.Eyes {
			red, green, blue := eyeColour(index, seconds)
			functions := frame.Functions
			functions.BindFramebuffer(gles.FramebufferTarget, eye.Target)
			functions.SetViewport(0, 0, eye.Width, eye.Height)
			functions.SetClearColor(red, green, blue, 1)
			functions.Clear(gles.ColorBufferBit | gles.DepthBufferBit)
		}
		return true
	})
	if err != nil {
		log.Fatal(err) // Android's log, under the tag Go
	}
}
