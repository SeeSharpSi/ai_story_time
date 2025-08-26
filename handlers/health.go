package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string      `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
	Version   string      `json:"version"`
	Uptime    string      `json:"uptime"`
	Memory    MemoryStats `json:"memory"`
}

type MemoryStats struct {
	Alloc        uint64 `json:"alloc_bytes"`
	TotalAlloc   uint64 `json:"total_alloc_bytes"`
	Sys          uint64 `json:"sys_bytes"`
	Lookups      uint64 `json:"lookups"`
	Mallocs      uint64 `json:"mallocs"`
	Frees        uint64 `json:"frees"`
	HeapAlloc    uint64 `json:"heap_alloc_bytes"`
	HeapSys      uint64 `json:"heap_sys_bytes"`
	HeapIdle     uint64 `json:"heap_idle_bytes"`
	HeapInuse    uint64 `json:"heap_inuse_bytes"`
	HeapReleased uint64 `json:"heap_released_bytes"`
	HeapObjects  uint64 `json:"heap_objects"`
	StackInuse   uint64 `json:"stack_inuse_bytes"`
	StackSys     uint64 `json:"stack_sys_bytes"`
	GCSys        uint64 `json:"gc_sys_bytes"`
	NextGC       uint64 `json:"next_gc_bytes"`
	LastGC       uint64 `json:"last_gc_timestamp"`
	NumGC        uint32 `json:"num_gc"`
}

var startTime = time.Now()

// HealthCheckHandler provides a health check endpoint
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // You might want to make this configurable
		Uptime:    time.Since(startTime).String(),
		Memory: MemoryStats{
			Alloc:        m.Alloc,
			TotalAlloc:   m.TotalAlloc,
			Sys:          m.Sys,
			Lookups:      m.Lookups,
			Mallocs:      m.Mallocs,
			Frees:        m.Frees,
			HeapAlloc:    m.HeapAlloc,
			HeapSys:      m.HeapSys,
			HeapIdle:     m.HeapIdle,
			HeapInuse:    m.HeapInuse,
			HeapReleased: m.HeapReleased,
			HeapObjects:  m.HeapObjects,
			StackInuse:   m.StackInuse,
			StackSys:     m.StackSys,
			GCSys:        m.GCSys,
			NextGC:       m.NextGC,
			LastGC:       m.LastGC,
			NumGC:        m.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ReadinessHandler provides a readiness check endpoint
func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	// Add more comprehensive checks here in the future
	// For now, just check if the service has been running for at least 5 seconds
	if time.Since(startTime) < 5*time.Second {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not ready"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
