// genicons draws deej's icons (app/tray logo and tray menu glyphs) and writes them as .ico files.
//
// The look follows the UI's design system (shadcn/ui, neutral theme): a near-black rounded square
// with three light faders, and 1.5px-stroke line glyphs for menu items. Every size is drawn on its
// own, snapped to the pixel grid, so the 16px tray icon stays crisp.
//
// Usage (from the repo root):
//
//	go run ./tools/genicons
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

const outputDir = "pkg/deej/icon"

var (
	neutral950 = color.NRGBA{0x17, 0x17, 0x17, 0xff} // shadcn primary (neutral-900)
	neutral50  = color.NRGBA{0xfa, 0xfa, 0xfa, 0xff} // shadcn primary-foreground
	neutral600 = color.NRGBA{0x52, 0x52, 0x52, 0xff} // reads on light and dark menus
)

func main() {
	write("logo.ico", []int{16, 20, 24, 32, 40, 48, 64, 128, 256}, drawLogo)
	write("edit-config.ico", []int{16, 20, 24, 32}, drawSettingsGlyph)
	write("refresh.ico", []int{16, 20, 24, 32}, drawRefreshGlyph)

	if len(os.Args) > 1 {
		preview(os.Args[1])
	}
}

// preview writes PNGs of every icon at a few sizes, to look at them while tweaking
func preview(dir string) {
	glyphs := map[string]func(int) []layer{"logo": drawLogo, "edit-config": drawSettingsGlyph, "refresh": drawRefreshGlyph}

	for name, draw := range glyphs {
		for _, size := range []int{16, 32, 256} {
			var buf bytes.Buffer
			if err := png.Encode(&buf, render(size, draw(size))); err != nil {
				log.Fatal(err)
			}

			os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s-%d.png", name, size)), buf.Bytes(), 0o644)
		}
	}
}

// shape is a signed distance function in pixel coordinates (negative inside)
type shape func(x, y float64) float64

type layer struct {
	shape shape
	color color.NRGBA
}

func write(name string, sizes []int, draw func(size int) []layer) {
	images := make([]*image.NRGBA, 0, len(sizes))
	for _, size := range sizes {
		images = append(images, render(size, draw(size)))
	}

	path := filepath.Join(outputDir, name)
	if err := os.WriteFile(path, encodeICO(images), 0o644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}

	log.Printf("wrote %s (%v)", path, sizes)
}

// render rasterizes the layers back to front with 8x8 supersampling per pixel
func render(size int, layers []layer) *image.NRGBA {
	const samples = 8

	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var r, g, b, a float64

			for _, l := range layers {
				covered := 0
				for sy := 0; sy < samples; sy++ {
					for sx := 0; sx < samples; sx++ {
						x := float64(px) + (float64(sx)+0.5)/samples
						y := float64(py) + (float64(sy)+0.5)/samples
						if l.shape(x, y) <= 0 {
							covered++
						}
					}
				}

				coverage := float64(covered) / (samples * samples) * float64(l.color.A) / 255
				if coverage == 0 {
					continue
				}

				// "over" compositing in premultiplied space
				r = float64(l.color.R)*coverage + r*(1-coverage)
				g = float64(l.color.G)*coverage + g*(1-coverage)
				b = float64(l.color.B)*coverage + b*(1-coverage)
				a = coverage + a*(1-coverage)
			}

			if a > 0 {
				img.SetNRGBA(px, py, color.NRGBA{
					R: uint8(math.Round(r / a)), G: uint8(math.Round(g / a)), B: uint8(math.Round(b / a)),
					A: uint8(math.Round(a * 255)),
				})
			}
		}
	}

	return img
}

// drawLogo: rounded square with three faders at different heights, like the mixer screen
func drawLogo(size int) []layer {
	s := float64(size)
	u := s / 24 // the design grid is 24 units, like lucide icons

	layers := []layer{{roundedRect(0, 0, s, s, math.Max(3, 5.5*u)), neutral950}}

	trackWidth := math.Max(1, math.Round(1.5*u))
	knobWidth := oddAtLeast(math.Round(4.6*u), 3)
	knobHeight := math.Max(2, math.Round(2.5*u))
	top, bottom := math.Round(5*u), s-math.Round(5*u)

	track := color.NRGBA{neutral50.R, neutral50.G, neutral50.B, 0x66}

	// knob centers, as a fraction of the track from the top
	for i, level := range []float64{0.62, 0.22, 0.44} {
		cx := snapCenter((6.5+5.5*float64(i))*u, trackWidth)
		layers = append(layers, layer{roundedRect(cx-trackWidth/2, top, cx+trackWidth/2, bottom, trackWidth/2), track})

		cy := math.Round(top + level*(bottom-top))
		layers = append(layers, layer{
			roundedRect(cx-knobWidth/2, cy-knobHeight/2, cx+knobWidth/2, cy+knobHeight/2, math.Min(knobHeight/2, 1.2*u)),
			neutral50,
		})
	}

	return layers
}

// drawSettingsGlyph: lucide-style "sliders-horizontal", the config is where the mixer is set up
func drawSettingsGlyph(size int) []layer {
	s := float64(size)
	u := s / 24
	w := math.Max(1, math.Round(2*u))

	layers := []layer{}
	for i, knob := range []float64{15, 8, 13} {
		y := snapCenter((6+6*float64(i))*u, w)
		kx := math.Round(knob * u)
		layers = append(layers,
			layer{capsule(3*u, y, kx-2.5*u, y, w/2), neutral600},
			layer{capsule(kx+2.5*u, y, 21*u, y, w/2), neutral600},
			layer{ring(kx, y, 2.2*u, w/2), neutral600},
		)
	}

	return layers
}

// drawRefreshGlyph: lucide-style "rotate-cw", an almost full circle with an arrow head at its end
func drawRefreshGlyph(size int) []layer {
	s := float64(size)
	u := s / 24
	w := math.Max(1, math.Round(2*u))
	cx, cy, r := s/2+0.5*u, s/2+0.5*u, 7*u

	// clockwise from the top, leaving a gap before coming back to the top
	const startDeg, sweepDeg = -90.0, 290.0
	end := (startDeg + sweepDeg) * math.Pi / 180
	ax, ay := cx+r*math.Cos(end), cy+r*math.Sin(end)

	// the arrow head points along the direction of travel (the clockwise tangent)
	tx, ty := -math.Sin(end), math.Cos(end)
	wing := func(angle float64) (float64, float64) {
		dx := tx*math.Cos(angle) - ty*math.Sin(angle)
		dy := tx*math.Sin(angle) + ty*math.Cos(angle)
		return ax - 3.5*u*dx, ay - 3.5*u*dy
	}
	w1x, w1y := wing(0.8)
	w2x, w2y := wing(-0.8)

	return []layer{
		{arc(cx, cy, r, w/2, startDeg, sweepDeg), neutral600},
		{capsule(ax, ay, w1x, w1y, w/2), neutral600},
		{capsule(ax, ay, w2x, w2y, w/2), neutral600},
	}
}

func roundedRect(x0, y0, x1, y1, radius float64) shape {
	return func(x, y float64) float64 {
		cx, cy := (x0+x1)/2, (y0+y1)/2
		hx, hy := (x1-x0)/2-radius, (y1-y0)/2-radius
		dx := math.Max(math.Abs(x-cx)-hx, 0)
		dy := math.Max(math.Abs(y-cy)-hy, 0)
		inside := math.Min(math.Max(math.Abs(x-cx)-hx, math.Abs(y-cy)-hy), 0)

		return math.Hypot(dx, dy) + inside - radius
	}
}

func capsule(x0, y0, x1, y1, radius float64) shape {
	return func(x, y float64) float64 {
		vx, vy := x1-x0, y1-y0
		t := 0.0
		if l := vx*vx + vy*vy; l > 0 {
			t = math.Max(0, math.Min(1, ((x-x0)*vx+(y-y0)*vy)/l))
		}

		return math.Hypot(x-(x0+t*vx), y-(y0+t*vy)) - radius
	}
}

func ring(cx, cy, r, halfWidth float64) shape {
	return func(x, y float64) float64 {
		return math.Abs(math.Hypot(x-cx, y-cy)-r) - halfWidth
	}
}

// arc is a stroked circle arc from startDeg sweeping sweepDeg clockwise (screen coordinates), round caps
func arc(cx, cy, r, halfWidth, startDeg, sweepDeg float64) shape {
	start := startDeg * math.Pi / 180
	sweep := sweepDeg * math.Pi / 180

	return func(x, y float64) float64 {
		angle := math.Atan2(y-cy, x-cx) - start
		for angle < 0 {
			angle += 2 * math.Pi
		}

		if angle <= sweep {
			return math.Abs(math.Hypot(x-cx, y-cy)-r) - halfWidth
		}

		// outside the sweep: distance to the nearest end cap
		end := start + sweep
		d0 := math.Hypot(x-(cx+r*math.Cos(start)), y-(cy+r*math.Sin(start)))
		d1 := math.Hypot(x-(cx+r*math.Cos(end)), y-(cy+r*math.Sin(end)))

		return math.Min(d0, d1) - halfWidth
	}
}

// snapCenter puts a stroke's center where the stroke covers whole pixels
func snapCenter(v, width float64) float64 {
	if int(width)%2 == 1 {
		return math.Floor(v) + 0.5
	}

	return math.Round(v)
}

func oddAtLeast(v, min float64) float64 {
	v = math.Max(v, min)
	if int(v)%2 == 0 {
		v++
	}

	return v
}

// encodeICO packs PNG-compressed images into an .ico container (supported since Windows Vista)
func encodeICO(images []*image.NRGBA) []byte {
	pngs := make([][]byte, len(images))
	for i, img := range images {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			log.Fatal(err)
		}
		pngs[i] = buf.Bytes()
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(images))})

	offset := 6 + 16*len(images)
	for i, img := range images {
		size := img.Bounds().Dx()
		dim := uint8(size)
		if size >= 256 {
			dim = 0
		}

		out.Write([]byte{dim, dim, 0, 0})
		binary.Write(&out, binary.LittleEndian, uint16(1))  // color planes
		binary.Write(&out, binary.LittleEndian, uint16(32)) // bits per pixel
		binary.Write(&out, binary.LittleEndian, uint32(len(pngs[i])))
		binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(pngs[i])
	}

	for _, data := range pngs {
		out.Write(data)
	}

	return out.Bytes()
}
