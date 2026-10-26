<p align="center"><img alt="" src="docs/images/logo.svg" width="120"></p>

# Go for Meta Quest

> **Note:** this project was written with an AI assistant and is pending thorough review. See the [AI disclosure](#ai-disclosure).

Go programs on the **Meta Quest 3**, as packages installed with `adb`. The drawing and the headset's
OpenXR session are [Go Renderer](https://github.com/akrck02/go-renderer)'s (`openxr`, `android`,
`gles`); this repository holds the rest of the platform: the program loop (`app`), the tools that
make a package, pointing at panels with a controller (`pointer`), and four programs: a colour in
each eye, a cube that stands still in the room, Go Renderer's sample island to walk with the
controllers, as a person or as a giant, and a board floating over it to click with a laser.

```bash
tools/build-apk.sh ./cmd/cube      # build/cube.apk
tools/install.sh build/cube.apk    # on the headset, in developer mode
```

It needs Go, the Android SDK and NDK, and a headset in developer mode. Everything is in
[docs/](docs/1-overview.md); to start, [the headset quest](docs/2-the-quest.md).

**Nothing here has run on a headset yet.** The programs build into packages and the tests pass on a
computer; see [what was checked](docs/3-what-is-where.md#what-was-checked).

`go.mod` builds against a Go Renderer checkout beside this one
(`replace github.com/akrck02/go-renderer => ../go-renderer`), because the headset's pieces are not
released yet. The line goes once Go Renderer releases them.

## License

MIT, in [LICENSE](LICENSE). A package built from this repository also carries the Go runtime,
Khronos's OpenXR loader and Go modules; their notices are in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md), to keep next to any package you share.

## AI disclosure

I write most of my code by hand, so I want to be clear about this project: **it was written by an
AI assistant (Claude, by Anthropic) under my direction.** Every commit made this way carries a
`Co-Authored-By: Claude` line.

**Status: pending thorough review.** The code compiles and its tests pass, but I have not yet
reviewed it line by line, and it has not yet run on a headset. Treat it as a working draft, not as
code I vouch for.
