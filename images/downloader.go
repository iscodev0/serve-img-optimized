package images

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ImageDownloader maneja la descarga de imágenes desde URLs
type ImageDownloader struct {
	client *http.Client
}

// NewImageDownloader crea una nueva instancia del descargador
func NewImageDownloader() *ImageDownloader {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	return &ImageDownloader{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

const imageProxyBaseURL = "https://image-proxy.mangasx.online/img"

func getProxiedImageUrl(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	u, err := url.Parse(imageURL)
	if err != nil {
		return imageURL
	}

	if u.Host == "image-proxy.mangasx.online" {
		return imageURL
	}

	proxyURL, err := url.Parse(imageProxyBaseURL)
	if err != nil {
		return imageURL
	}

	q := proxyURL.Query()
	q.Set("url", imageURL)
	originalOrigin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	q.Set("origin", originalOrigin)
	proxyURL.RawQuery = q.Encode()

	return proxyURL.String()
}

// DownloadImage descarga una imagen intentando primero la original, y si falla, usando el proxy
func (id *ImageDownloader) DownloadImage(imageURL, origin, referer string) ([]byte, error) {
	data, err := id.download(imageURL, origin, referer)
	if err != nil {
		// Intentar descargar con la URL usando nuestro proxy
		proxiedURL := getProxiedImageUrl(imageURL)
		if proxiedURL != imageURL {
			proxyData, proxyErr := id.download(proxiedURL, origin, referer)
			if proxyErr == nil {
				return proxyData, nil
			}
			return nil, fmt.Errorf("fallo original: %v \n fallo proxy: %v", err, proxyErr)
		}
		return nil, err
	}
	return data, nil
}

// download realiza la descarga real modificando los headers y leyendo la respuesta
func (id *ImageDownloader) download(imageURL, origin, referer string) ([]byte, error) {
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