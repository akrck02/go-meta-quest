//go:build !android

package app

import "errors"

// Off the headset there is no session to open: Run hands the program to a stand-in for the headset
// on the computer, one that hands out the same Frame (a window whose mouse, keys or game pads are
// the head and the controllers). The stand-in registers itself with UseStandIn, from a package the
// program imports, so the program itself stays as it is on the headset.

// ErrNoHeadset is what Run gives off the headset when no stand-in was registered.
var ErrNoHeadset = errors.New("app: no headset here: run the program on a Meta Quest, or in a stand-in for one (app.UseStandIn)")

// standIn is the registered stand-in, if any.
var standIn func(options Options, step func(frame Frame) bool) error

// UseStandIn makes Run, off the headset, run programs with run: a stand-in for the headset that
// calls step once a frame with the same Frame, until step returns false or it is closed.
func UseStandIn(run func(options Options, step func(frame Frame) bool) error) { standIn = run }

// Run, off the headset, runs the program in the stand-in registered with UseStandIn, or fails with
// ErrNoHeadset when there is none. It must be called from main.
func Run(options Options, step func(frame Frame) bool) error {
	if standIn == nil {
		return ErrNoHeadset
	}
	if err := CheckPanels(options.Panels); err != nil {
		return err
	}
	return standIn(options, step)
}
