package images

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ImageParams contiene los parámetros para optimización de imagen
type ImageParams struct {
	URL     string
	Quality int
	Width   int
	Origin  string
	Referer string
}

// ParamsParser maneja el parsing de parámetros de URL
type ParamsParser struct{}

// NewParamsParser crea una nueva instancia del parser
func NewParamsParser() *ParamsParser {
	return &ParamsParser{}
}

// ParseURLParams extrae los parámetros de la URL
// Formato esperado: /w_400,q_90/url?origin="dominio.com"
func (pp *ParamsParser) ParseURLParams(r *http.Request) (*ImageParams, error) {
	// Obtener el path completo
	fullPath := chi.URLParam(r, "*")
	if fullPath == "" {
		return nil, fmt.Errorf("no path provided")
	}

	// Regex mejorada para parsear el formato /w_400,q_90,r_domain.com/url
	re := regexp.MustCompile(`^/?(?:w_(\d+))?(?:,q_(\d+))?(?:,r_([^/]+))?/?(.+)$`)
	matches := re.FindStringSubmatch(fullPath)
	
	if len(matches) < 5 {
		return nil, fmt.Errorf("invalid URL format. Expected: /w_400,q_90,r_domain.com/url or /w_400/url or /q_90/url")
	}

	params := &ImageParams{
		Width:   400, // default
		Quality: 90, // default
	}

	// Parsear width
	if matches[1] != "" {
		if w, err := strconv.Atoi(matches[1]); err == nil {
			if err := pp.validateWidth(w); err != nil {
				return nil, err
			}
			params.Width = w
		}
	}

	// Parsear quality
	if matches[2] != "" {
		if q, err := strconv.Atoi(matches[2]); err == nil {
			if err := pp.validateQuality(q); err != nil {
				return nil, err
			}
			params.Quality = q
		}
	}

	// Parsear referer
	if matches[3] != "" {
		params.Referer = matches[3]
	}

	// Parsear y validar URL
	imageURL, err := pp.parseImageURL(matches[4])
	if err != nil {
		return nil, err
	}
	params.URL = imageURL

	// Obtener origin de query parameters
	if origin := r.URL.Query().Get("origin"); origin != "" {
		params.Origin = strings.Trim(origin, `"`)
	}

	return params, nil
}

// parseImageURL procesa y valida la URL de la imagen
func (pp *ParamsParser) parseImageURL(rawURL string) (string, error) {
	// Decodificar URL encoding
	decodedURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to decode URL: %v", err)
	}

	// Agregar protocolo si no existe
	if !strings.HasPrefix(decodedURL, "http://") && !strings.HasPrefix(decodedURL, "https://") {
		decodedURL = "https://" + decodedURL
	}

	// Validar URL
	parsedURL, err := url.Parse(decodedURL)
	if err != nil {
		return "", fmt.Errorf("invalid image URL: %v", err)
	}

	// Verificar que tenga host
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid image URL: missing host")
	}

	return parsedURL.String(), nil
}

// validateWidth valida el parámetro de ancho
func (pp *ParamsParser) validateWidth(width int) error {
	if width <= 0 {
		return fmt.Errorf("width must be greater than 0")
	}
	if width > 2000 {
		return fmt.Errorf("width cannot exceed 2000 pixels")
	}
	return nil
}

// validateQuality valida el parámetro de calidad
func (pp *ParamsParser) validateQuality(quality int) error {
	if quality <= 0 {
		return fmt.Errorf("quality must be greater than 0")
	}
	if quality > 100 {
		return fmt.Errorf("quality cannot exceed 100")
	}
	return nil
}

// GetDefaultParams retorna parámetros por defecto
func (pp *ParamsParser) GetDefaultParams() *ImageParams {
	return &ImageParams{
		Width:   400,
		Quality: 90,
		URL:     "",
		Origin:  "",
		Referer: "",
	}
}