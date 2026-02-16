package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version  = getEnv("VERSION", "1.0")
	behavior = getEnv("BEHAVIOR", "normal") // normal, slow, error-prone, chaotic
	port     = getEnv("PORT", "8080")
	hostname = getHostname()

	// Prometheus metrics
	requestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "endpoint", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint"})

	versionGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_version_info",
		Help: "Application version information",
	}, []string{"version", "behavior", "hostname"})
)

type Response struct {
	Version   string            `json:"version"`
	Behavior  string            `json:"behavior"`
	Hostname  string            `json:"hostname"`
	Timestamp string            `json:"timestamp"`
	Message   string            `json:"message"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func main() {
	// Set version gauge
	versionGauge.WithLabelValues(version, behavior, hostname).Set(1)

	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Routes
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/data", handleAPIData)
	http.HandleFunc("/api/process", handleProcess)
	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Starting server - Version: %s, Behavior: %s, Port: %s\n", version, behavior, port)

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.Method, "/").Observe(duration)
	}()

	// Apply behavior
	status := applyBehavior(w, r)

	requestCounter.WithLabelValues(r.Method, "/", fmt.Sprintf("%d", status)).Inc()

	if status != http.StatusOK {
		http.Error(w, http.StatusText(status), status)
		return
	}

	response := Response{
		Version:   version,
		Behavior:  behavior,
		Hostname:  hostname,
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   getMessage(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.Method, "/health").Observe(duration)
	}()

	// Health check might fail in error-prone mode
	if behavior == "error-prone" && rand.Float32() < 0.3 {
		requestCounter.WithLabelValues(r.Method, "/health", "503").Inc()
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"reason": "simulated failure",
		})
		return
	}

	requestCounter.WithLabelValues(r.Method, "/health", "200").Inc()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "healthy",
		"version":  version,
		"hostname": hostname,
	})
}

func handleAPIData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.Method, "/api/data").Observe(duration)
	}()

	status := applyBehavior(w, r)
	requestCounter.WithLabelValues(r.Method, "/api/data", fmt.Sprintf("%d", status)).Inc()

	if status != http.StatusOK {
		http.Error(w, http.StatusText(status), status)
		return
	}

	// Simulate some data processing
	data := map[string]interface{}{
		"items":     rand.Intn(100),
		"processed": true,
		"version":   version,
		"hostname":  hostname,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.Method, "/api/process").Observe(duration)
	}()

	status := applyBehavior(w, r)
	requestCounter.WithLabelValues(r.Method, "/api/process", fmt.Sprintf("%d", status)).Inc()

	if status != http.StatusOK {
		http.Error(w, http.StatusText(status), status)
		return
	}

	// Simulate processing time
	if behavior == "slow" {
		time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)
	}

	response := map[string]interface{}{
		"status":   "completed",
		"duration": time.Since(start).Milliseconds(),
		"version":  version,
		"hostname": hostname,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func applyBehavior(w http.ResponseWriter, r *http.Request) int {
	switch behavior {
	case "normal":
		return http.StatusOK

	case "slow":
		// Add artificial delay
		delay := time.Duration(200+rand.Intn(800)) * time.Millisecond
		time.Sleep(delay)
		return http.StatusOK

	case "error-prone":
		// Return errors randomly (50% chance)
		if rand.Float32() < 0.5 {
			return http.StatusInternalServerError
		}
		return http.StatusOK

	case "chaotic":
		// Mix of slow and errors
		if rand.Float32() < 0.3 {
			time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)
		}
		if rand.Float32() < 0.4 {
			return http.StatusInternalServerError
		}
		return http.StatusOK

	default:
		return http.StatusOK
	}
}

func getMessage() string {
	messages := map[string][]string{
		"normal": {
			"Service operating normally",
			"All systems functional",
			"Request processed successfully",
		},
		"slow": {
			"Service is experiencing delays",
			"Processing taking longer than usual",
			"High latency detected",
		},
		"error-prone": {
			"Service unstable",
			"Errors may occur",
			"Degraded performance",
		},
		"chaotic": {
			"Unpredictable behavior",
			"System under stress",
			"Erratic performance",
		},
	}

	msgs := messages[behavior]
	if len(msgs) == 0 {
		return "Unknown state"
	}
	return msgs[rand.Intn(len(msgs))]
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
