package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/loxzer01/serve-img-optimized/cache"
	"github.com/loxzer01/serve-img-optimized/images"
)

func NewRoutes() *chi.Mux {
	r := chi.NewRouter()

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // En producción, especifica dominios específicos
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Configurar cache manager
	cacheManager, err := setupCacheManager()
	if err != nil {
		fmt.Errorf("failed to setup cache manager: %v", err)
		return nil
	}

	// Crear optimizador de imágenes
	imageOptimizer := images.NewImageOptimizer(cacheManager)

	// Rutas de la API
	r.Route("/api", func(r chi.Router) {
		// Aplicar middleware de autenticación a todas las rutas de la API
		r.Use(authMiddleware)

		// Ruta para optimización de imágenes
		// Formato: /api/image/w_400,q_90,r_domain.com/example.com/image.jpg?origin="domain.com"
		r.Get("/image/*", OptimizeImageHandler(imageOptimizer))

		// Información de imagen
		r.Get("/info", ImageInfoHandler(imageOptimizer))

		// Estadísticas de caché
		r.Get("/cache/stats", CacheStatsHandler(imageOptimizer))

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","service":"image-optimizer"}`))
		})
	})

	// Ruta principal
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Image Optimization Server - Use /w_400,q_90,r_domain.com/image-url?origin=domain.com"))
	})

	// Ruta para optimización de imágenes (sin autenticación para compatibilidad Testing)
	// Formato: /w_400,q_90,r_domain.com/url?origin="dominio.com"
	// r.HandleFunc("/*", OptimizeImageHandler(imageOptimizer))

	return r
}

// setupCacheManager configura el gestor de cache basado en variables de entorno
func setupCacheManager() (*cache.CacheManager, error) {
	// Obtener configuración del cache desde variables de entorno
	cacheDurationStr := os.Getenv("TIME_CACHE")
	if cacheDurationStr == "" {
		cacheDurationStr = "1h" // default
	}

	cacheDir := os.Getenv("CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "./cache/images" // default
	}

	maxCacheSizeStr := os.Getenv("MAX_CACHE_SIZE")
	maxCacheSize := 1000 // default 1000MB
	if maxCacheSizeStr != "" {
		if size, err := strconv.Atoi(maxCacheSizeStr); err == nil {
			maxCacheSize = size
		}
	}

	// Parsear duración del cache
	cacheDuration := cache.ParseCacheDuration(cacheDurationStr)

	fmt.Printf("Cache configuration: Duration=%v, Directory=%s, MaxSize=%dMB\n",
		cacheDuration, cacheDir, maxCacheSize)

	cacheManager := cache.NewCacheManager(cacheDir, cacheDuration, maxCacheSize)

	// Iniciar limpieza automática en segundo plano
	startAutomaticCleanup(cacheManager, cacheDuration)

	return cacheManager, nil
}

// startAutomaticCleanup inicia una goroutine que limpia archivos expirados periódicamente
func startAutomaticCleanup(cacheManager *cache.CacheManager, cacheDuration time.Duration) {
	// Calcular intervalo de limpieza (cada 1/4 de la duración del cache, mínimo 1 minuto)
	cleanupInterval := cacheDuration / 4
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}

	fmt.Printf("Starting automatic cache cleanup every %v\n", cleanupInterval)

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := cacheManager.CleanupOldFiles(); err != nil {
					fmt.Printf("Error during automatic cache cleanup: %v\n", err)
				} else {
					fmt.Printf("Automatic cache cleanup completed\n")
				}
			}
		}
	}()
}

// authMiddleware verifica el token de autenticación en las solicitudes
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Obtener el token esperado desde variables de entorno
		expectedToken := os.Getenv("API_TOKEN")
		if expectedToken == "" {
			// Si no hay token configurado, permitir acceso (modo desarrollo)
			next.ServeHTTP(w, r)
			return
		}

		// Obtener token del header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Token de autorización requerido")
			return
		}

		// Verificar formato "Bearer token"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "Formato de token inválido. Use: Bearer <token>")
			return
		}

		// Verificar que el token coincida
		providedToken := parts[1]
		if providedToken != expectedToken {
			respondWithError(w, http.StatusUnauthorized, "Token de autorización inválido")
			return
		}

		// Token válido, continuar con la solicitud
		next.ServeHTTP(w, r)
	})
}

// respondWithError envía una respuesta de error en formato JSON
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	}
	json.NewEncoder(w).Encode(errorResponse)
}
