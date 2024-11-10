package main

import (
	"bytes"
	"fmt"
	"github.com/kbinani/screenshot"
	"image/png"
	"log"
	"os"
)

func captureScreen() ([]byte, error) {
	log.Println("capturing screen")
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		log.Println("found this many displays:", n)
		return nil, fmt.Errorf("no active display found")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %v", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}

	if DEVFLAG {
		f, err := os.Create("test.png")
		if err != nil {
			log.Println("Error creating file:", err)
		}
		defer f.Close()
		png.Encode(f, img)
	}

	return buf.Bytes(), nil
}
