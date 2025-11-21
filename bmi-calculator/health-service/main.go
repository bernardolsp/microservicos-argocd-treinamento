package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gorilla/mux"
)

type HealthStatus struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Timestamp   string            `json:"timestamp"`
	Version     string            `json:"version"`
	Uptime      string            `json:"uptime"`
	GoVersion   string            `json:"go_version"`
	Environment map[string]string `json:"environment"`
	System      SystemInfo        `json:"system"`
}

type SystemInfo struct {
	NumGoroutines int `json:"num_goroutines"`
	NumCPU        int `json:"num_cpu"`
}

type ServiceCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

var startTime = time.Now()

func main() {
	r := mux.NewRouter()

	r.Use(loggingMiddleware)

	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/health/detailed", detailedHealthHandler).Methods("GET")
	r.HandleFunc("/health/services", servicesHealthHandler).Methods("GET")
	r.HandleFunc("/ready", readinessHandler).Methods("GET")
	r.HandleFunc("/live", livenessHandler).Methods("GET")

	port := getEnv("PORT", "8082")
	log.Printf("Health Service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("Completed: %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:      "healthy",
		Service:     "health-service",
		Timestamp:   time.Now().Format(time.RFC3339),
		Version:     getEnv("IMAGE_VERSION", "unknown"),
		Uptime:      time.Since(startTime).String(),
		GoVersion:   runtime.Version(),
		Environment: getEnvironmentVars(),
		System: SystemInfo{
			NumGoroutines: runtime.NumGoroutine(),
			NumCPU:        runtime.NumCPU(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func detailedHealthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"service":   "health-service",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   getEnv("IMAGE_VERSION", "unknown"),
		"uptime":    time.Since(startTime).String(),
		"runtime": map[string]interface{}{
			"go_version":     runtime.Version(),
			"num_goroutines": runtime.NumGoroutine(),
			"num_cpu":        runtime.NumCPU(),
			"gomaxprocs":     runtime.GOMAXPROCS(0),
		},
		"memory": map[string]interface{}{
			"alloc":       getMemoryStats().Alloc,
			"total_alloc": getMemoryStats().TotalAlloc,
			"sys":         getMemoryStats().Sys,
		},
		"environment": getEnvironmentVars(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func servicesHealthHandler(w http.ResponseWriter, r *http.Request) {
	services := []ServiceCheck{
		{
			Name:   "gateway",
			Status: checkServiceHealth("http://gateway:8080/health"),
			URL:    "http://gateway:8080/health",
		},
		{
			Name:   "bmi-service",
			Status: checkServiceHealth("http://bmi-service:8081/health"),
			URL:    "http://bmi-service:8081/health",
		},
	}

	response := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  services,
		"overall":   getOverallStatus(services),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ready",
		"service": "health-service",
	})
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "alive",
		"service": "health-service",
	})
}

func checkServiceHealth(url string) string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "unhealthy"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "healthy"
	}
	return "unhealthy"
}

func getOverallStatus(services []ServiceCheck) string {
	for _, service := range services {
		if service.Status == "unhealthy" {
			return "degraded"
		}
	}
	return "healthy"
}

func getEnvironmentVars() map[string]string {
	env := make(map[string]string)
	relevantVars := []string{"PORT", "ENVIRONMENT", "NAMESPACE", "POD_NAME", "POD_IP", "IMAGE_VERSION"}

	for _, key := range relevantVars {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}

	return env
}

func getMemoryStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
