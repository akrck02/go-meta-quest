package app

// longestStep is the most a frame's Seconds can be. After a pause (the headset taken off, the
// system menu) the next frame's display time can be minutes later; a program that moves things by
// Seconds would make them jump all that way at once.
const longestStep = 0.1

// clock turns the display times OpenXR predicts for each frame into the seconds between frames.
// Display times, not the wall clock: they are when the picture will be seen, so motion keeps step
// with what the eyes see even when a frame is late.
type clock struct {
	previous int64 // the previous frame's display time, in nanoseconds
	started  bool
}

// advance takes a frame's display time, in nanoseconds, and returns the seconds since the previous
// frame's: 0 for the first frame, and never less than 0 or more than longestStep.
func (clock *clock) advance(displayTime int64) float64 {
	if !clock.started {
		clock.previous, clock.started = displayTime, true
		return 0
	}
	seconds := float64(displayTime-clock.previous) / 1e9
	clock.previous = displayTime
	if seconds < 0 {
		return 0
	}
	if seconds > longestStep {
		return longestStep
	}
	return seconds
}

// restart forgets the previous frame, so the frame after a pause counts from itself.
func (clock *clock) restart() {
	clock.started = false
}
