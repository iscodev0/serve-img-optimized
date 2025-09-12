package images

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// ImageDownloader maneja la descarga de imágenes desde URLs
type ImageDownloader struct {
	client *http.Client
}

// NewImageDownloader crea una nueva instancia del descargador
func NewImageDownloader() *ImageDownloader {
	return &ImageDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DownloadImage descarga una imagen desde la URL especificada
func (id *ImageDownloader) DownloadImage(imageURL, origin, referer string) ([]byte, error) {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Establecer headers para evitar bloqueos
	id.setHeaders(req, origin, referer)

	resp, err := id.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Verificar content-type
	contentType := resp.Header.Get("Content-Type")
	processor := NewImageProcessor()
	if err := processor.ValidateImageURL(contentType); err != nil {
		return nil, err
	}

	// Leer el contenido
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %v", err)
	}

	return buf.Bytes(), nil
}

// setHeaders configura los headers necesarios para la petición
func (id *ImageDownloader) setHeaders(req *http.Request, origin, referer string) {
	// User-Agent para evitar bloqueos
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	
	// Usar referer si está disponible, sino usar origin
	refererValue := referer
	if refererValue == "" {
		refererValue = origin
	}
	
	// Headers de origen si se especifica
	if refererValue != "" {
		req.Header.Set("Referer", "https://"+refererValue)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	
	// Headers adicionales
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
}