package processor

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"

	"github.com/malbinjvc/pixel-forge/internal/model"
)

func Resize(img image.Image, params model.ResizeParams) (image.Image, error) {
	if params.Width <= 0 || params.Height <= 0 {
		return nil, fmt.Errorf("invalid dimensions: %dx%d", params.Width, params.Height)
	}
	if params.Width > 10000 || params.Height > 10000 {
		return nil, fmt.Errorf("dimensions too large: max 10000x10000")
	}

	return resizeNearest(img, params.Width, params.Height), nil
}

func Crop(img image.Image, params model.CropParams) (image.Image, error) {
	bounds := img.Bounds()

	if params.X < 0 || params.Y < 0 {
		return nil, fmt.Errorf("crop coordinates must be non-negative")
	}
	if params.Width <= 0 || params.Height <= 0 {
		return nil, fmt.Errorf("crop dimensions must be positive")
	}
	if params.X+params.Width > bounds.Dx() || params.Y+params.Height > bounds.Dy() {
		return nil, fmt.Errorf("crop region exceeds image bounds (%dx%d)", bounds.Dx(), bounds.Dy())
	}

	rect := image.Rect(params.X, params.Y, params.X+params.Width, params.Y+params.Height)
	cropped := image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))
	draw.Draw(cropped, cropped.Bounds(), img, rect.Min, draw.Src)

	return cropped, nil
}

func Rotate(img image.Image, params model.RotateParams) (image.Image, error) {
	angle := int(params.Angle) % 360
	if angle < 0 {
		angle += 360
	}

	switch angle {
	case 0:
		return img, nil
	case 90:
		return rotate90(img), nil
	case 180:
		return rotate180(img), nil
	case 270:
		return rotate270(img), nil
	default:
		return nil, fmt.Errorf("only 90, 180, 270 degree rotations supported")
	}
}

func ApplyFilter(img image.Image, params model.FilterParams) (image.Image, error) {
	switch params.Type {
	case "grayscale":
		return grayscale(img), nil
	case "brightness":
		return brightness(img, params.Intensity), nil
	case "contrast":
		return contrast(img, params.Intensity), nil
	case "invert":
		return invert(img), nil
	case "sepia":
		return sepia(img), nil
	default:
		return nil, fmt.Errorf("unknown filter: %s", params.Type)
	}
}

func Convert(data []byte, params model.ConvertParams) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	var buf bytes.Buffer
	quality := params.Quality
	if quality <= 0 {
		quality = 85
	}
	if quality > 100 {
		quality = 100
	}

	switch params.Format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	case "png":
		err = png.Encode(&buf, img)
	case "gif":
		err = gif.Encode(&buf, img, nil)
	default:
		return nil, fmt.Errorf("unsupported format: %s", params.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", params.Format, err)
	}

	return buf.Bytes(), nil
}

func Encode(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer

	if quality <= 0 {
		quality = 85
	}

	switch format {
	case "jpeg", "jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		return buf.Bytes(), err
	case "png":
		err := png.Encode(&buf, img)
		return buf.Bytes(), err
	case "gif":
		err := gif.Encode(&buf, img, nil)
		return buf.Bytes(), err
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func Decode(data []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(data))
}

// Resize using nearest-neighbor interpolation
func resizeNearest(img image.Image, width, height int) image.Image {
	src := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(src.Dx()) / float64(width)
	yRatio := float64(src.Dy()) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x)*xRatio) + src.Min.X
			srcY := int(float64(y)*yRatio) + src.Min.Y
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	return dst
}

func rotate90(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.Y-1-y, x, img.At(x, y))
		}
	}
	return dst
}

func rotate180(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x, b.Max.Y-1-y, img.At(x, y))
		}
	}
	return dst
}

func rotate270(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(y, b.Max.X-1-x, img.At(x, y))
		}
	}
	return dst
}

func grayscale(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b_, a := img.At(x, y).RGBA()
			gray := uint8((0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b_)) / 256)
			dst.Set(x, y, color.NRGBA{R: gray, G: gray, B: gray, A: uint8(a >> 8)})
		}
	}
	return dst
}

func brightness(img image.Image, factor float64) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			dst.Set(x, y, color.NRGBA{
				R: clampU8(float64(r>>8) + factor*255),
				G: clampU8(float64(g>>8) + factor*255),
				B: clampU8(float64(bl>>8) + factor*255),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

func contrast(img image.Image, factor float64) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	f := (259 * (factor*255 + 255)) / (255 * (259 - factor*255))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			dst.Set(x, y, color.NRGBA{
				R: clampU8(f*(float64(r>>8)-128) + 128),
				G: clampU8(f*(float64(g>>8)-128) + 128),
				B: clampU8(f*(float64(bl>>8)-128) + 128),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

func invert(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			dst.Set(x, y, color.NRGBA{
				R: uint8(255 - r>>8),
				G: uint8(255 - g>>8),
				B: uint8(255 - bl>>8),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

func sepia(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			rf, gf, bf := float64(r>>8), float64(g>>8), float64(bl>>8)
			dst.Set(x, y, color.NRGBA{
				R: clampU8(0.393*rf + 0.769*gf + 0.189*bf),
				G: clampU8(0.349*rf + 0.686*gf + 0.168*bf),
				B: clampU8(0.272*rf + 0.534*gf + 0.131*bf),
				A: uint8(a >> 8),
			})
		}
	}
	return dst
}

func clampU8(v float64) uint8 {
	return uint8(math.Max(0, math.Min(255, v)))
}
