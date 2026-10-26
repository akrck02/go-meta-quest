package app

import (
	"testing"
	"time"
)

// shaker remembers what it was asked to shake.
type shaker struct {
	hand      int
	amplitude float32
	duration  time.Duration
}

func (made *shaker) Vibrate(hand int, amplitude float32, duration time.Duration) {
	made.hand, made.amplitude, made.duration = hand, amplitude, duration
}

func TestAFrameShakesThroughItsHapticsAndNothingWithout(test *testing.T) {
	Frame{}.Vibrate(RightHand, 1, time.Second) // nothing to shake: nothing happens
	shaken := &shaker{}
	Frame{Haptics: shaken}.Vibrate(RightHand, 0.5, 40*time.Millisecond)
	if *shaken != (shaker{RightHand, 0.5, 40 * time.Millisecond}) {
		test.Fatalf("the haptics were asked %+v, want the right hand at 0.5 for 40 ms", *shaken)
	}
}
