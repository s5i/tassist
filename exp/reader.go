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
	"strings"

	"github.com/deluan/lookup"

	_ "embed"
	_ "image/png"
)

func NewReader(tmpDir string, windowTitle string) (*Reader, error) {
	baseDir := filepath.Join(tmpDir, "exp")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("os.MkdirAll(%q) failed: %v", baseDir, err)
	}
	if err := os.CopyFS(baseDir, fontFS); err != nil {
		return nil, fmt.Errorf("os.CopyFS(%q) failed: %v", baseDir, err)
	}
	defer os.RemoveAll(baseDir)

	eImg, _, err := image.Decode(bytes.NewReader(expKeyBytes))
	if err != nil {
		log.Fatalf("image.Decode(expKeyBytes) failed: %v", err)
	}

	ocr := lookup.NewOCR(0.6)
	if err := ocr.LoadFont(filepath.Join(baseDir, "assets/font")); err != nil {
		return nil, err
	}

	return &Reader{
		ocr:       ocr,
		expKeyImg: eImg,
		wc:        newWindowCapturer(windowTitle),
	}, nil
}

type Reader struct {
	ocr       *lookup.OCR
	expKeyImg image.Image
	wc        *windowCapturer
}

func (r *Reader) Read() (int, bool, error) {
	windowImg, err := r.wc.capture()
	if err != nil {
		return 0, false, fmt.Errorf(`capture failed: %v`, err)
	}

	rightBarImg := windowImg.SubImage(image.Rect(
		windowImg.Rect.Max.X-200,
		windowImg.Rect.Min.Y,
		windowImg.Rect.Max.X,
		windowImg.Rect.Max.Y,
	)).(*image.RGBA)

	expMatches, err := lookup.NewLookup(rightBarImg).FindAll(r.expKeyImg, 0.6)
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

	expValX := rightBarImg.Rect.Min.X + expMatch.X + r.expKeyImg.Bounds().Dx()
	expValY := rightBarImg.Rect.Min.Y + expMatch.Y
	expValDX := 80
	expValDY := r.expKeyImg.Bounds().Dy()
	expValueImg := rightBarImg.SubImage(image.Rect(expValX, expValY, expValX+expValDX, expValY+expValDY))

	expStr, err := r.ocr.Recognize(expValueImg)
	if err != nil {
		return 0, false, fmt.Errorf("ocr.Recognize() failed: %v", err)
	}

	exp, err := strconv.Atoi(strings.ReplaceAll(expStr, " ", ""))
	if err != nil {
		return 0, false, err
	}

	return exp, true, nil
}

var (
	//go:embed assets/experience.png
	expKeyBytes []byte

	//go:embed assets/font
	fontFS embed.FS
)
