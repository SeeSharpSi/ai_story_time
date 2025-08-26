package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

// Metric represents a single metric measurement
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description,omitempty"`
}

// MetricsCollector collects and manages application metrics
type MetricsCollector struct {
	metrics     map[string]*Metric
	mutex       sync.RWMutex
	externalURL string
	client      *http.Client
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(externalURL string) *MetricsCollector {
	return &MetricsCollector{
		metrics:     make(map[string]*Metric),
		externalURL: externalURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// RecordCounter increments a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value float64, labels map[string]string, description string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.getMetricKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		metric.Value += value
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:        name,
			Type:        Counter,
			Value:       value,
			Labels:      labels,
			Timestamp:   time.Now(),
			Description: description,
		}
	}

	// Send to external database if configured
	if mc.externalURL != "" {
		go mc.sendToExternal(mc.metrics[key])
	}
}

// SetGauge sets a gauge metric to a specific value
func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string, description string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.getMetricKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:        name,
		Type:        Gauge,
		Value:       value,
		Labels:      labels,
		Timestamp:   time.Now(),
		Description: description,
	}

	// Send to external database if configured
	if mc.externalURL != "" {
		go mc.sendToExternal(mc.metrics[key])
	}
}

// RecordHistogram records a histogram observation
func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string, description string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.getMetricKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		// For histogram, we could implement buckets, but for simplicity we'll just track the latest value
		metric.Value = value
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:        name,
			Type:        Histogram,
			Value:       value,
			Labels:      labels,
			Timestamp:   time.Now(),
			Description: description,
		}
	}

	// Send to external database if configured
	if mc.externalURL != "" {
		go mc.sendToExternal(mc.metrics[key])
	}
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

// GetMetricsByName returns metrics filtered by name
func (mc *MetricsCollector) GetMetricsByName(name string) []*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	var result []*Metric
	for _, metric := range mc.metrics {
		if metric.Name == name {
			result = append(result, metric)
		}
	}
	return result
}

// getMetricKey generates a unique key for a metric based on name and labels
func (mc *MetricsCollector) getMetricKey(name string, labels map[string]string) string {
	key := name
	if labels != nil {
		for k, v := range labels {
			key += fmt.Sprintf("{%s=%s}", k, v)
		}
	}
	return key
}

// sendToExternal sends a metric to an external database
func (mc *MetricsCollector) sendToExternal(metric *Metric) {
	if mc.externalURL == "" {
		return
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		// In a production system, you'd want to log this error
		return
	}

	req, err := http.NewRequest("POST", mc.externalURL+"/metrics", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// In production, you might want to check resp.StatusCode and handle errors
}

// Global metrics collector instance
var defaultCollector *MetricsCollector

// InitDefaultCollector initializes the default metrics collector
func InitDefaultCollector(externalURL string) {
	defaultCollector = NewMetricsCollector(externalURL)
}

// RecordStoryGeneration records metrics for story generation
func RecordStoryGeneration(duration time.Duration, genre string, difficulty string, success bool) {
	if defaultCollector == nil {
		return
	}

	labels := map[string]string{
		"genre":      genre,
		"difficulty": difficulty,
		"success":    strconv.FormatBool(success),
	}

	defaultCollector.RecordCounter("story_generation_total", 1, labels, "Total number of story generation requests")
	defaultCollector.RecordHistogram("story_generation_duration", duration.Seconds(), labels, "Duration of story generation in seconds")

	if success {
		defaultCollector.RecordCounter("story_generation_success_total", 1, labels, "Total number of successful story generations")
	} else {
		defaultCollector.RecordCounter("story_generation_error_total", 1, labels, "Total number of failed story generations")
	}
}

// RecordAPIUsage records AI API usage metrics
func RecordAPIUsage(provider string, tokens int, duration time.Duration, success bool) {
	if defaultCollector == nil {
		return
	}

	labels := map[string]string{
		"provider": provider,
		"success":  strconv.FormatBool(success),
	}

	defaultCollector.RecordCounter("ai_api_requests_total", 1, labels, "Total number of AI API requests")
	defaultCollector.RecordHistogram("ai_api_duration", duration.Seconds(), labels, "Duration of AI API calls in seconds")
	defaultCollector.SetGauge("ai_api_tokens_used", float64(tokens), labels, "Number of tokens used in AI API call")

	if success {
		defaultCollector.RecordCounter("ai_api_success_total", 1, labels, "Total number of successful AI API calls")
	} else {
		defaultCollector.RecordCounter("ai_api_error_total", 1, labels, "Total number of failed AI API calls")
	}
}

// RecordUserActivity records user activity metrics
func RecordUserActivity(action string, genre string, sessionDuration time.Duration) {
	if defaultCollector == nil {
		return
	}

	labels := map[string]string{
		"action": action,
		"genre":  genre,
	}

	defaultCollector.RecordCounter("user_activity_total", 1, labels, "Total number of user actions")
	defaultCollector.RecordHistogram("user_session_duration", sessionDuration.Seconds(), labels, "Duration of user sessions in seconds")
}

// RecordError records application errors
func RecordError(errorType string, message string) {
	if defaultCollector == nil {
		return
	}

	labels := map[string]string{
		"error_type": errorType,
	}

	defaultCollector.RecordCounter("application_errors_total", 1, labels, "Total number of application errors")
}

// RecordRateLimit records rate limiting events
func RecordRateLimit(identifier string, blocked bool) {
	if defaultCollector == nil {
		return
	}

	labels := map[string]string{
		"identifier": identifier,
		"blocked":    strconv.FormatBool(blocked),
	}

	defaultCollector.RecordCounter("rate_limit_events_total", 1, labels, "Total number of rate limiting events")
}

// GetMetricsEndpoint returns an HTTP handler for the metrics endpoint
func GetMetricsEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if defaultCollector == nil {
			http.Error(w, "Metrics collector not initialized", http.StatusInternalServerError)
			return
		}

		metrics := defaultCollector.GetAllMetrics()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}
