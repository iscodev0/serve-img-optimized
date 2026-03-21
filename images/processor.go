package images

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/disintegration/imaging"
	"golang.org/x/image/webp"
)

func init() {
	// Registrar formatos de imagen soportados globalmente, solo una vez
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("gif", "GIF8", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("webp", "RIFF????WEBP", webp.Decode, webp.DecodeConfig)
}

// ImageProcessor maneja el procesamiento de imágenes
type ImageProcessor struct{}

// NewImageProcessor crea una nueva instancia del procesador
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

// ProcessImage redimensiona y optimiza una imagen
func (ip *ImageProcessor) ProcessImage(imageData []byte, width, quality int) ([]byte, error) {
	// Decodificar imagen
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format: %s): %v", format, err)
	}

	// Redimensionar imagen manteniendo aspect ratio
	resizedImg := imaging.Resize(img, width, 0, imaging.Lanczos)

	// Codificar como JPEG con calidad especificada
	var buf bytes.Buffer
	options := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(&buf, resizedImg, options); err != nil {
		return nil, fmt.Errorf("failed to encode image: %v", err)
	}

	return buf.Bytes(), nil
}

// ValidateImageURL verifica si la URL apunta a una imagen válida
func (ip *ImageProcessor) ValidateImageURL(contentType string) error {
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("URL does not point to an image: %s", contentType)
	}
	return nil
}

// GetImageInfo obtiene información básica de una imagen
func (ip *ImageProcessor) GetImageInfo(imageData []byte) (map[string]interface{}, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"width":  config.Width,
		"height": config.Height,
		"format": format,
	}, nil
}
