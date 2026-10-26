//go:build android

// Go Renderer's sample island around you, at room scale: you stand on the ground a short walk from
// the village square, a person among the houses. The left stick walks the way you look, the right
// stick turns you in steps (smooth turning makes people queasy); walking in the room walks on the
// island too. Y or B makes you a giant, twenty times your size, with the whole island a model at
// your feet, and back again. A small cube sits in each controller's grip.
//
//	tools/build-apk.sh ./cmd/explorer && tools/install.sh build/explorer.apk
package main

import (
	"log"

	"github.com/akrck02/go-meta-quest/explore"
	"github.com/akrck02/go-renderer/scene/sample"
)

func main() {
	world := sample.Island()
	err := explore.Run(world, explore.Options{ApplicationName: "Explorer", Start: startRig(world)})
	if err != nil {
		log.Fatal(err) // Android's log, under the tag Go
	}
}
