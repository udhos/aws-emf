// Package emf helps with AWS Embedded Metric Format.
package emf

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// Metric holds full EMF metric context.
type Metric struct {
	table   map[string]*metricContext // dimensions => context
	options Options
	lock    sync.Mutex
}

type metricContext struct {
	meta   *Metadata
	values map[string]any
}

// Metadata defines EMF Metadata.
type Metadata struct {
	CloudWatchMetrics []*MetricDirective `json:"CloudWatchMetrics"`
	Timestamp         int64              `json:"Timestamp"`
}

// MetricDirective defines EMF MetricDirective.
type MetricDirective struct {
	Namespace  string             `json:"Namespace"`
	Dimensions []DimensionSet     `json:"Dimensions"`
	Metrics    []MetricDefinition `json:"Metrics"`
}

// DimensionSet defines EMF DimensionSet.
type DimensionSet []string

// MetricDefinition defines EMF MetricDefinition.
type MetricDefinition struct {
	Name              string `json:"Name"`
	Unit              string `json:"Unit,omitempty"`
	StorageResolution int    `json:"StorageResolution,omitempty"`
}

// Options define options.
type Options struct {
	UnixMilli func() int64
}

// DefaultUnixMilli is default function used when Options.UnixMilli is left undefined.
func DefaultUnixMilli() int64 {
	return time.Now().UnixMilli()
}

// New creates EMF metric.
func New(options Options) *Metric {
	if options.UnixMilli == nil {
		options.UnixMilli = DefaultUnixMilli
	}
	m := &Metric{
		options: options,
	}
	m.Reset()
	return m
}

// Reset clears all defined metrics and dimensions.
// If in the previsou cycle you sent any metric what you won't update
// with Record() in the next cycle, use Reset() to clear all metrics
// before the next cycle. Otherwise those stale metrics will be sent again.
func (m *Metric) Reset() {
	m.lock.Lock()
	m.table = map[string]*metricContext{}
	m.lock.Unlock()
}

// Record records a metric.
func (m *Metric) Record(namespace string, metric MetricDefinition, dimensions map[string]string, value int) {
	m.lock.Lock()
	c := m.defineMetric(namespace, metric, dimensions)
	c.values[metric.Name] = value
	for k, v := range dimensions {
		c.values[k] = v
	}
	m.lock.Unlock()
}

func getDimensionKey(namespace string, dimensions map[string]string, dimSet DimensionSet) string {
	var list []string
	slices.Sort(dimSet)
	for _, k := range dimSet {
		v := dimensions[k]
		list = append(list, fmt.Sprintf("%s:%s", k, v))
	}
	return namespace + " " + strings.Join(list, ",")
}

func getDimensionSet(dimensions map[string]string) DimensionSet {
	return slices.Collect(maps.Keys(dimensions))
}

func (m *Metric) getContext(namespace string, dimensions map[string]string) (*metricContext, DimensionSet) {
	dimSet := getDimensionSet(dimensions)
	dimKey := getDimensionKey(namespace, dimensions, dimSet)
	c, foundContext := m.table[dimKey]
	if !foundContext {
		c = &metricContext{
			meta:   &Metadata{},
			values: map[string]any{},
		}
		m.table[dimKey] = c
	}
	return c, dimSet
}

// defineMetric defines a metric.
func (m *Metric) defineMetric(namespace string, metric MetricDefinition, dimensions map[string]string) *metricContext {
	//
	// get context
	//
	c, dimSet := m.getContext(namespace, dimensions)

	var dimSetList []DimensionSet
	if len(dimSet) == 0 {
		dimSetList = []DimensionSet{}
	} else {
		dimSetList = []DimensionSet{dimSet}
	}

	//
	// define metric
	//
	var directive *MetricDirective
	var foundDir bool
	for _, dir := range c.meta.CloudWatchMetrics {
		if namespace == dir.Namespace {
			directive = dir
			foundDir = true
			break
		}
	}
	if foundDir {
		var foundMetric bool
		var index int
		for i, md := range directive.Metrics {
			if md.Name == metric.Name {
				foundMetric = true
				index = i
				break
			}
		}
		if foundMetric {
			directive.Metrics[index] = metric
		} else {
			directive.Metrics = append(directive.Metrics, metric)
		}

		//
		// define dimensions
		//
		directive.Dimensions = dimSetList

	} else {
		dir := &MetricDirective{
			Namespace: namespace,
			Metrics:   []MetricDefinition{metric},
		}

		//
		// define dimensions
		//
		dir.Dimensions = dimSetList

		c.meta.CloudWatchMetrics = append(c.meta.CloudWatchMetrics, dir)
	}

	return c
}

func (m *Metric) count() (metrics, dimensions int) {
	for _, c := range m.table {
		for _, md := range c.meta.CloudWatchMetrics {
			metrics += len(md.Metrics)
			dimensions += len(md.Dimensions)
		}
	}
	return
}

// Render renders metrics as string.
func (m *Metric) Render() []string {
	t := m.options.UnixMilli()
	return m.renderWithTimestamp(t)
}

func (m *Metric) renderWithTimestamp(t int64) []string {
	m.lock.Lock()
	list := make([]string, 0, len(m.table))
	for _, c := range m.table {
		c.meta.Timestamp = t
		c.values["_aws"] = c.meta
		data, _ := json.Marshal(c.values)
		list = append(list, string(data))
	}
	m.lock.Unlock()
	return list
}

// Fprintln yields EMF metric to Writer.
func (m *Metric) Fprintln(w io.Writer) {
	for _, item := range m.Render() {
		fmt.Fprintln(w, item)
	}
}

// Println yields EMF metric to stdout.
func (m *Metric) Println() {
	m.Fprintln(os.Stdout)
}

// CloudWatchLogEvents yields EMF metric as input for cloudwatch log events.
func (m *Metric) CloudWatchLogEvents() []types.InputLogEvent {
	t := m.options.UnixMilli()
	list := m.renderWithTimestamp(t)
	var eventList []types.InputLogEvent
	for _, m := range list {
		eventList = append(eventList, types.InputLogEvent{
			Message:   aws.String(m),
			Timestamp: aws.Int64(t),
		})
	}
	return eventList
}

// CloudWatchString yields EMF metric as cloudwatch string for aws cli.
func (m *Metric) CloudWatchString() string {
	t := m.options.UnixMilli()
	list := m.renderWithTimestamp(t)
	var cwList []entry
	for _, m := range list {
		cwList = append(cwList, newEntry(m, t))
	}
	data, _ := json.Marshal(cwList)
	return string(data)
}

func newEntry(m string, t int64) entry {
	return entry{Message: m, Timestamp: t}
}

type entry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}
