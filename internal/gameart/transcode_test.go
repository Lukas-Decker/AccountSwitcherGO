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

// A JPEG must never be re-encoded: lossless WebP has to reproduce its
// compression artefacts exactly, which measured 235% larger on real capsules.
func TestTranscode_NeverTouchesJPEG(t *testing.T) {
	t.Parallel()

	src := jpegOf(t, 300, 450)
	out, ext := transcode(src, "jpg")
	if ext != "jpg" {
		t.Errorf("ext = %q, want jpg left alone", ext)
	}
	if !bytes.Equal(out, src) {
		t.Error("jpeg bytes were rewritten")
	}
}

// A PNG with real detail converts, because that is where the saving is.
func TestTranscode_ConvertsPNGWhenSmaller(t *testing.T) {
	t.Parallel()

	src := noisyPNG(t, 300, 450)
	out, ext := transcode(src, "png")
	if ext != "webp" {
		t.Fatalf("ext = %q, want webp", ext)
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
func TestTranscode_KeepsPNGWhenWebPIsBigger(t *testing.T) {
	t.Parallel()

	src := flatPNG(t, 300, 450)
	out, ext := transcode(src, "png")
	if ext == "webp" && len(out) >= len(src) {
		t.Errorf("kept a %d byte webp over a %d byte png", len(out), len(src))
	}
	if ext == "png" && !bytes.Equal(out, src) {
		t.Error("png bytes were rewritten without changing format")
	}
}

// Formats with nothing to gain, or that the encoder cannot read, pass through
// untouched rather than failing the publish.
func TestTranscode_LeavesOtherFormatsAlone(t *testing.T) {
	t.Parallel()

	for _, ext := range []string{"webp", "gif", "ico", "avif", "svg"} {
		src := []byte("not really " + ext)
		out, got := transcode(src, ext)
		if got != ext || !bytes.Equal(out, src) {
			t.Errorf("%s was altered: ext=%q", ext, got)
		}
	}
}

// Corrupt bytes must not lose the candidate: the publish still happens with
// whatever was fetched, and the decode check upstream is what rejects garbage.
func TestTranscode_KeepsUndecodableInput(t *testing.T) {
	t.Parallel()

	src := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0}
	out, ext := transcode(src, "png")
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
			LocalFile(TierLogo, writeFile(t, filepath.Join(dir, "logo.png"), noisyPNG(t, 300, 450))),
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
