package images

import (
	"fmt"
	"net/http"

	"github.com/loxzer01/serve-img-optimized/cache"
)

// ImageOptimizer coordina todo el proceso de optimización de imágenes
type ImageOptimizer struct {
	cacheManager *cache.CacheManager
	downloader   *ImageDownloader
	processor    *ImageProcessor
	paramsParser *ParamsParser
}

// NewImageOptimizer crea una nueva instancia del optimizador
func NewImageOptimizer(cacheManager *cache.CacheManager) *ImageOptimizer {
	return &ImageOptimizer{
		cacheManager: cacheManager,
		downloader:   NewImageDownloader(),
		processor:    NewImageProcessor(),
		paramsParser: NewParamsParser(),
	}
}

// OptimizeImage procesa una imagen según los parámetros especificados
func (io *ImageOptimizer) OptimizeImage(r *http.Request) ([]byte, string, error) {
	// 1. Parsear parámetros
	params, err := io.paramsParser.ParseURLParams(r)
	if err != nil {
		return nil, "", fmt.Errorf("parameter parsing error: %v", err)
	}

	// 2. Generar clave de caché
	cacheKey := io.cacheManager.GenerateCacheKey(params.URL, params.Width, params.Quality)

	// 3. Verificar caché
	if cachedData, found := io.cacheManager.GetCachedImage(cacheKey); found {
		return cachedData, "image/jpeg", nil
	}

	// 4. Descargar imagen
	imageData, err := io.downloader.DownloadImage(params.URL, params.Origin, params.Referer)
	if err != nil {
		return nil, "", fmt.Errorf("download error: %v", err)
	}

	// 5. Procesar imagen
	processedData, err := io.processor.ProcessImage(imageData, params.Width, params.Quality)
	if err != nil {
		return nil, "", fmt.Errorf("processing error: %v", err)
	}

	// 6. Guardar en caché
	if err := io.cacheManager.SaveImageToCache(cacheKey, processedData); err != nil {
		// Log error pero no fallar la respuesta
		fmt.Printf("Warning: Failed to save to cache: %v\n", err)
	}

	return processedData, "image/jpeg", nil
}

// GetImageInfo obtiene información de una imagen sin procesarla
func (io *ImageOptimizer) GetImageInfo(imageURL string) (map[string]interface{}, error) {
	// Descargar imagen
	imageData, err := io.downloader.DownloadImage(imageURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("download error: %v", err)
	}

	// Obtener información
	return io.processor.GetImageInfo(imageData)
}

// ValidateImageURL valida si una URL contiene una imagen válida
func (io *ImageOptimizer) ValidateImageURL(imageURL string) error {
	return io.processor.ValidateImageURL(imageURL)
}

// GetCacheStats obtiene estadísticas del caché
func (io *ImageOptimizer) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size": io.cacheManager.GetCacheSize(),
		"cache_dir":  io.cacheManager.GetCacheDir(),
	}
}

// CleanupCache limpia archivos antiguos del caché
func (io *ImageOptimizer) CleanupCache() error {
	return io.cacheManager.CleanupOldFiles()
}
