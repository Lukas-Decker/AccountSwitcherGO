// Command genicons rebuilds every shipped icon from the branding master SVG.
//
// The generated files are committed, so this is not part of the build: it is
// run by hand when the artwork changes, and its output is reviewed like any
// other change. Keeping it in the tree is what makes "regenerate from the
// master" in build/branding/README.md an instruction rather than a wish.
//
// Usage, from the repository root:
//
//	go run ./tools/genicons
//	go run ./tools/genicons -check
//
// The -check form regenerates into memory and reports files that have drifted
// from the master without writing anything.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"account-switcher/internal/winutil"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// masterPath is the artwork every other file here is derived from.
const masterPath = "build/branding/AccountSwitcher.svg"

// The fitted crop: a square canvas with the artwork centred on it.
//
// The master's own 120x120 canvas leaves uneven margins (the drawn bounds are
// x 10..100, y 18..100), which at 16 pixels reads as an icon sitting off
// centre. Shifting by these offsets puts the artwork's centre on the canvas
// centre, leaving 5 units of margin left and right and 9 top and bottom.
// build/branding/README.md calls this the fitted form.
//
// The shift is a transform rather than a viewBox offset because oksvg honours a
// viewBox's width and height but ignores its min-x and min-y, so cropping that
// way scales correctly and then draws in the wrong place.
const (
	fittedCanvas  = 100
	fittedOffsetX = -5
	fittedOffsetY = -9
)

// masterDark is the rear figure's colour in the artwork as supplied.
const masterDark = "#201e1d"

// trayDarkModeInk replaces masterDark for the dark-mode tray icon.
//
// The rear figure is very nearly black, which disappears against a dark
// taskbar and leaves the mark reading as one blue shape with a bite out of it.
// This is the same warm neutral inverted, so the two figures keep their
// relationship and only the surface they sit on decides which is drawn light.
const trayDarkModeInk = "#e9e6e4"

// target is one generated file.
type target struct {
	path string
	// sizes has one entry for a PNG and several for an ICO, largest first.
	sizes []int
	ico   bool
	// fitted selects the cropped square box rather than the master's canvas.
	fitted bool
	// recolour swaps colours in the master before rasterising, for the
	// variants that have to survive a surface the artwork was not drawn for.
	recolour map[string]string
}

// targets are every icon this tool owns.
//
// build/windows/icon.ico and build/darwin/icons.icns are deliberately absent:
// the build regenerates both from build/appicon.png with
// "wails3 task common:generate:icons". Writing them here as well would give one
// file two sources, and -check would report drift every time the build ran.
var targets = []target{
	// The input wails3 generates the Windows and macOS icons from.
	{path: "build/appicon.png", sizes: []int{1024}, fitted: true},
	// Embedded into the binary by main.go for the system tray. Two variants:
	// Wails picks the second one when the system is in dark mode.
	{path: "build/trayicon.png", sizes: []int{32}, fitted: true},
	{path: "build/trayicon-darkmode.png", sizes: []int{32}, fitted: true,
		recolour: map[string]string{masterDark: trayDarkModeInk}},
	// The master multi-size icon kept beside the artwork.
	{path: "build/branding/AccountSwitcher.ico", sizes: []int{256, 128, 64, 48, 40, 32, 24, 20, 16}, ico: true, fitted: true},
	// The browser tab icon for the webview.
	{path: "frontend/public/img/favicon.png", sizes: []int{256}, fitted: true},
}

func main() {
	check := flag.Bool("check", false, "report drift from the master without writing")
	flag.Parse()

	master, err := os.ReadFile(masterPath)
	if err != nil {
		fail("read master: %v", err)
	}

	var drifted []string
	for _, t := range targets {
		want, err := render(master, t)
		if err != nil {
			fail("%s: %v", t.path, err)
		}
		if *check {
			got, err := os.ReadFile(t.path)
			if err != nil || !bytes.Equal(got, want) {
				drifted = append(drifted, t.path)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
			fail("%s: %v", t.path, err)
		}
		if err := os.WriteFile(t.path, want, 0o644); err != nil {
			fail("%s: %v", t.path, err)
		}
		fmt.Printf("wrote %s (%v)\n", t.path, t.sizes)
	}

	if *check {
		if len(drifted) == 0 {
			fmt.Println("every icon matches the master")
			return
		}
		fmt.Fprintf(os.Stderr, "stale, rerun go run ./tools/genicons:\n")
		for _, p := range drifted {
			fmt.Fprintf(os.Stderr, "  %s\n", p)
		}
		os.Exit(1)
	}
}

func render(master []byte, t target) ([]byte, error) {
	svg := master
	if t.fitted {
		var err error
		if svg, err = fitted(master); err != nil {
			return nil, err
		}
	}
	for from, to := range t.recolour {
		swapped := bytes.ReplaceAll(svg, []byte(from), []byte(to))
		if bytes.Equal(swapped, svg) {
			return nil, fmt.Errorf("recolour %s: colour not present in the master", from)
		}
		svg = swapped
	}

	pngs := make([][]byte, 0, len(t.sizes))
	for _, size := range t.sizes {
		img, err := rasterize(svg, size)
		if err != nil {
			return nil, fmt.Errorf("rasterize at %d: %w", size, err)
		}
		encoded, err := winutil.EncodePNG(img)
		if err != nil {
			return nil, fmt.Errorf("encode at %d: %w", size, err)
		}
		pngs = append(pngs, encoded)
	}

	if !t.ico {
		return pngs[0], nil
	}
	var buf bytes.Buffer
	if err := winutil.SavePNGsAsICO(&buf, pngs); err != nil {
		return nil, fmt.Errorf("encode ico: %w", err)
	}
	return buf.Bytes(), nil
}

var (
	svgOpenRE  = regexp.MustCompile(`(?is)^.*?<svg[^>]*>`)
	svgCloseRE = regexp.MustCompile(`(?is)</svg\s*>\s*$`)
)

// fitted rewraps the master's contents on a square canvas, centred.
//
// The master is left untouched on disk: it is the artwork as supplied, and the
// framing an icon needs is a property of the icon, not of the drawing.
func fitted(master []byte) ([]byte, error) {
	open := svgOpenRE.FindIndex(master)
	if open == nil {
		return nil, fmt.Errorf("master has no <svg> element")
	}
	closing := svgCloseRE.FindIndex(master)
	if closing == nil {
		return nil, fmt.Errorf("master has no closing </svg>")
	}
	body := bytes.TrimSpace(master[open[1]:closing[0]])
	if len(body) == 0 {
		return nil, fmt.Errorf("master is empty")
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`,
		fittedCanvas, fittedCanvas, fittedCanvas, fittedCanvas)
	fmt.Fprintf(&buf, `<g transform="translate(%d,%d)">`, fittedOffsetX, fittedOffsetY)
	buf.Write(body)
	buf.WriteString(`</g></svg>`)
	return buf.Bytes(), nil
}

// rasterize draws the SVG onto a transparent square.
//
// winutil's rasterizer is not reused because it paints an opaque background for
// the platform logo tiles it was written for. An app icon has to keep its alpha:
// the shell composites it over a taskbar, a desktop, and a title bar, and a
// flattened corner shows up as a grey box on every one of them.
func rasterize(svg []byte, size int) (*image.NRGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.StrictErrorMode)
	if err != nil {
		return nil, err
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		return nil, fmt.Errorf("master has no usable viewBox")
	}
	icon.SetTarget(0, 0, float64(size), float64(size))

	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{}), image.Point{}, draw.Src)

	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(size, size, scanner), 1)

	if err := assertNotBlank(img); err != nil {
		return nil, err
	}
	return img, nil
}

// assertNotBlank catches the failure mode this tool exists to avoid.
//
// A malformed attribute makes a renderer produce a perfectly valid, perfectly
// empty image, and an empty icon looks like a missing file rather than a bug.
// The master arrived with exactly that defect, so it is worth checking.
func assertNotBlank(img *image.NRGBA) error {
	for _, v := range img.Pix {
		if v != 0 {
			return nil
		}
	}
	return fmt.Errorf("rendered nothing: check the master's viewBox and paths")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, strings.TrimRight(format, "\n")+"\n", args...)
	os.Exit(1)
}
