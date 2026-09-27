package utils

import "strings"

const (
	APIBasePath   = "/api/v1"
	MetricsPath   = "/metrics"
	SwaggerPath   = "/swagger/*any"
	SwaggerPrefix = "/swagger/"
	HealthPath    = "/health"
	ReadyPath     = "/ready"
	DocsPath      = "/docs/*any"
)

var InfraSkipPaths = []string{
	MetricsPath,
	HealthPath,
	SwaggerPrefix,
	ReadyPath,
}

func IsInfraPath(path string) bool {
	for _, p := range InfraSkipPaths {
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(path, p) {
				return true
			}
			continue
		}
		if path == p || strings.HasSuffix(path, p) {
			return true
		}
	}
	return false
}
