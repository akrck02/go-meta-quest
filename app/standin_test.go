//go:build !android

package app

import (
	"errors"
	"testing"
)

func TestOffTheHeadsetRunNeedsAStandIn(test *testing.T) {
	standIn = nil
	if err := Run(Options{}, func(Frame) bool { return false }); !errors.Is(err, ErrNoHeadset) {
		test.Fatalf("Run with no stand-in gave %v", err)
	}
}

func TestOffTheHeadsetRunHandsTheProgramToTheStandIn(test *testing.T) {
	defer func() { standIn = nil }()
	frames := 0
	UseStandIn(func(options Options, step func(Frame) bool) error {
		for step(Frame{}) {
		}
		if options.ApplicationName != "Program" {
			test.Fatalf("the stand-in got %+v", options)
		}
		return nil
	})
	err := Run(Options{ApplicationName: "Program"}, func(Frame) bool {
		frames++
		return frames < 3
	})
	if err != nil || frames != 3 {
		test.Fatalf("ran %d frames, error %v", frames, err)
	}
	if err := Run(Options{Panels: []PanelSize{{}}}, func(Frame) bool { return false }); err == nil {
		test.Fatal("a panel with no size reached the stand-in")
	}
}
