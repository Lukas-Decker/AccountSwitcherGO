package gameart

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// flatPNG compresses to almost nothing, so lossless WebP has no room to beat
// it. noisyPNG is the opposite: real artwork with detail, which is where the
// saving actually comes from.
func flatPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 30, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func noisyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	rng := rand.New(rand.NewSource(1))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Blocky noise rather than per-pixel noise: per-pixel is incompressible by
	// anything, which would prove nothing about the encoder.
	for y := 0; y < h; y += 4 {
		for x := 0; x < w; x += 4 {
			c := color.RGBA{
				R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)),
				B: uint8(rng.Intn(256)), A: 255,
			}
			for dy := 0; dy < 4 && y+dy < h; dy++ {
				for dx := 0; dx < 4 && x+dx < w; dx++ {
					img.Set(x+dx, y+dy, c)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// transparentPNG has a real alpha channel, which is what a wordmark looks like.
func transparentPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	rng := rand.New(rand.NewSource(3))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 4 {
		for x := 0; x < w; x += 4 {
			c := color.RGBA{R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: uint8(rng.Intn(256)), A: 255}
			// A transparent margin, as a logo has.
			if x < w/6 || x > w-w/6 {
				c.A = 0
			}
			for dy := 0; dy < 4 && y+dy < h; dy++ {
				for dx := 0; dx < 4 && x+dx < w; dx++ {
					img.Set(x+dx, y+dy, c)
				}
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// An oversized capsule is the common case and the whole reason this exists:
// the tile is drawn at about 172 pixels tall and the source ships at 900.
func TestNormalize_DownscalesOversizedArt(t *testing.T) {
	t.Parallel()

	src := noisyPNG(t, 1200, 1800)
	out, ext := normalize(src, "png")
	if len(out) >= len(src) {
		t.Fatalf("output %d bytes against a %d byte source, want smaller", len(out), len(src))
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output does not decode: %v", err)
	}
	if cfg.Width > maxArtEdge || cfg.Height > maxArtEdge {
		t.Errorf("output is %dx%d, want nothing over %d", cfg.Width, cfg.Height, maxArtEdge)
	}
	if ext != "jpg" {
		t.Errorf("ext = %q, want an opaque image encoded as jpeg", ext)
	}
}

// Nothing is ever scaled up: enlarging a small source only costs bytes.
func TestNormalize_NeverEnlarges(t *testing.T) {
	t.Parallel()

	src := noisyPNG(t, 120, 180)
	out, _ := normalize(src, "png")
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 120 || cfg.Height != 180 {
		t.Errorf("output is %dx%d, want the original 120x180", cfg.Width, cfg.Height)
	}
}

// Transparency is the point of a wordmark, so those keep an alpha channel and
// take the lossless encoder even though it saves less than jpeg would.
func TestNormalize_KeepsTransparencyAsWebP(t *testing.T) {
	t.Parallel()

	src := transparentPNG(t, 300, 450)
	out, ext := normalize(src, "png")
	if ext != "webp" {
		t.Fatalf("ext = %q, want webp for an image with alpha", ext)
	}
	if len(out) >= len(src) {
		t.Errorf("webp is %d bytes against a %d byte png, which is not a saving", len(out), len(src))
	}
	// It still has to be an image, not just smaller.
	if got, ok := extFromMagic(out); !ok || got != "webp" {
		t.Errorf("output magic = %q,%v want webp", got, ok)
	}
	if _, _, err := image.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("published webp does not decode: %v", err)
	}
}

// The rule is the outcome, not the format: a PNG that WebP cannot beat is kept
// as it was.
func TestNormalize_KeepsPNGWhenWebPIsBigger(t *testing.T) {
	t.Parallel()

	src := flatPNG(t, 300, 450)
	out, ext := normalize(src, "png")
	if ext == "webp" && len(out) >= len(src) {
		t.Errorf("kept a %d byte webp over a %d byte png", len(out), len(src))
	}
	if ext == "png" && !bytes.Equal(out, src) {
		t.Error("png bytes were rewritten without changing format")
	}
}

// Formats with nothing to gain, or that the encoder cannot read, pass through
// untouched rather than failing the publish.
func TestNormalize_LeavesOtherFormatsAlone(t *testing.T) {
	t.Parallel()

	for _, ext := range []string{"webp", "gif", "ico", "avif", "svg"} {
		src := []byte("not really " + ext)
		out, got := normalize(src, ext)
		if got != ext || !bytes.Equal(out, src) {
			t.Errorf("%s was altered: ext=%q", ext, got)
		}
	}
}

// Corrupt bytes must not lose the candidate: the publish still happens with
// whatever was fetched, and the decode check upstream is what rejects garbage.
func TestNormalize_KeepsUndecodableInput(t *testing.T) {
	t.Parallel()

	src := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0}
	out, ext := normalize(src, "png")
	if ext != "png" || !bytes.Equal(out, src) {
		t.Errorf("undecodable input was altered: ext=%q", ext)
	}
}

// End to end: what lands in wwwroot is the converted file, and the URL the view
// loads points at it.
func TestResolve_PublishesTranscodedArt(t *testing.T) {
	wwwroot := setupWwwroot(t)
	dir := t.TempDir()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "logo-game",
		Candidates: []Candidate{
			LocalFile(TierLogo, writeFile(t, filepath.Join(dir, "logo.png"), transparentPNG(t, 300, 450))),
		},
	}
	res := Resolve(context.Background(), http.DefaultClient, req, false)
	if !strings.HasSuffix(res.PublicURL, ".webp") {
		t.Fatalf("publicURL = %q, want a .webp", res.PublicURL)
	}
	got := publishedFiles(t, wwwroot, "Steam")
	if len(got) != 1 || !strings.HasSuffix(got[0], ".webp") {
		t.Fatalf("published %v, want one webp", got)
	}
	// And a second pass must reuse it rather than re-encoding.
	if again := Resolve(context.Background(), http.DefaultClient, req, false); again.PublicURL != res.PublicURL {
		t.Errorf("second pass returned %q, want the cached %q", again.PublicURL, res.PublicURL)
	}
}
