// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
)

// AttributeBuildStatus specifies the a value build.status attribute.
type AttributeBuildStatus int

const (
	_ AttributeBuildStatus = iota
	AttributeBuildStatusPending
	AttributeBuildStatusRunning
	AttributeBuildStatusSuccess
	AttributeBuildStatusFailure
	AttributeBuildStatusSkipped
	AttributeBuildStatusError
	AttributeBuildStatusKilled
	AttributeBuildStatusBlocked
	AttributeBuildStatusPaused
	AttributeBuildStatusWaitingOnDependencies
	AttributeBuildStatusUnknown
)

// String returns the string representation of the AttributeBuildStatus.
func (av AttributeBuildStatus) String() string {
	switch av {
	case AttributeBuildStatusPending:
		return "pending"
	case AttributeBuildStatusRunning:
		return "running"
	case AttributeBuildStatusSuccess:
		return "success"
	case AttributeBuildStatusFailure:
		return "failure"
	case AttributeBuildStatusSkipped:
		return "skipped"
	case AttributeBuildStatusError:
		return "error"
	case AttributeBuildStatusKilled:
		return "killed"
	case AttributeBuildStatusBlocked:
		return "blocked"
	case AttributeBuildStatusPaused:
		return "paused"
	case AttributeBuildStatusWaitingOnDependencies:
		return "waiting_on_dependencies"
	case AttributeBuildStatusUnknown:
		return "unknown"
	}
	return ""
}

// MapAttributeBuildStatus is a helper map of string to AttributeBuildStatus attribute value.
var MapAttributeBuildStatus = map[string]AttributeBuildStatus{
	"pending":                 AttributeBuildStatusPending,
	"running":                 AttributeBuildStatusRunning,
	"success":                 AttributeBuildStatusSuccess,
	"failure":                 AttributeBuildStatusFailure,
	"skipped":                 AttributeBuildStatusSkipped,
	"error":                   AttributeBuildStatusError,
	"killed":                  AttributeBuildStatusKilled,
	"blocked":                 AttributeBuildStatusBlocked,
	"paused":                  AttributeBuildStatusPaused,
	"waiting_on_dependencies": AttributeBuildStatusWaitingOnDependencies,
	"unknown":                 AttributeBuildStatusUnknown,
}

type metricBuildsNumber struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills builds_number metric with initial data.
func (m *metricBuildsNumber) init() {
	m.data.SetName("builds_number")
	m.data.SetDescription("Number of builds.")
	m.data.SetUnit("1")
	m.data.SetEmptySum()
	m.data.Sum().SetIsMonotonic(false)
	m.data.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	m.data.Sum().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricBuildsNumber) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, buildStatusAttributeValue string, repoNameAttributeValue string, repoBranchAttributeValue string) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Sum().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("build.status", buildStatusAttributeValue)
	dp.Attributes().PutStr("repo.name", repoNameAttributeValue)
	dp.Attributes().PutStr("repo.branch", repoBranchAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricBuildsNumber) updateCapacity() {
	if m.data.Sum().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Sum().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricBuildsNumber) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Sum().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricBuildsNumber(cfg MetricConfig) metricBuildsNumber {
	m := metricBuildsNumber{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricRestartsTotal struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills restarts_total metric with initial data.
func (m *metricRestartsTotal) init() {
	m.data.SetName("restarts_total")
	m.data.SetDescription("Total number build restarts.")
	m.data.SetUnit("1")
	m.data.SetEmptySum()
	m.data.Sum().SetIsMonotonic(true)
	m.data.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
}

func (m *metricRestartsTotal) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Sum().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricRestartsTotal) updateCapacity() {
	if m.data.Sum().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Sum().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricRestartsTotal) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Sum().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricRestartsTotal(cfg MetricConfig) metricRestartsTotal {
	m := metricRestartsTotal{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

// MetricsBuilder provides an interface for scrapers to report metrics while taking care of all the transformations
// required to produce metric representation defined in metadata and user config.
type MetricsBuilder struct {
	startTime           pcommon.Timestamp   // start time that will be applied to all recorded data points.
	metricsCapacity     int                 // maximum observed number of metrics per resource.
	metricsBuffer       pmetric.Metrics     // accumulates metrics data before emitting.
	buildInfo           component.BuildInfo // contains version information
	metricBuildsNumber  metricBuildsNumber
	metricRestartsTotal metricRestartsTotal
}

// metricBuilderOption applies changes to default metrics builder.
type metricBuilderOption func(*MetricsBuilder)

// WithStartTime sets startTime on the metrics builder.
func WithStartTime(startTime pcommon.Timestamp) metricBuilderOption {
	return func(mb *MetricsBuilder) {
		mb.startTime = startTime
	}
}

func NewMetricsBuilder(mbc MetricsBuilderConfig, settings receiver.CreateSettings, options ...metricBuilderOption) *MetricsBuilder {
	mb := &MetricsBuilder{
		startTime:           pcommon.NewTimestampFromTime(time.Now()),
		metricsBuffer:       pmetric.NewMetrics(),
		buildInfo:           settings.BuildInfo,
		metricBuildsNumber:  newMetricBuildsNumber(mbc.Metrics.BuildsNumber),
		metricRestartsTotal: newMetricRestartsTotal(mbc.Metrics.RestartsTotal),
	}
	for _, op := range options {
		op(mb)
	}
	return mb
}

// updateCapacity updates max length of metrics and resource attributes that will be used for the slice capacity.
func (mb *MetricsBuilder) updateCapacity(rm pmetric.ResourceMetrics) {
	if mb.metricsCapacity < rm.ScopeMetrics().At(0).Metrics().Len() {
		mb.metricsCapacity = rm.ScopeMetrics().At(0).Metrics().Len()
	}
}

// ResourceMetricsOption applies changes to provided resource metrics.
type ResourceMetricsOption func(pmetric.ResourceMetrics)

// WithResource sets the provided resource on the emitted ResourceMetrics.
// It's recommended to use ResourceBuilder to create the resource.
func WithResource(res pcommon.Resource) ResourceMetricsOption {
	return func(rm pmetric.ResourceMetrics) {
		res.CopyTo(rm.Resource())
	}
}

// WithStartTimeOverride overrides start time for all the resource metrics data points.
// This option should be only used if different start time has to be set on metrics coming from different resources.
func WithStartTimeOverride(start pcommon.Timestamp) ResourceMetricsOption {
	return func(rm pmetric.ResourceMetrics) {
		var dps pmetric.NumberDataPointSlice
		metrics := rm.ScopeMetrics().At(0).Metrics()
		for i := 0; i < metrics.Len(); i++ {
			switch metrics.At(i).Type() {
			case pmetric.MetricTypeGauge:
				dps = metrics.At(i).Gauge().DataPoints()
			case pmetric.MetricTypeSum:
				dps = metrics.At(i).Sum().DataPoints()
			}
			for j := 0; j < dps.Len(); j++ {
				dps.At(j).SetStartTimestamp(start)
			}
		}
	}
}

// EmitForResource saves all the generated metrics under a new resource and updates the internal state to be ready for
// recording another set of data points as part of another resource. This function can be helpful when one scraper
// needs to emit metrics from several resources. Otherwise calling this function is not required,
// just `Emit` function can be called instead.
// Resource attributes should be provided as ResourceMetricsOption arguments.
func (mb *MetricsBuilder) EmitForResource(rmo ...ResourceMetricsOption) {
	rm := pmetric.NewResourceMetrics()
	rm.SetSchemaUrl(conventions.SchemaURL)
	ils := rm.ScopeMetrics().AppendEmpty()
	ils.Scope().SetName("otelcol/dronereceiver")
	ils.Scope().SetVersion(mb.buildInfo.Version)
	ils.Metrics().EnsureCapacity(mb.metricsCapacity)
	mb.metricBuildsNumber.emit(ils.Metrics())
	mb.metricRestartsTotal.emit(ils.Metrics())

	for _, op := range rmo {
		op(rm)
	}
	if ils.Metrics().Len() > 0 {
		mb.updateCapacity(rm)
		rm.MoveTo(mb.metricsBuffer.ResourceMetrics().AppendEmpty())
	}
}

// Emit returns all the metrics accumulated by the metrics builder and updates the internal state to be ready for
// recording another set of metrics. This function will be responsible for applying all the transformations required to
// produce metric representation defined in metadata and user config, e.g. delta or cumulative.
func (mb *MetricsBuilder) Emit(rmo ...ResourceMetricsOption) pmetric.Metrics {
	mb.EmitForResource(rmo...)
	metrics := mb.metricsBuffer
	mb.metricsBuffer = pmetric.NewMetrics()
	return metrics
}

// RecordBuildsNumberDataPoint adds a data point to builds_number metric.
func (mb *MetricsBuilder) RecordBuildsNumberDataPoint(ts pcommon.Timestamp, val int64, buildStatusAttributeValue AttributeBuildStatus, repoNameAttributeValue string, repoBranchAttributeValue string) {
	mb.metricBuildsNumber.recordDataPoint(mb.startTime, ts, val, buildStatusAttributeValue.String(), repoNameAttributeValue, repoBranchAttributeValue)
}

// RecordRestartsTotalDataPoint adds a data point to restarts_total metric.
func (mb *MetricsBuilder) RecordRestartsTotalDataPoint(ts pcommon.Timestamp, val int64) {
	mb.metricRestartsTotal.recordDataPoint(mb.startTime, ts, val)
}

// Reset resets metrics builder to its initial state. It should be used when external metrics source is restarted,
// and metrics builder should update its startTime and reset it's internal state accordingly.
func (mb *MetricsBuilder) Reset(options ...metricBuilderOption) {
	mb.startTime = pcommon.NewTimestampFromTime(time.Now())
	for _, op := range options {
		op(mb)
	}
}
