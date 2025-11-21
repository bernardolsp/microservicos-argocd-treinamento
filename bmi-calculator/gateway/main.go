package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	bmiServiceURL := getEnv("BMI_SERVICE_URL", "http://bmi-service:8081")
	healthServiceURL := getEnv("HEALTH_SERVICE_URL", "http://health-service:8082")

	log.Printf("BMI Service URL: %s", bmiServiceURL)
	log.Printf("Health Service URL: %s", healthServiceURL)

	bmiProxy := createReverseProxy(bmiServiceURL)
	healthProxy := createReverseProxy(healthServiceURL)

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":        "healthy",
			"service":       "gateway",
			"image_version": getEnv("IMAGE_VERSION", "unknown"),
		})
	}).Methods("GET")

	r.PathPrefix("/api/health").Handler(loggingMiddleware(http.StripPrefix("/api", healthProxy)))

	r.PathPrefix("/api/bmi").Handler(loggingMiddleware(http.StripPrefix("/api/bmi", bmiProxy)))

	port := getEnv("PORT", "8080")
	log.Printf("Gateway starting on port %s", port)
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

func createReverseProxy(target string) *httputil.ReverseProxy {
	targetURL, _ := url.Parse(target)
	return httputil.NewSingleHostReverseProxy(targetURL)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
