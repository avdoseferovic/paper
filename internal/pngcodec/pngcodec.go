// Package pngcodec encodes PNG images with pooled encoder state.
package pngcodec

import (
	"image"
	"image/png"
	"io"
	"sync"
)

// encoderBufferPool reuses png encoder state across Encode calls. Every
// png.Encoder.Encode otherwise builds a fresh zlib writer, and a flate
// compressor carries a window plus Huffman tables that dwarf the pixels of a
// small image. Paper encodes a PNG per gradient, SVG, barcode, and normalised
// raster, so those buffers were the largest single allocation source in
// document generation.
var encoderBufferPool = &bufferPool{}

type bufferPool struct {
	pool sync.Pool
}

func (p *bufferPool) Get() *png.EncoderBuffer {
	buffer, _ := p.pool.Get().(*png.EncoderBuffer)
	return buffer
}

func (p *bufferPool) Put(buffer *png.EncoderBuffer) {
	p.pool.Put(buffer)
}

// compressed and uncompressed are shared encoders. png.Encoder holds no
// per-call state of its own, so one instance per compression level is enough.
var (
	compressed   = &png.Encoder{BufferPool: encoderBufferPool}
	uncompressed = &png.Encoder{CompressionLevel: png.NoCompression, BufferPool: encoderBufferPool}
)

// Encode writes img to w as a PNG, reusing pooled encoder buffers.
func Encode(w io.Writer, img image.Image) error {
	return compressed.Encode(w, img)
}

// EncodeUncompressed writes img to w without deflate compression. Use it for
// buffers that are decoded again immediately: the PDF writer re-compresses the
// pixels itself, so compressing here is pure waste.
func EncodeUncompressed(w io.Writer, img image.Image) error {
	return uncompressed.Encode(w, img)
}
