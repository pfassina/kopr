package editor

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/image/draw"
)

// ProcessedImage holds an image that has been loaded, resized, and encoded
// for transmission via the Kitty graphics protocol.
type ProcessedImage struct {
	WidthCells  int
	HeightCells int
	WidthPx     int
	HeightPx    int
	Base64Data  string // PNG data, base64-encoded
	ModTime     int64  // file mod time for cache invalidation
}

// ImageCache caches processed images keyed by absolute path.
type ImageCache struct {
	mu    sync.Mutex
	items map[string]*ProcessedImage
}

// NewImageCache creates an empty image cache.
func NewImageCache() *ImageCache {
	return &ImageCache{items: make(map[string]*ProcessedImage)}
}

// Get returns a cached image if the file hasn't changed, or nil.
func (c *ImageCache) Get(absPath string) *ProcessedImage {
	c.mu.Lock()
	defer c.mu.Unlock()
	cached, ok := c.items[absPath]
	if !ok {
		return nil
	}
	info, err := os.Stat(absPath)
	if err != nil || info.ModTime().UnixNano() != cached.ModTime {
		delete(c.items, absPath)
		return nil
	}
	return cached
}

// Put stores a processed image in the cache.
func (c *ImageCache) Put(absPath string, img *ProcessedImage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[absPath] = img
}

// LoadImage loads an image from disk, resizes it to fit within the given cell
// dimensions, and encodes it for the Kitty graphics protocol.
// cellWidth and cellHeight are approximate pixel dimensions per cell.
func LoadImage(path string, maxCols, maxRows, cellWidth, cellHeight int) (_ *ProcessedImage, retErr error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	var img image.Image
	switch ext {
	case ".png":
		img, err = png.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	case ".gif":
		img, err = gif.Decode(f)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", ext, err)
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if origW == 0 || origH == 0 {
		return nil, fmt.Errorf("image has zero dimensions")
	}

	// Calculate target pixel dimensions based on cell budget
	maxPxW := maxCols * cellWidth
	maxPxH := maxRows * cellHeight

	// Scale down to fit, preserving aspect ratio
	scale := 1.0
	if origW > maxPxW || origH > maxPxH {
		scaleW := float64(maxPxW) / float64(origW)
		scaleH := float64(maxPxH) / float64(origH)
		if scaleW < scaleH {
			scale = scaleW
		} else {
			scale = scaleH
		}
	}

	newW := int(float64(origW) * scale)
	newH := int(float64(origH) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	// Resize using high-quality interpolation
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// Encode to PNG
	var buf strings.Builder
	b64w := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := png.Encode(b64w, dst); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	if err := b64w.Close(); err != nil {
		return nil, fmt.Errorf("close base64 encoder: %w", err)
	}

	// Compute cell dimensions
	widthCells := (newW + cellWidth - 1) / cellWidth
	heightCells := (newH + cellHeight - 1) / cellHeight
	if widthCells < 1 {
		widthCells = 1
	}
	if heightCells < 1 {
		heightCells = 1
	}

	return &ProcessedImage{
		WidthCells:  widthCells,
		HeightCells: heightCells,
		WidthPx:     newW,
		HeightPx:    newH,
		Base64Data:  buf.String(),
		ModTime:     info.ModTime().UnixNano(),
	}, nil
}

// KittyTransmit generates a Kitty graphics protocol escape sequence that
// uploads the image data. Uses chunked transmission for large payloads.
// The image is assigned the given numeric ID for later placement.
func KittyTransmit(id uint32, img *ProcessedImage) string {
	data := img.Base64Data
	var b strings.Builder

	// Kitty protocol chunks must be at most 4096 bytes of payload.
	const chunkSize = 4096

	if len(data) <= chunkSize {
		// Single chunk: transmit and display
		fmt.Fprintf(&b, "\x1b_Gi=%d,f=100,a=T,t=d,s=%d,v=%d,C=1;%s\x1b\\",
			id, img.WidthPx, img.HeightPx, data)
		return b.String()
	}

	// Multi-chunk: first chunk with m=1 (more data follows)
	first := data[:chunkSize]
	rest := data[chunkSize:]
	fmt.Fprintf(&b, "\x1b_Gi=%d,f=100,a=T,t=d,s=%d,v=%d,C=1,m=1;%s\x1b\\",
		id, img.WidthPx, img.HeightPx, first)

	// Middle chunks
	for len(rest) > chunkSize {
		chunk := rest[:chunkSize]
		rest = rest[chunkSize:]
		fmt.Fprintf(&b, "\x1b_Gm=1;%s\x1b\\", chunk)
	}

	// Final chunk with m=0
	fmt.Fprintf(&b, "\x1b_Gm=0;%s\x1b\\", rest)

	return b.String()
}

// KittyPlace generates a Kitty graphics protocol escape sequence that
// places a previously uploaded image at the current cursor position.
// C=1 prevents the cursor from moving.
func KittyPlace(id uint32, cols, rows int) string {
	return fmt.Sprintf("\x1b_Ga=p,i=%d,C=1,c=%d,r=%d;\x1b\\", id, cols, rows)
}

// KittyDelete generates a Kitty graphics protocol escape sequence that
// deletes an image by ID.
func KittyDelete(id uint32) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d;\x1b\\", id)
}
