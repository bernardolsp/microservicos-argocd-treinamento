package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type BMICalculation struct {
	Weight    float64 `json:"weight"`
	Height    float64 `json:"height"`
	BMI       float64 `json:"bmi"`
	Category  string  `json:"category"`
	Timestamp string  `json:"timestamp"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

var calculations []BMICalculation

func main() {
	r := mux.NewRouter()

	r.Use(loggingMiddleware)

	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/calculate", calculateHandler).Methods("POST")
	r.HandleFunc("/history", historyHandler).Methods("GET")
	r.HandleFunc("/bmi/{weight}/{height}", quickCalculateHandler).Methods("GET")

	port := getEnv("PORT", "8081")
	log.Printf("BMI Service starting on port %s", port)
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
	response := map[string]string{
		"status":        "healthy",
		"service":       "bmi-service",
		"timestamp":     time.Now().Format(time.RFC3339),
		"version":       "1.0.0",
		"image_version": getEnv("IMAGE_VERSION", "unknown"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Weight float64 `json:"weight"`
		Height float64 `json:"height"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Weight <= 0 || req.Height <= 0 {
		http.Error(w, "weight and height must be positive numbers", http.StatusBadRequest)
		return
	}

	bmi := req.Weight / (req.Height * req.Height)
	category := getBMICategory(bmi)

	calculation := BMICalculation{
		Weight:    req.Weight,
		Height:    req.Height,
		BMI:       bmi,
		Category:  category,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	calculations = append(calculations, calculation)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculation)
}

func quickCalculateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	weight, err := strconv.ParseFloat(vars["weight"], 64)
	if err != nil {
		http.Error(w, "invalid weight parameter", http.StatusBadRequest)
		return
	}

	height, err := strconv.ParseFloat(vars["height"], 64)
	if err != nil {
		http.Error(w, "invalid height parameter", http.StatusBadRequest)
		return
	}

	if weight <= 0 || height <= 0 {
		http.Error(w, "weight and height must be positive numbers", http.StatusBadRequest)
		return
	}

	bmi := weight / (height * height)
	category := getBMICategory(bmi)

	calculation := BMICalculation{
		Weight:    weight,
		Height:    height,
		BMI:       bmi,
		Category:  category,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	calculations = append(calculations, calculation)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculation)
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calculations": calculations,
		"count":        len(calculations),
	})
}

func getBMICategory(bmi float64) string {
	switch {
	case bmi < 18.5:
		return "Underweight"
	case bmi < 25:
		return "Normal weight"
	case bmi < 30:
		return "Overweight"
	default:
		return "Obese"
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
