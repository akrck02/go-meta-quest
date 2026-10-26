<p align="center"><img alt="" src="images/logo.svg" width="120"></p>

# Welcome to Go for Meta Quest 👋

**Go for Meta Quest runs Go programs on the Meta Quest 3**: a picture in each eye, a 3D scene that
stays put in your room as you walk round it, and the controllers in your hands.

It is the headset's side of the family: the program's loop, the controllers, and the tools that turn
a Go program into a package the headset installs. Go Renderer draws, and talks to the headset through
OpenXR, the open standard most headsets speak. Programs are built with the ordinary Go compiler and
Android's NDK: no game engine, no Java.

> 🚧 **A first step.** The programs build, and their tests pass on a computer, but nothing has run
> on a headset yet. [What was checked](3-what-is-where.md#what-was-checked) says exactly what.

## Is it for me?

- ✅ You have a **Meta Quest 3** (or 3S) and want to put **your own program** in it.
- ✅ You want to see how a headset draws: two eyes, a pose for the head, a frame every 14 ms.
- ❌ You want to publish on the headset's store: this is for your own programs, installed with a
  cable.

## Where to start

Pick the line that sounds like you.

| If you… | Read | Time |
|---|---|---|
| are **new** and want a program in the headset | 🥽 [The quest](2-the-quest.md): a cube in your room, then an island to walk, in 5 levels | 1 hour |
| want to know **what each piece does** and **what you need** | [What is where](3-what-is-where.md) | 10 min |

## All the pages, in reading order

The menu on the left (or the files in `docs/`) follows this order.

**Start**

- [Overview](1-overview.md): this page
- [The quest](2-the-quest.md): five levels, from developer mode to walking an island
- [What is where](3-what-is-where.md): the packages, the tools, what you need, what was checked, and
  what is not done yet

## 🚧 Not done yet

See [what is where](3-what-is-where.md#not-yet).

## Words you will meet

| Word | What it means |
|---|---|
| Go | A programming language. To try these projects you install it once, from go.dev, and type `go run …` in a terminal. |
| Terminal | The window where you type commands: *Terminal* on macOS and Linux, *PowerShell* on Windows. |
| Repository | A project's folder of files, kept on GitHub. `git clone <address>` copies it to your computer. |
| Package (`.apk`) | An Android application in one file. The headset runs Android underneath, so its programs are packages too. |
| adb | Android's tool to talk to a device over a cable: install a package, start it, read its log. |
| OpenXR | The open standard for headsets: it tells a program where the head and the hands are and takes a picture for each eye. |
| Pose | Where something is and which way it faces: a position in metres and a rotation. |
| Quest | The tutorial of these docs (not the headset!): short levels, each ending with something that works and an achievement. |

## The family

Go for Meta Quest is one of a family of small projects that work together:

| Project | What it does |
|---|---|
| [Go Renderer](https://github.com/akrck02/go-renderer) | draws 3D scenes and flat interfaces |
| [Go UI](https://github.com/akrck02/go-ui) | buttons, panels and whole applications |
| [Go UI for Android](https://github.com/akrck02/go-ui-android) | Android apps that look like Android's own |
| [Go Sound](https://github.com/akrck02/go-sound) | sound and music |
| [Go Scene Builder](https://github.com/akrck02/go-scene-builder) | 3D scenes from a flat map |
| [Go Simulator](https://github.com/akrck02/go-simulator) | simulations that change a scene |
| [Go Terrain Simulator](https://github.com/akrck02/go-simulator-terrain) | ground shaped by rivers |
| [Go Climate Simulator](https://github.com/akrck02/go-simulator-climate) | weather and biomes |
| [Go Flora Simulator](https://github.com/akrck02/go-simulator-flora) | forests that grow |
| [Go Urban Simulator](https://github.com/akrck02/go-simulator-urban) | towns that grow |
| [Go Motion Simulator](https://github.com/akrck02/go-simulator-motion) | a scene that moves |
| [Go for 3DS](https://github.com/akrck02/go-3ds) | Go on the Nintendo 3DS |
| **Go for Meta Quest** (this one) | Go on the Meta Quest 3 |

> ⚠️ All of them are **educational experiments**, written with an AI assistant and pending review by their author: good for learning and playing, not for production.
