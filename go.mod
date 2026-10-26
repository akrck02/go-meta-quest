module github.com/akrck02/go-meta-quest

go 1.25.6

require github.com/akrck02/go-renderer v0.0.0

require (
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// The headset's pieces of Go Renderer (android.RunHeadset, openxr, graphics.Pose, camera.Rig,
// gles.SceneRenderer.DrawViews) are not released yet: build against a checkout beside this one.
// This line goes once Go Renderer releases them.
replace github.com/akrck02/go-renderer => ../go-renderer
