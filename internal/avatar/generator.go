package avatar

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

type Generator struct {
	layers [][]image.Image
	width  int
	height int
}

func NewGenerator(assetsDir string) (*Generator, error) {
	var allLayers [][]image.Image
	var width, height int
	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to laod folder: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		layerDir := filepath.Join(assetsDir, entry.Name())
		files, err := os.ReadDir(layerDir)
		if err != nil {
			return nil, fmt.Errorf("failed to laod file: %w", err)
		}
		var currentLayer []image.Image
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".png" {
				continue
			}
			filePath := filepath.Join(layerDir, file.Name())
			f, err := os.Open(filePath)
			if err != nil {
				return nil, fmt.Errorf("Failed to open picture: %w", err)
			}
			img, err := png.Decode(f)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode picture: %w", err)
			}
			currentLayer = append(currentLayer, img)
			if width == 0 {
				width = img.Bounds().Dx()
				height = img.Bounds().Dy()
			}
			f.Close()
		}
		if len(currentLayer) > 0 {
			allLayers = append(allLayers, currentLayer)
		}
	}
	if len(allLayers) == 0 {
		return nil, fmt.Errorf("no avatar layers found in %s", assetsDir)
	}
	return &Generator{
		layers: allLayers,
		width:  width,
		height: height,
	}, nil
}

func (g *Generator) Generate(seed string) ([]byte, error) {
	hash := sha256.Sum256([]byte(seed))
	dst := image.NewRGBA(image.Rect(0, 0, g.width, g.height))
	for i, layerOptions := range g.layers {
		if len(layerOptions) == 0 {
			continue
		}
		idx := int(hash[i%len(hash)]) % len(layerOptions)
		img := layerOptions[idx]
		draw.Draw(dst, dst.Bounds(), img, image.Point{}, draw.Over)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, fmt.Errorf("failed to encode png: %w", err)
	}
	return buf.Bytes(), nil
}
