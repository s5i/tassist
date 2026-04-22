//go:build windows

package exp

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/deluan/lookup"

	_ "embed"
	_ "image/png"
)

func NewReader() (*Reader, error) {
	ocr := lookup.NewOCR(0.7)
	if err := ocr.LoadFont(fontPath); err != nil {
		return nil, err
	}
	return &Reader{
		ocr: ocr,
	}, nil
}

type Reader struct {
	ocr *lookup.OCR
}

func (r *Reader) Read() (int, bool, error) {
	windowImg, err := captureWindow("Tibiantis")
	if err != nil {
		return 0, false, fmt.Errorf(`CaptureWindow("Tibiantis") failed: %v`, err)
	}

	rightBarImg := windowImg.SubImage(image.Rect(
		windowImg.Rect.Max.X-200,
		windowImg.Rect.Min.Y,
		windowImg.Rect.Max.X,
		windowImg.Rect.Max.Y,
	)).(*image.RGBA)

	expMatches, err := lookup.NewLookup(rightBarImg).FindAll(expKeyImg, 0.8)
	if err != nil {
		return 0, false, fmt.Errorf(`lookup.FindAll(expImg) failed: %v`, err)
	}

	if len(expMatches) == 0 {
		return 0, false, nil
	}

	expMatch := expMatches[0]
	for i := range expMatches {
		if expMatches[i].G > expMatch.G {
			expMatch = expMatches[i]
		}
	}

	expValX := rightBarImg.Rect.Min.X + expMatch.X + expKeyImg.Bounds().Dx()
	expValY := rightBarImg.Rect.Min.Y + expMatch.Y
	expValDX := 80
	expValDY := expKeyImg.Bounds().Dy()
	expValueImg := rightBarImg.SubImage(image.Rect(expValX, expValY, expValX+expValDX, expValY+expValDY))

	expStr, err := r.ocr.Recognize(expValueImg)
	if err != nil {
		return 0, false, fmt.Errorf("ocr.Recognize() failed: %v", err)
	}

	exp, err := strconv.Atoi(expStr)
	if err != nil {
		return 0, false, err
	}

	return exp, true, nil
}

var (
	//go:embed assets/experience.png
	expKeyBytes []byte
	expKeyImg   image.Image

	//go:embed assets/font
	fontFS embed.FS

	basePath = filepath.Join(os.TempDir(), "tassist")
	fontPath = filepath.Join(basePath, "assets", "font")
)

func init() {
	os.RemoveAll(basePath)
	if err := os.MkdirAll(basePath, 0755); err != nil {
		log.Fatalf("os.MkdirAll(%q) failed: %v", basePath, err)
	}
	if err := os.CopyFS(basePath, fontFS); err != nil {
		log.Fatalf("os.CopyFS(%q) failed: %v", basePath, err)
	}

	eImg, _, err := image.Decode(bytes.NewReader(expKeyBytes))
	if err != nil {
		log.Fatalf("image.Decode(expKeyBytes) failed: %v", err)
	}
	expKeyImg = eImg
}
