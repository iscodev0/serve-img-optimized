package cache

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type CacheManager struct {
	CacheDir      string
	CacheDuration time.Duration
	MaxCacheSize  int64 // en bytes
	currentSize   atomic.Int64
}

// NewCacheManager crea una nueva instancia del gestor de cache
func NewCacheManager(cacheDir string, cacheDuration time.Duration, maxCacheSizeMB int) *CacheManager {
	// Crear directorio de cache si no existe
	os.MkdirAll(cacheDir, 0755)

	cm := &CacheManager{
		CacheDir:      cacheDir,
		CacheDuration: cacheDuration,
		MaxCacheSize:  int64(maxCacheSizeMB) * 1024 * 1024, // convertir MB a bytes
	}

	// Precalcular el tamaño del folder 1 sola vez
	size, _ := cm.calculateFolderSize()
	cm.currentSize.Store(size)

	return cm
}

// GenerateCacheKey genera una clave única para el cache basada en los parámetros
func (cm *CacheManager) GenerateCacheKey(url string, width int, quality int) string {
	data := fmt.Sprintf("%s_%d_%d", url, width, quality)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x.jpg", hash)
}

// GetCachedImage verifica si existe una imagen en cache y si no ha expirado
func (cm *CacheManager) GetCachedImage(cacheKey string) ([]byte, bool) {
	cachePath := filepath.Join(cm.CacheDir, cacheKey)

	// Verificar si el archivo existe
	fileInfo, err := os.Stat(cachePath)
	if err != nil {
		return nil, false
	}

	// Verificar si el archivo no ha expirado
	if time.Since(fileInfo.ModTime()) > cm.CacheDuration {
		// Archivo expirado, eliminarlo
		if err := os.Remove(cachePath); err == nil {
			cm.currentSize.Add(-fileInfo.Size())
		}
		return nil, false
	}

	// Leer el archivo
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	return data, true
}

// SaveToCache guarda una imagen en el cache
func (cm *CacheManager) SaveToCache(cacheKey string, data []byte) error {
	// Verificar espacio disponible antes de guardar
	if err := cm.cleanupIfNeeded(int64(len(data))); err != nil {
		return err
	}

	cachePath := filepath.Join(cm.CacheDir, cacheKey)
	err := os.WriteFile(cachePath, data, 0644)
	if err == nil {
		cm.currentSize.Add(int64(len(data)))
	}
	return err
}

// cleanupIfNeeded limpia archivos antiguos si es necesario para hacer espacio
func (cm *CacheManager) cleanupIfNeeded(newFileSize int64) error {
	currentSize, err := cm.getCacheSize()
	if err != nil {
		return err
	}

	if currentSize+newFileSize <= cm.MaxCacheSize {
		return nil // No necesita limpieza
	}

	// Obtener lista de archivos ordenados por fecha de modificación
	files, err := cm.getCacheFilesSorted()
	if err != nil {
		return err
	}

	// Eliminar archivos más antiguos hasta tener espacio suficiente
	for _, file := range files {
		if currentSize+newFileSize <= cm.MaxCacheSize {
			break
		}

		filePath := filepath.Join(cm.CacheDir, file.Name())
		fileSize := file.Size()

		if err := os.Remove(filePath); err == nil {
			currentSize -= fileSize
			cm.currentSize.Add(-fileSize)
		}
	}

	return nil
}

// getCacheSize calcula el tamaño total del cache rápidamente (O(1))
func (cm *CacheManager) getCacheSize() (int64, error) {
	return cm.currentSize.Load(), nil
}

// calculateFolderSize recorre el disco para calcular tamaño base
func (cm *CacheManager) calculateFolderSize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(cm.CacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// getCacheFilesSorted obtiene archivos del cache ordenados por fecha de modificación (más antiguos primero)
func (cm *CacheManager) getCacheFilesSorted() ([]os.FileInfo, error) {
	files, err := os.ReadDir(cm.CacheDir)
	if err != nil {
		return nil, err
	}

	var fileInfos []os.FileInfo
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err == nil {
				fileInfos = append(fileInfos, info)
			}
		}
	}

	// Ordenar por fecha de modificación (más antiguos primero)
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].ModTime().Before(fileInfos[j].ModTime())
	})

	return fileInfos, nil
}

// SaveImageToCache es un alias para SaveToCache para compatibilidad
func (cm *CacheManager) SaveImageToCache(cacheKey string, data []byte) error {
	return cm.SaveToCache(cacheKey, data)
}

// GetCacheSize retorna el tamaño actual del caché
func (cm *CacheManager) GetCacheSize() string {
	size, err := cm.getCacheSize()
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
}

// GetCacheDir retorna el directorio del caché
func (cm *CacheManager) GetCacheDir() string {
	return cm.CacheDir
}

// CleanupOldFiles limpia archivos expirados del caché
func (cm *CacheManager) CleanupOldFiles() error {
	files, err := cm.getCacheFilesSorted()
	if err != nil {
		return err
	}

	for _, file := range files {
		if time.Since(file.ModTime()) > cm.CacheDuration {
			filePath := filepath.Join(cm.CacheDir, file.Name())
			if err := os.Remove(filePath); err == nil {
				cm.currentSize.Add(-file.Size())
			}
		}
	}
	return nil
}

// ParseCacheDuration convierte string de duración a time.Duration
func ParseCacheDuration(duration string) time.Duration {
	if duration == "" {
		return time.Hour // default 1 hora
	}

	duration = strings.ToLower(strings.TrimSpace(duration))

	// Extraer número y unidad
	var num string
	var unit string

	for i, char := range duration {
		if char >= '0' && char <= '9' {
			num += string(char)
		} else {
			unit = duration[i:]
			break
		}
	}

	value, err := strconv.Atoi(num)
	if err != nil {
		return time.Hour // default en caso de error
	}

	switch unit {
	case "min":
		return time.Duration(value) * time.Minute
	case "h":
		return time.Duration(value) * time.Hour
	case "d":
		return time.Duration(value) * 24 * time.Hour
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour
	case "m": // month
		return time.Duration(value) * 30 * 24 * time.Hour
	case "y": // year
		return time.Duration(value) * 365 * 24 * time.Hour
	default:
		return time.Hour // default en caso de unidad no soportada
	}
}
