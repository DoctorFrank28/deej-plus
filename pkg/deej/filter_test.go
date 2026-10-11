package deej

import (
	"math"
	"testing"
)

func feed(f *sliderFilter, values ...float32) (emitted []float32) {
	for _, v := range values {
		if value, changed := f.update(v); changed {
			emitted = append(emitted, value)
		}
	}

	return emitted
}

func TestFilterIgnoresJitterBetweenSteps(t *testing.T) {
	f := newSliderFilter(noiseReductionPresets[noiseReductionDefault])

	// a slider resting right between 47% and 48%, with the ±1 count noise measured on the real board
	got := feed(f, 0.4750, 0.4755, 0.4745, 0.4760, 0.4750, 0.4740, 0.4755)
	if len(got) != 1 {
		t.Fatalf("expected only the initial value, got %v", got)
	}
}

func TestFilterFollowsRealMovementInOnePercentSteps(t *testing.T) {
	f := newSliderFilter(noiseReductionPresets[noiseReductionDefault])

	values := []float32{}
	for v := 0.40; v <= 0.60001; v += 0.002 {
		values = append(values, float32(v))
	}

	got := feed(f, values...)
	if len(got) < 15 {
		t.Fatalf("expected the volume to follow a 20%% move closely, got %d steps: %v", len(got), got)
	}

	last := got[len(got)-1]
	if math.Abs(float64(last)-0.60) > 0.0101 {
		t.Fatalf("expected to end near 60%%, got %v", last)
	}

	for i := 1; i < len(got); i++ {
		if step := math.Abs(float64(got[i] - got[i-1])); step > 0.0201 {
			t.Fatalf("expected small steps, got a jump of %.3f in %v", step, got)
		}
	}
}

func TestFilterSnapsToEdges(t *testing.T) {
	f := newSliderFilter(noiseReductionPresets[noiseReductionDefault])

	got := feed(f, 0.5, 0.2, 0.05, 0.012, 0.006)
	if got[len(got)-1] != 0 {
		t.Fatalf("expected to snap to 0, got %v", got)
	}

	got = feed(f, 0.5, 0.9, 0.985, 0.994)
	if got[len(got)-1] != 1 {
		t.Fatalf("expected to snap to 1, got %v", got)
	}
}

func TestFilterSmoothingConvergesOnHeartbeats(t *testing.T) {
	f := newSliderFilter(noiseReductionPresets[noiseReductionHigh])

	feed(f, 0.30)

	// the board stops at 0.50 and then only repeats it once a second
	values := []float32{}
	for i := 0; i < 12; i++ {
		values = append(values, 0.50)
	}

	got := feed(f, values...)
	if len(got) == 0 || math.Abs(float64(got[len(got)-1])-0.50) > 0.0101 {
		t.Fatalf("expected to settle at 50%%, got %v", got)
	}
}
