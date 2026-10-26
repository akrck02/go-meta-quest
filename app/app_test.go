package app

import (
	"testing"
	"time"

	"github.com/akrck02/go-renderer/openxr"
)

func TestLeavingWinsOverRunning(test *testing.T) {
	if got := nextMove(true, true); got != moveStop {
		test.Fatalf("exiting while running: got %v, want moveStop", got)
	}
	if got := nextMove(true, false); got != moveStop {
		test.Fatalf("exiting while stopped: got %v, want moveStop", got)
	}
}

func TestASessionThatDoesNotRunIsWaitedFor(test *testing.T) {
	if got := nextMove(false, false); got != moveWait {
		test.Fatalf("got %v, want moveWait", got)
	}
}

func TestARunningSessionMakesFrames(test *testing.T) {
	if got := nextMove(false, true); got != moveFrame {
		test.Fatalf("got %v, want moveFrame", got)
	}
}

func TestTheFirstFrameTakesNoTime(test *testing.T) {
	var frames clock
	if seconds := frames.advance(5_000_000_000); seconds != 0 {
		test.Fatalf("first frame: got %v seconds, want 0", seconds)
	}
}

func TestFramesAreTheDisplayTimesApart(test *testing.T) {
	var frames clock
	frames.advance(1_000_000_000)
	// 72 frames a second: about 13.9 ms between pictures.
	seconds := frames.advance(1_013_888_889)
	if seconds < 0.0138 || seconds > 0.0139 {
		test.Fatalf("got %v seconds, want about 0.0139", seconds)
	}
}

func TestALongPauseCountsAsOneShortStep(test *testing.T) {
	var frames clock
	frames.advance(1_000_000_000)
	if seconds := frames.advance(301_000_000_000); seconds != longestStep {
		test.Fatalf("after five minutes away: got %v seconds, want %v", seconds, longestStep)
	}
}

func TestTimeNeverRunsBackwards(test *testing.T) {
	var frames clock
	frames.advance(2_000_000_000)
	if seconds := frames.advance(1_000_000_000); seconds != 0 {
		test.Fatalf("an earlier display time: got %v seconds, want 0", seconds)
	}
}

func TestARestartedClockStartsAgain(test *testing.T) {
	var frames clock
	frames.advance(1_000_000_000)
	frames.restart()
	if seconds := frames.advance(1_050_000_000); seconds != 0 {
		test.Fatalf("first frame after a restart: got %v seconds, want 0", seconds)
	}
}

func TestLeavingWaitsForTheRuntimeWithEmptyFrames(test *testing.T) {
	if got := leavingMove(false, 100*time.Millisecond); got != moveEmptyFrame {
		test.Fatalf("got %v, want moveEmptyFrame", got)
	}
}

func TestLeavingStopsOnceTheSessionIsFinished(test *testing.T) {
	if got := leavingMove(true, 0); got != moveStop {
		test.Fatalf("got %v, want moveStop", got)
	}
}

func TestLeavingNeverWaitsForeverForTheRuntime(test *testing.T) {
	if got := leavingMove(false, exitPatience); got != moveStop {
		test.Fatalf("after %v: got %v, want moveStop", exitPatience, got)
	}
}

func TestThePanelsAskedForAreCheckedBeforeAnyIsMade(test *testing.T) {
	if err := CheckPanels([]PanelSize{{1024, 768}, {512, 512}}); err != nil {
		test.Fatalf("two sized panels were refused: %v", err)
	}
	if err := CheckPanels([]PanelSize{{1024, 0}}); err == nil {
		test.Fatal("a panel with no height was taken")
	}
	if err := CheckPanels(make([]PanelSize, openxr.MostPanels+1)); err == nil {
		test.Fatal("more panels than a session holds were taken")
	}
}
