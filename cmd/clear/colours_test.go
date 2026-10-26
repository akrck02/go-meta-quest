package main

import (
	"math"
	"testing"
)

func near(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-4 }

func TestTheWheelStartsRedThenGreenThenBlue(test *testing.T) {
	cases := []struct {
		share            float64
		red, green, blue float32
	}{
		{0, 0.8, 0, 0},
		{1.0 / 3, 0, 0.8, 0},
		{2.0 / 3, 0, 0, 0.8},
		{1.0 / 6, 0.8, 0.8, 0}, // yellow, between red and green
	}
	for _, each := range cases {
		red, green, blue := hue(each.share)
		if !near(red, each.red) || !near(green, each.green) || !near(blue, each.blue) {
			test.Errorf("hue(%v) = %v %v %v, want %v %v %v", each.share, red, green, blue, each.red, each.green, each.blue)
		}
	}
}

func TestTheEyesAreHalfAWheelApart(test *testing.T) {
	leftRed, leftGreen, leftBlue := eyeColour(0, 0)
	rightRed, rightGreen, rightBlue := eyeColour(1, 0)
	if !near(leftRed, 0.8) || !near(leftGreen, 0) || !near(leftBlue, 0) {
		test.Fatalf("left eye at the start: got %v %v %v, want red", leftRed, leftGreen, leftBlue)
	}
	// half a turn from red is cyan
	if !near(rightRed, 0) || !near(rightGreen, 0.8) || !near(rightBlue, 0.8) {
		test.Fatalf("right eye at the start: got %v %v %v, want cyan", rightRed, rightGreen, rightBlue)
	}
}

func TestAColourComesBackAfterACycle(test *testing.T) {
	firstRed, firstGreen, firstBlue := eyeColour(0, 3)
	laterRed, laterGreen, laterBlue := eyeColour(0, 3+cycleSeconds)
	if !near(firstRed, laterRed) || !near(firstGreen, laterGreen) || !near(firstBlue, laterBlue) {
		test.Fatalf("got %v %v %v then %v %v %v", firstRed, firstGreen, firstBlue, laterRed, laterGreen, laterBlue)
	}
}
