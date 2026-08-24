package gameart

import (
	"bytes"
	"image"
	"log/slog"

	"github.com/HugoSmits86/nativewebp"
)

// transcode re-encodes published art as WebP when that is actually smaller.
//
// Baking everything into one format is the appealing version of this, and it is
// wrong. The encoder available here is lossless VP8L, and a lossless container
// has to reproduce a JPEG's compression artefacts exactly, which costs far more
// than the artefacts were ever worth. Measured over 25 real Steam capsules the
// result was 235% larger; over 25 real logo.png files it was 19% smaller.
//
// So the rule is the outcome rather than the format: encode, compare, and keep
// whichever is smaller. That converts the logos and icons, leaves the JPEG
// capsules alone, and stays correct without retuning if a source or an encoder
// changes what the trade looks like.
//
// A lossy encoder would convert everything at a real saving, but every one
// available to Go binds libwebp through cgo, and this project cross-compiles to
// Linux and macOS from Windows.
func transcode(raw []byte, ext string) ([]byte, string) {
	if !worthTranscoding(ext) {
		return raw, ext
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return raw, ext
	}
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		artLog.Debug("webp encode failed, keeping the original", slog.Any("err", err))
		return raw, ext
	}
	if buf.Len() == 0 || buf.Len() >= len(raw) {
		return raw, ext
	}
	return buf.Bytes(), "webp"
}

// worthTranscoding filters to the formats where lossless WebP can win.
//
// A source that is already lossy only grows, and one that is already WebP has
// nothing to gain. bmp is included because it is uncompressed, so anything at
// all beats it.
func worthTranscoding(ext string) bool {
	switch ext {
	case "png", "bmp":
		return true
	default:
		return false
	}
}
