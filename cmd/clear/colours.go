package main

import "math"

// cycleSeconds is how long an eye's colour takes to go round all the hues: slow enough to watch
// it change, fast enough to see that frames are coming.
const cycleSeconds = 20

// eyeColour is an eye's colour at a moment: a bright hue going round the colour wheel, the right
// eye half a turn behind the left, so each eye clearly sees its own. The colour is red, green and
// blue from 0 to 1.
func eyeColour(eye int, seconds float64) (red, green, blue float32) {
	turn := seconds/cycleSeconds + float64(eye)*0.5
	return hue(turn - math.Floor(turn))
}

// hue is a fully saturated colour at a share of the way round the colour wheel (0 red, a third
// green, two thirds blue), a little darker than full so it is comfortable to look at.
func hue(share float64) (red, green, blue float32) {
	const brightness = 0.8
	channel := func(offset float64) float32 {
		// a triangle wave: 1 at its own third of the wheel, falling to 0 a third away either side
		distance := math.Abs(math.Mod(share-offset+1.5, 1) - 0.5)
		return float32(brightness * math.Max(0, math.Min(1, 2-6*distance)))
	}
	return channel(0), channel(1.0 / 3), channel(2.0 / 3)
}
