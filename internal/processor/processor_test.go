package processor

import (
	"image"
	"image/color"
	"testing"

	"github.com/malbinjvc/pixel-forge/internal/model"
)

func testImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	return img
}

func TestResize(t *testing.T) {
	img := testImage(100, 80)

	result, err := Resize(img, model.ResizeParams{Width: 50, Height: 40})
	if err != nil {
		t.Fatalf("resize failed: %v", err)
	}

	b := result.Bounds()
	if b.Dx() != 50 || b.Dy() != 40 {
		t.Errorf("expected 50x40, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestResizeInvalidDimensions(t *testing.T) {
	img := testImage(100, 80)

	_, err := Resize(img, model.ResizeParams{Width: -1, Height: 40})
	if err == nil {
		t.Error("expected error for negative width")
	}

	_, err = Resize(img, model.ResizeParams{Width: 20000, Height: 40})
	if err == nil {
		t.Error("expected error for oversized width")
	}
}

func TestCrop(t *testing.T) {
	img := testImage(100, 80)

	result, err := Crop(img, model.CropParams{X: 10, Y: 10, Width: 50, Height: 40})
	if err != nil {
		t.Fatalf("crop failed: %v", err)
	}

	b := result.Bounds()
	if b.Dx() != 50 || b.Dy() != 40 {
		t.Errorf("expected 50x40, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestCropOutOfBounds(t *testing.T) {
	img := testImage(100, 80)

	_, err := Crop(img, model.CropParams{X: 60, Y: 50, Width: 50, Height: 40})
	if err == nil {
		t.Error("expected error for out-of-bounds crop")
	}
}

func TestRotate90(t *testing.T) {
	img := testImage(100, 80)

	result, err := Rotate(img, model.RotateParams{Angle: 90})
	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}

	b := result.Bounds()
	if b.Dx() != 80 || b.Dy() != 100 {
		t.Errorf("expected 80x100 after 90deg rotation, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestRotate180(t *testing.T) {
	img := testImage(100, 80)

	result, err := Rotate(img, model.RotateParams{Angle: 180})
	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}

	b := result.Bounds()
	if b.Dx() != 100 || b.Dy() != 80 {
		t.Errorf("expected 100x80 after 180deg rotation, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestRotateInvalidAngle(t *testing.T) {
	img := testImage(100, 80)

	_, err := Rotate(img, model.RotateParams{Angle: 45})
	if err == nil {
		t.Error("expected error for unsupported angle")
	}
}

func TestFilterGrayscale(t *testing.T) {
	img := testImage(10, 10)

	result, err := ApplyFilter(img, model.FilterParams{Type: "grayscale"})
	if err != nil {
		t.Fatalf("filter failed: %v", err)
	}

	r, g, b, _ := result.At(5, 5).RGBA()
	if r != g || g != b {
		t.Error("grayscale should produce equal RGB values")
	}
}

func TestFilterInvert(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 100, G: 150, B: 200, A: 255})

	result, err := ApplyFilter(img, model.FilterParams{Type: "invert"})
	if err != nil {
		t.Fatalf("filter failed: %v", err)
	}

	r, g, b, _ := result.At(0, 0).RGBA()
	// 255-100=155, 255-150=105, 255-200=55
	if uint8(r>>8) != 155 || uint8(g>>8) != 105 || uint8(b>>8) != 55 {
		t.Errorf("invert produced wrong values: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestFilterSepia(t *testing.T) {
	img := testImage(10, 10)

	_, err := ApplyFilter(img, model.FilterParams{Type: "sepia"})
	if err != nil {
		t.Fatalf("sepia filter failed: %v", err)
	}
}

func TestFilterUnknown(t *testing.T) {
	img := testImage(10, 10)

	_, err := ApplyFilter(img, model.FilterParams{Type: "unknown"})
	if err == nil {
		t.Error("expected error for unknown filter")
	}
}

func TestEncodeDecode(t *testing.T) {
	img := testImage(50, 50)

	for _, format := range []string{"jpeg", "png", "gif"} {
		data, err := Encode(img, format, 85)
		if err != nil {
			t.Fatalf("encode %s failed: %v", format, err)
		}
		if len(data) == 0 {
			t.Errorf("encode %s produced empty data", format)
		}

		decoded, _, err := Decode(data)
		if err != nil {
			t.Fatalf("decode %s failed: %v", format, err)
		}
		if decoded.Bounds().Dx() != 50 {
			t.Errorf("decoded %s has wrong width", format)
		}
	}
}

func TestConvert(t *testing.T) {
	img := testImage(50, 50)
	pngData, _ := Encode(img, "png", 85)

	jpegData, err := Convert(pngData, model.ConvertParams{Format: "jpeg", Quality: 80})
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if len(jpegData) == 0 {
		t.Error("convert produced empty data")
	}
}

func TestConvertInvalidFormat(t *testing.T) {
	img := testImage(50, 50)
	pngData, _ := Encode(img, "png", 85)

	_, err := Convert(pngData, model.ConvertParams{Format: "bmp"})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestBrightness(t *testing.T) {
	img := testImage(10, 10)

	_, err := ApplyFilter(img, model.FilterParams{Type: "brightness", Intensity: 0.5})
	if err != nil {
		t.Fatalf("brightness filter failed: %v", err)
	}
}

func TestContrast(t *testing.T) {
	img := testImage(10, 10)

	_, err := ApplyFilter(img, model.FilterParams{Type: "contrast", Intensity: 0.5})
	if err != nil {
		t.Fatalf("contrast filter failed: %v", err)
	}
}
