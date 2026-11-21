package renderkit

import (
	"cmp"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"slices"
)

// EncodeMotionGIF encodes rendered frames using a shared palette.
func EncodeMotionGIF(rendered []image.Image, width, height, delay int) *gif.GIF {
	colors := AdaptivePalette(rendered, 255)
	clear := uint8(len(colors))
	colors = append(colors, color.RGBA{})
	nearest := map[color.RGBA]uint8{}
	var full []*image.Paletted
	for _, img := range rendered {
		indexed := image.NewPaletted(img.Bounds(), colors)
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				index, ok := nearest[c]
				if !ok {
					index = uint8(colors.Index(c))
					nearest[c] = index
				}
				indexed.SetColorIndex(x, y, index)
			}
		}
		full = append(full, indexed)
	}
	animation := &gif.GIF{LoopCount: 0}
	for i, indexed := range full {
		frame := indexed
		if i > 0 {
			frame = ChangedBox(full[i-1], indexed, clear)
		}
		animation.Image = append(animation.Image, frame)
		animation.Delay = append(animation.Delay, delay)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	animation.Config = image.Config{ColorModel: colors, Width: width, Height: height}
	return animation
}

// ChangedBox returns the changed portion of a frame.
func ChangedBox(before, next *image.Paletted, clear uint8) *image.Paletted {
	bounds := next.Bounds()
	box := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if before.ColorIndexAt(x, y) == next.ColorIndexAt(x, y) {
				continue
			}
			box.Min.X, box.Min.Y = min(box.Min.X, x), min(box.Min.Y, y)
			box.Max.X, box.Max.Y = max(box.Max.X, x+1), max(box.Max.Y, y+1)
		}
	}
	if box.Empty() {
		box = image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+1, bounds.Min.Y+1)
	}
	out := image.NewPaletted(box, next.Palette)
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			if before.ColorIndexAt(x, y) == next.ColorIndexAt(x, y) {
				out.SetColorIndex(x, y, clear)
				continue
			}
			out.SetColorIndex(x, y, next.ColorIndexAt(x, y))
		}
	}
	return out
}

// AdaptivePalette chooses the colors used most often by the frames.
func AdaptivePalette(images []image.Image, size int) color.Palette {
	type bucket struct {
		count   int
		r, g, b int
	}
	buckets := make(map[uint16]*bucket)
	for _, img := range images {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				key := uint16(c.R>>3)<<10 | uint16(c.G>>3)<<5 | uint16(c.B>>3)
				b := buckets[key]
				if b == nil {
					b = &bucket{}
					buckets[key] = b
				}
				b.count++
				b.r += int(c.R)
				b.g += int(c.G)
				b.b += int(c.B)
			}
		}
	}
	popular := make([]*bucket, 0, len(buckets))
	for _, b := range buckets {
		popular = append(popular, b)
	}
	slices.SortFunc(popular, func(a, b *bucket) int { return cmp.Compare(b.count, a.count) })
	if len(popular) > size {
		popular = popular[:size]
	}
	colors := make(color.Palette, 0, len(popular))
	for _, b := range popular {
		colors = append(colors, color.RGBA{R: uint8(b.r / b.count), G: uint8(b.g / b.count), B: uint8(b.b / b.count), A: 255})
	}
	return colors
}

// WriteGIF writes an animation to its output path.
func WriteGIF(output string, animation *gif.GIF) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".animation-*.gif")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()
	if err := gif.EncodeAll(file, animation); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Chmod(file.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(file.Name(), output)
}
