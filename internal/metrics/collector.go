package metrics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"crystal/internal/interfaces"
)

// MetricsCollector interface for collecting metrics with zero-allocation design for performance monitoring
// Interface MetricsCollector untuk mengumpulkan metrik dengan desain zero-allocation untuk pemantauan kinerja
type MetricsCollector interface {
	// IncrementCounter increments a counter metric for the specified log level with associated tags
	// IncrementCounter menambahkan metrik counter untuk tingkat log yang ditentukan dengan tag terkait
	IncrementCounter(level interfaces.Level, tags map[string]string)
	
	// RecordHistogram records a histogram value for the specified metric with associated tags
	// RecordHistogram mencatat nilai histogram untuk metrik yang ditentukan dengan tag terkait
	RecordHistogram(metric string, value float64, tags map[string]string)
	
	// RecordGauge records a gauge value for the specified metric with associated tags
	// RecordGauge mencatat nilai gauge untuk metrik yang ditentukan dengan tag terkait
	RecordGauge(metric string, value float64, tags map[string]string)
}

// DefaultMetricsCollector is a simple in-memory metrics collector
type DefaultMetricsCollector struct {
	counters   map[string]int64
	histograms map[string][]float64
	gauges     map[string]float64
	mu         sync.RWMutex
}

// NewDefaultMetricsCollector creates a new DefaultMetricsCollector
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		counters:   make(map[string]int64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

// IncrementCounter increments a counter metric
func (d *DefaultMetricsCollector) IncrementCounter(level interfaces.Level, tags map[string]string) {
	key := fmt.Sprintf("log.%s", strings.ToLower(level.String()))
	d.mu.Lock()
	defer d.mu.Unlock()
	d.counters[key]++
}

// RecordHistogram records a histogram metric
func (d *DefaultMetricsCollector) RecordHistogram(metric string, value float64, tags map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.histograms[metric] = append(d.histograms[metric], value)
}

// RecordGauge records a gauge metric
func (d *DefaultMetricsCollector) RecordGauge(metric string, value float64, tags map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.gauges[metric] = value
}

// GetCounter returns the value of a counter metric
func (d *DefaultMetricsCollector) GetCounter(metric string) int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.counters[metric]
}

// GetAllCounters returns all counter metrics
func (d *DefaultMetricsCollector) GetAllCounters() map[string]int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	result := make(map[string]int64, len(d.counters))
	for k, v := range d.counters {
		result[k] = v
	}
	return result
}

// GetHistogram returns statistics for a histogram metric
func (d *DefaultMetricsCollector) GetHistogram(metric string) (min, max, avg, p95 float64) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	values := d.histograms[metric]
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	
	// Work with a copy to avoid race conditions during sorting
	valuesCopy := make([]float64, len(values))
	copy(valuesCopy, values)
	
	sort.Float64s(valuesCopy)
	min = valuesCopy[0]
	max = valuesCopy[len(valuesCopy)-1]
	sum := 0.0
	for _, v := range valuesCopy {
		sum += v
	}
	avg = sum / float64(len(valuesCopy))
	p95Index := int(math.Ceil(0.95 * float64(len(valuesCopy)))) - 1
	if p95Index < 0 {
		p95Index = 0
	}
	p95 = valuesCopy[p95Index]
	return min, max, avg, p95
}

// GetAllHistograms returns all histogram metrics
func (d *DefaultMetricsCollector) GetAllHistograms() map[string][]float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	result := make(map[string][]float64, len(d.histograms))
	for k, v := range d.histograms {
		valuesCopy := make([]float64, len(v))
		copy(valuesCopy, v)
		result[k] = valuesCopy
	}
	return result
}

// GetGauge returns the value of a gauge metric
func (d *DefaultMetricsCollector) GetGauge(metric string) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.gauges[metric]
}

// GetAllGauges returns all gauge metrics
func (d *DefaultMetricsCollector) GetAllGauges() map[string]float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	result := make(map[string]float64, len(d.gauges))
	for k, v := range d.gauges {
		result[k] = v
	}
	return result
}