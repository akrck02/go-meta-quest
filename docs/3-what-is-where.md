# What is where

The packages of Go for Meta Quest, the programs and tools it comes with, what you need to build for
the headset, and what was checked: so far, only on a computer.

## What is where

| Piece | What it does |
|---|---|
| `app` | The program loop. `Run` starts Go Renderer's headset activity (`android.RunHeadset`: no window, an OpenGL ES 3 context on its own thread), opens the OpenXR session there (`openxr.Open`), and goes round: it reads the session's news; while the session does not run (the headset off the head, starting, paused) it sleeps 10 ms; while it runs it begins a frame, reads the controllers (only while the program has them), calls the program's step, and ends the frame. It stops when the runtime asks the program to leave; when the step returns false or a frame fails it first asks the runtime to stop the session (`RequestExit`) and hands back empty frames until it has, two seconds at most. It closes the session on the drawing thread. Each `Frame` carries the input state (the controllers as a game pad), the head's pose and whether it is tracked, both hands, the seconds since the last frame (from the display times, at most a tenth of a second), whether the program has the controllers, whether the frame will be shown, the two eyes (pose, field of view, framebuffer), whether those encode sRGB, and OpenGL ES. `Options.Panels` asks for panels by their size in pixels (`PanelSize`, at most `openxr.MostPanels`, checked before any is made); they are made when the session opens and every `Frame.Panels` holds them in that order, to place (`Pose`, `FitWidth`, `Shown`) and draw into (`Target`, only when the picture changes). `Frame.Vibrate(hand, amplitude, duration)` shakes a controller (through `Frame.Haptics`). `Options`, `Frame` and `CheckPanels` are plain Go, built everywhere: a stand-in for a headset on a computer hands out the same `Frame`, so a program written for the headset runs there too. Off the headset `Run` hands the program to that stand-in, registered with `UseStandIn` by a package the program imports (it fails with `ErrNoHeadset` when none was), so the program itself is not changed. |
| `cmd/clear` | The first program: each eye cleared to a colour of its own going slowly round the colour wheel, the right eye half a turn behind. No renderer: it checks the package, the loader, the session and the pictures with nothing else in the way. |
| `cmd/cube` | A lit 30 cm cube 1.5 m ahead of where the player first looked, at the head's height, that stays there in the room; a 5 cm cube in each tracked controller's grip. Drawn by Go Renderer's `gles.SceneRenderer.DrawViews`: the frame planned once from a lens covering both eyes (`camera.CoveringLens`), then drawn in each eye with its own lopsided projection. |
| `cmd/room` | The cube of `cmd/cube` in the player's own room: it opens the session with `Options.Passthrough`, and while `Frame.ShowsRoom` it draws with `gles.SceneRenderer.SeeThrough`, so the headset's cameras show the room wherever the cube is not. The first step of mixed reality; on a headset without passthrough it is the cube on black. |
| `explore` | Walks a player through any `scene.Scene` at room scale: `Run(world, Options)` is the whole program loop. `camera.RigWalk` walks the rig with the left stick the way the head looks and turns it in steps around the head with the right one, the floor on the scene's walkable ground under the head (`world.GroundHeight`). `Options.WalkMetresSecond` walks faster, and `Options.TurnRadiansSecond` turns smoothly about the head instead of in steps (a program whose right stick scrolls a panel feeds it 0 sideways meanwhile). Y or B switches between a person (scale 1) and a giant (`GiantScale`, 20 by default), the head kept over the same point of the ground. The near plane grows with the scale, `Far` is the scene's; a cube in each tracked hand, as in `cmd/cube`. `Options.Start` is where the room stands at the start: the program decides. A frame's steering makes no garbage (tested); the views are made in the same slices every frame. `Show` explores another scene (the hands go with the player); `Move` puts the room somewhere else in the same one, at any scale. A program with its own loop (panels, say) calls `New`, then `Steer`, `Follow` and `Views` every frame instead of `Run`. It builds everywhere, on `app.Run`, so it runs in a stand-in too. |
| `place` | Where things stand, as sums on poses, tested anywhere. In the scene: `RigWithHeadAt` puts the head over a point of the floor looking a way, at a scale; `TurnedAround` and `RescaledAround` turn or resize the person about their head, which stays over the same point; `RoomPoint` is a point of the scene in the room and `Keeping` moves the rig so one is at a point of the room; `TurnedAbout` and `ResizedAbout` turn or resize a model about its middle; `ModelAhead` puts a scene on a table ahead of the head; `StandingBefore` sets the person down before a point, on their side, facing it. In the room: `Ahead` (a panel called up), `FacingHead` (a label), `LeaningBack` and `UnderEyes` (a strip low ahead, leaned back toward the eyes), `Following` (a yaw that follows the head only once it looks well away), `PalmFacing` and `OverPalm` (a menu over a hand whose palm is turned to the face), `HeightForAngle`; and `HeadYaw`, `YawFacing`, `AngleBetween`, `AnglesOf`. |
| `mirror` | A finger on a picture of the headset's screen points as the right controller does. `Listen` reads lines from a Unix socket (`@name` is Android's abstract namespace, reached with `adb forward`): `down`, `move` and `up` with a point of the left eye's picture, `look` with two (two fingers dragged), `mark` with one or `off`. `Touches` hands them to the frames one change at a time (a quick tap still presses and lets go), `Hand` is the controller the finger is, `Carry` carries out a drag (a panel it began on swings about the eye; elsewhere it says how far to turn the person), `MarkPose` places the mark, and `Direction` and `PicturePoint` go between the eye's picture and the room. |
| `selftest` | Goes through a program as a person would, with controllers of its own. A `Runner` runs `Step`s (ready, act, check), each settling a while before its check: `PointAt` points the right hand from beside the head at a point of the room and pulls the trigger once, `Press` presses a button for a frame, `PressMenu` the menu, `HoldPalmUp` turns the left palm to the face. A program calls `Drive` before its frame and `Check` after; in the headset it reads the runner's `Pad` instead of the real controllers. `AskedFor` tells whether a file was put in the program's folder with adb, to run it there. |
| `cmd/explorer` | Go Renderer's sample island (`sample.Island()`, its afternoon light and fog) explored with `explore`. The player starts on the ground twelve metres south of the village square, facing it, at a person's size. |
| `pointer` | Points at panels with a controller, as a laser. `FromAim` is the ray along a hand's aim (-Z); `HitPanel` finds the pixel it meets on a panel, from the side it faces, and how far. A `Pointer` feeds an interface's own `input.State` from one hand each frame: the cursor to that pixel, the trigger as the left button (pressed past 0.7, let go under 0.3), the stick up and down as the wheel (`ScrollNotchesPerSecond` at full tilt); when the ray leaves the panel a held button is let go and the cursor leaves the interface. The state's window is the panel's picture. `PointAtNearest` points one hand at several `Interface`s (a panel, its input and its pointer) at once: the nearest shown one the ray meets has it, the others let go. `SceneRay` is the hand's ray in the scene, through the rig; `AimAt` aims a hand of a program's own at a point; `PanelPoint` is a pixel of a panel in the room. A `Beam` is a thin unlit bar from the hand to what it meets, placed through the rig, hidden on an untracked hand. Pointing makes no garbage (tested). |
| `cmd/panel` | A board of six tiles floating 1.5 m ahead over the sample island, a panel drawn by `gles.FlatRenderer` (linear colour): the right controller's beam points at it, the tile under it is outlined with a light buzz, the trigger lights a tile or puts it out with a stronger one. The board is drawn again only when it changes; the sticks walk the island while it stays with you. |
| `tools/AndroidManifest.xml` | The manifest every package gets: Android's `NativeActivity` loading `libmain.so`, the `com.oculus.intent.category.VR` category (immersive, not a flat window), head tracking required, `com.oculus.supportedDevices` (Quest 3 and 3S), `com.oculus.vr.focusaware` on the activity (the program keeps running, shown, while the system's menu is up), landscape, kept out of the recent list, never recreated by a configuration change (the session holds the activity), one at a time, no Java. It names the OpenXR permissions and the broker's providers the loader asks which runtime to load. |
| `tools/fetch-loader.sh` | Reads Maven Central's newest `org.khronos.openxr:openxr_loader_for_android`, downloads it, and keeps its arm64 `libopenxr_loader.so` and its licence in `third_party/openxr-loader/` (not committed). A version can be named: `tools/fetch-loader.sh 1.1.65`. |
| `tools/build-apk.sh` | Builds a program into `build/<name>.apk`, from the folder it runs in: `go build -buildmode=c-shared` with the NDK's clang and 16 KB pages, the manifest linked by `aapt2` against the newest platform's `android.jar`, the program's library and the loader stored uncompressed under `lib/arm64-v8a/`, aligned to 16 KB (`zipalign -P 16`), signed with the debug key (made if missing). Its id is `io.github.akrck02.metaquest.<name>` and its label the folder's name, unless `APK_ID` and `APK_LABEL` say otherwise: a program in a module of its own builds with the same script, under its own id. |
| `tools/install.sh` | Installs a package on the headset with `adb install -r`, starts it (by `APK_ID` when it was built with one), and shows the log under the tags `Go` and `OpenXR`. |

What is not here is Go Renderer's, as on the 3DS: the drawing (`gles`), the activity (`android`),
the OpenXR session, its swapchains and the controllers (`openxr`), the poses and the eyes'
projections (`graphics`), the rig and the covering lens (`camera`).

## What you need

- Go 1.25 or later.
- The Android SDK's build tools (`aapt2`, `zipalign` with `-P`, so 35 or later; `apksigner`) and a
  platform, in `ANDROID_HOME`; the NDK in `ANDROID_NDK_HOME`. On a Mac, Homebrew's
  `android-commandlinetools` is found without them.
- Go Renderer's headset pieces, which `go.mod` fetches by itself: a commit of its `feature/openxr`
  branch, until they are released.
- `adb`, and a Meta Quest 3 or 3S in developer mode.

```bash
tools/build-apk.sh ./cmd/cube
tools/install.sh build/cube.apk
```

## What was checked

**On a Meta Quest 3** (Android 14, API 34, connected with adb in developer mode), from the system's
own log and the headset's screen captures (`adb exec-out screencap -p`, both eyes side by side):

- Both packages start immersive. The loader finds the runtime, and the session goes IDLE → READY →
  SYNCHRONIZED → VISIBLE → FOCUSED within a tenth of a second.
- `cmd/clear`: each eye shows its own colour, the right eye half a turn behind on the colour wheel.
  With the system's menu open the session drops to VISIBLE and the program keeps drawing under it;
  going home stops the session (STOPPING → IDLE) and the program waits without spinning.
- `cmd/cube`: the cube and both hands' cubes are drawn in each eye, over the horizon's colour,
  with near things (the hands) further apart between the eyes than far ones, as they should be.
- Frames keep the headset's pace: 72 a second, none stale, the program's processor time under 1 ms
  a frame (the runtime's own `VrApi` line in `adb logcat`).
- `cmd/room`: not checked on a headset. It builds for Android (its library compiles with the
  passthrough's C); whether the room shows behind the cube needs the headset.
- `cmd/explorer`: the island is drawn in each eye at 72 frames a second, the program's processor
  time under 1 ms a frame, seen through the headset's screen recording (`adb exec-out screenrecord`)
  with the headset lying on a desk, so without the sticks.
- `cmd/panel`: the board is shown over the island, its own colours (not washed out) and its title
  readable, at 72 frames a second, with the haptics server connected and no error in the log. The
  headset lay on a desk: pointing, clicking and the buzzes are not tried with hands yet.

With the headset on, the cube was seen working as it should. Leaving from the system's menu (Quit)
is not tried yet.

On a Mac (Go 1.27, NDK 29, build tools 37, platform android-37, OpenXR loader 1.1.65):

- `go vet ./...` and `go test ./...` pass: the loop's decisions (leave, wait, frame), the clock
  (the first frame, 72 frames a second, a long pause, time going back, a restart), the eyes'
  colours, the cube's world (its size, faces pointing out, placed ahead at eye height and
  staying there, the hands' cubes following tracked hands only), the island's start (on the ground south of
  the square), and the exploration's (the options' defaults, the giant and back with the head over the same point, Y waiting
  for a tracked head, a second of the stick walking 1.5 m on the ground, the views and their
  covering lens, no allocations while steering).
- `go vet` for Android (arm64, with the NDK's clang) passes on every package.
- `tools/fetch-loader.sh` downloads the loader; `tools/build-apk.sh` builds `build/clear.apk`,
  `build/cube.apk` and `build/explorer.apk`. Each holds the manifest, `libmain.so` and `libopenxr_loader.so` stored
  uncompressed; `zipalign -c -P 16` and `apksigner verify` accept them, and both libraries' segments
  are aligned to 16 KB.
- On an Android 16 phone emulator (arm64), both packages install and start: the Go runtime runs
  `main`, the drawing thread gets its OpenGL ES 3 context, and the OpenXR loader starts and asks the
  runtime broker for a runtime. A phone has none, so the program ends there, cleanly, with
  `openxr.instance-not-created` in the log (`adb logcat -s Go`).

The manifest's SDK levels (minimum 32, target 34) follow Meta's documentation as last read; the
headset above runs API 34.

## Not yet

- Walking, turning and the giant tried with the headset on.
- Pointing at a panel, clicking and the buzzes, tried with the headset on; a Go UI interface on a
  panel (the pieces are there: `pointer` feeds an `input.State` as Go UI reads it).
- The headset's own settings: refresh rate, foveated rendering, passthrough.
- Drawing both eyes in one pass (multiview), sound, storage.
- A frame entirely without garbage: `camera.CoveringLens` (Go Renderer's) still makes two small
  slices a frame, and `explore` cannot avoid them.
- Programs on other headsets: the OpenXR part is standard, but the manifest is Meta's.
