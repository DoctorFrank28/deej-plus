package deej

import "math"

// filterParams tune how slider jitter is told apart from real movement. All values are fractions of the full range
type filterParams struct {

	// extra distance (on top of half a 1% step) the input must travel away from the current volume
	// before the volume follows. this hysteresis keeps a slider resting between two steps from flickering
	Deadband float64 `json:"deadband"`

	// weight of the previous smoothed value in an exponential moving average (0 = no smoothing)
	Smoothing float64 `json:"smoothing"`

	// inputs this close to either end go straight to 0% or 100%
	EdgeSnap float64 `json:"edgeSnap"`
}

const volumeStep = 0.01

var noiseReductionPresets = map[string]filterParams{
	noiseReductionLow:     {Deadband: 0.002, Smoothing: 0, EdgeSnap: 0.005},
	noiseReductionDefault: {Deadband: 0.004, Smoothing: 0, EdgeSnap: 0.01},
	noiseReductionHigh:    {Deadband: 0.01, Smoothing: 0.5, EdgeSnap: 0.02},
}

const (
	noiseReductionLow     = "low"
	noiseReductionDefault = "default"
	noiseReductionHigh    = "high"
	noiseReductionCustom  = "custom"
)

// sliderFilter turns a stream of noisy 0..1 readings into 1% volume steps
type sliderFilter struct {
	params filterParams

	initialized bool
	smoothed    float64
	emitted     float64
}

func newSliderFilter(params filterParams) *sliderFilter {
	return &sliderFilter{params: params}
}

// update feeds a new reading and returns the volume to apply, plus whether it changed
func (f *sliderFilter) update(raw float32) (float32, bool) {
	x := float64(raw)

	if !f.initialized {
		f.initialized = true
		f.smoothed = x
		f.emitted = f.quantize(x)

		return float32(f.emitted), true
	}

	smoothing := math.Max(0, math.Min(f.params.Smoothing, 0.95))
	f.smoothed = f.smoothed*smoothing + x*(1-smoothing)

	target := f.quantize(f.smoothed)
	if target == f.emitted {
		return float32(f.emitted), false
	}

	// the ends always pass, so a slider pushed all the way really mutes or maxes out
	isEdge := target == 0 || target == 1

	if !isEdge && math.Abs(f.smoothed-f.emitted) < volumeStep/2+f.params.Deadband {
		return float32(f.emitted), false
	}

	f.emitted = target

	return float32(f.emitted), true
}

func (f *sliderFilter) quantize(x float64) float64 {
	if x <= f.params.EdgeSnap {
		return 0
	}

	if x >= 1-f.params.EdgeSnap {
		return 1
	}

	return math.Round(x/volumeStep) * volumeStep
}
