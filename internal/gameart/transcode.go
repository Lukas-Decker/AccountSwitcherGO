package gameart

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log/slog"

	"github.com/HugoSmits86/nativewebp"
	"github.com/nfnt/resize"
)

// Published art is re-encoded to the size and quality the grid actually draws.
//
// Sources ship store assets, not thumbnails: a Steam capsule arrives at 600x900
// and a hero at 1920 wide, while the tile is about 115x172 CSS pixels. Keeping
// the originals meant a library of four thousand games cost hundreds of
// megabytes to show images nothing ever displayed at full size.
//
// Measured over a sample of a real cache, downscaling to a 600 pixel long edge
// and encoding opaque art as JPEG at quality 82 gives 84% smaller files. The
// same images as lossless WebP came out 1% smaller than the source and as PNG
// 74% larger, so neither is worth having for the common case.
const (
	// maxArtEdge bounds the longest side. The tile is drawn at roughly 172
	// pixels tall, so this still has headroom at three times that density, and
	// nothing on screen is ever scaled up.
	maxArtEdge = 600

	// jpegQuality is where the size curve flattens without visible loss at the
	// size these are drawn. Quality 88 was only 20% larger for no difference a
	// tile shows.
	jpegQuality = 82
)

// normalize re-encodes published art, returning the bytes and extension to
// store.
//
// The original is kept whenever re-encoding does not actually help, so a source
// that already ships a small optimised image is left alone rather than being
// decoded and rebuilt for nothing.
func normalize(raw []byte, ext string) ([]byte, string) {
	if !worthNormalizing(ext) {
		return raw, ext
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		// Undecodable here means a format Go cannot read, such as an icon
		// pulled from an executable. It is published as it arrived.
		return raw, ext
	}

	small := fitWithin(img, maxArtEdge)

	// Transparency is the whole point of a wordmark, so those keep an alpha
	// channel and take the lossless encoder even though it saves less.
	if hasAlpha(small) {
		var buf bytes.Buffer
		if err := nativewebp.Encode(&buf, small, nil); err != nil {
			artLog.Debug("webp encode failed, keeping the original", slog.Any("err", err))
			return raw, ext
		}
		if buf.Len() > 0 && buf.Len() < len(raw) {
			return buf.Bytes(), "webp"
		}
		return raw, ext
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flatten(small), &jpeg.Options{Quality: jpegQuality}); err != nil {
		artLog.Debug("jpeg encode failed, keeping the original", slog.Any("err", err))
		return raw, ext
	}
	if buf.Len() == 0 || buf.Len() >= len(raw) {
		return raw, ext
	}
	return buf.Bytes(), "jpg"
}

// worthNormalizing filters to the formats Go can decode and re-encode.
//
// An icon extracted from an executable is left alone: it is already small, and
// it is the one shape where the square original is the right thing to keep.
func worthNormalizing(ext string) bool {
	switch ext {
	case "jpg", "png", "webp", "gif", "bmp":
		return true
	default:
		return false
	}
}

// fitWithin scales an image down so neither side exceeds maxEdge. It never
// scales up: enlarging a small source only costs bytes.
func fitWithin(img image.Image, maxEdge uint) image.Image {
	b := img.Bounds()
	w, h := uint(b.Dx()), uint(b.Dy())
	if w <= maxEdge && h <= maxEdge {
		return img
	}
	if w >= h {
		return resize.Resize(maxEdge, 0, img, resize.Lanczos3)
	}
	return resize.Resize(0, maxEdge, img, resize.Lanczos3)
}

// hasAlpha reports whether the image has any transparency.
//
// Sampled rather than exhaustive: every other pixel is enough to find a
// transparent background, and a full scan of a 600x900 image for every game in
// a library is work with nothing to show for it.
func hasAlpha(img image.Image) bool {
	if _, ok := img.(*image.YCbCr); ok {
		// A decoded JPEG has no alpha channel at all.
		return false
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x += 2 {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}

// flatten composites onto black, which is what the grid draws tiles on, so a
// source that turns out to have a stray transparent edge does not gain a white
// border on the way into a format with no alpha.
func flatten(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, image.NewUniform(color.Black), image.Point{}, draw.Src)
	draw.Draw(dst, b, img, b.Min, draw.Over)
	return dst
}
