// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

// AttributeCiGithubWorkflowJobConclusion specifies the value ci.github.workflow.job.conclusion attribute.
type AttributeCiGithubWorkflowJobConclusion int

const (
	_ AttributeCiGithubWorkflowJobConclusion = iota
	AttributeCiGithubWorkflowJobConclusionSuccess
	AttributeCiGithubWorkflowJobConclusionFailure
	AttributeCiGithubWorkflowJobConclusionCancelled
	AttributeCiGithubWorkflowJobConclusionNeutral
	AttributeCiGithubWorkflowJobConclusionNull
	AttributeCiGithubWorkflowJobConclusionSkipped
	AttributeCiGithubWorkflowJobConclusionTimedOut
	AttributeCiGithubWorkflowJobConclusionActionRequired
)

// String returns the string representation of the AttributeCiGithubWorkflowJobConclusion.
func (av AttributeCiGithubWorkflowJobConclusion) String() string {
	switch av {
	case AttributeCiGithubWorkflowJobConclusionSuccess:
		return "success"
	case AttributeCiGithubWorkflowJobConclusionFailure:
		return "failure"
	case AttributeCiGithubWorkflowJobConclusionCancelled:
		return "cancelled"
	case AttributeCiGithubWorkflowJobConclusionNeutral:
		return "neutral"
	case AttributeCiGithubWorkflowJobConclusionNull:
		return "null"
	case AttributeCiGithubWorkflowJobConclusionSkipped:
		return "skipped"
	case AttributeCiGithubWorkflowJobConclusionTimedOut:
		return "timed_out"
	case AttributeCiGithubWorkflowJobConclusionActionRequired:
		return "action_required"
	}
	return ""
}

// MapAttributeCiGithubWorkflowJobConclusion is a helper map of string to AttributeCiGithubWorkflowJobConclusion attribute value.
var MapAttributeCiGithubWorkflowJobConclusion = map[string]AttributeCiGithubWorkflowJobConclusion{
	"success":         AttributeCiGithubWorkflowJobConclusionSuccess,
	"failure":         AttributeCiGithubWorkflowJobConclusionFailure,
	"cancelled":       AttributeCiGithubWorkflowJobConclusionCancelled,
	"neutral":         AttributeCiGithubWorkflowJobConclusionNeutral,
	"null":            AttributeCiGithubWorkflowJobConclusionNull,
	"skipped":         AttributeCiGithubWorkflowJobConclusionSkipped,
	"timed_out":       AttributeCiGithubWorkflowJobConclusionTimedOut,
	"action_required": AttributeCiGithubWorkflowJobConclusionActionRequired,
}

// AttributeCiGithubWorkflowJobStatus specifies the value ci.github.workflow.job.status attribute.
type AttributeCiGithubWorkflowJobStatus int

const (
	_ AttributeCiGithubWorkflowJobStatus = iota
	AttributeCiGithubWorkflowJobStatusCompleted
	AttributeCiGithubWorkflowJobStatusInProgress
	AttributeCiGithubWorkflowJobStatusQueued
	AttributeCiGithubWorkflowJobStatusWaiting
	AttributeCiGithubWorkflowJobStatusAborted
)

// String returns the string representation of the AttributeCiGithubWorkflowJobStatus.
func (av AttributeCiGithubWorkflowJobStatus) String() string {
	switch av {
	case AttributeCiGithubWorkflowJobStatusCompleted:
		return "completed"
	case AttributeCiGithubWorkflowJobStatusInProgress:
		return "in_progress"
	case AttributeCiGithubWorkflowJobStatusQueued:
		return "queued"
	case AttributeCiGithubWorkflowJobStatusWaiting:
		return "waiting"
	case AttributeCiGithubWorkflowJobStatusAborted:
		return "aborted"
	}
	return ""
}

// MapAttributeCiGithubWorkflowJobStatus is a helper map of string to AttributeCiGithubWorkflowJobStatus attribute value.
var MapAttributeCiGithubWorkflowJobStatus = map[string]AttributeCiGithubWorkflowJobStatus{
	"completed":   AttributeCiGithubWorkflowJobStatusCompleted,
	"in_progress": AttributeCiGithubWorkflowJobStatusInProgress,
	"queued":      AttributeCiGithubWorkflowJobStatusQueued,
	"waiting":     AttributeCiGithubWorkflowJobStatusWaiting,
	"aborted":     AttributeCiGithubWorkflowJobStatusAborted,
}

// AttributeCiGithubWorkflowRunConclusion specifies the value ci.github.workflow.run.conclusion attribute.
type AttributeCiGithubWorkflowRunConclusion int

const (
	_ AttributeCiGithubWorkflowRunConclusion = iota
	AttributeCiGithubWorkflowRunConclusionSuccess
	AttributeCiGithubWorkflowRunConclusionFailure
	AttributeCiGithubWorkflowRunConclusionCancelled
	AttributeCiGithubWorkflowRunConclusionNeutral
	AttributeCiGithubWorkflowRunConclusionNull
	AttributeCiGithubWorkflowRunConclusionSkipped
	AttributeCiGithubWorkflowRunConclusionTimedOut
	AttributeCiGithubWorkflowRunConclusionActionRequired
)

// String returns the string representation of the AttributeCiGithubWorkflowRunConclusion.
func (av AttributeCiGithubWorkflowRunConclusion) String() string {
	switch av {
	case AttributeCiGithubWorkflowRunConclusionSuccess:
		return "success"
	case AttributeCiGithubWorkflowRunConclusionFailure:
		return "failure"
	case AttributeCiGithubWorkflowRunConclusionCancelled:
		return "cancelled"
	case AttributeCiGithubWorkflowRunConclusionNeutral:
		return "neutral"
	case AttributeCiGithubWorkflowRunConclusionNull:
		return "null"
	case AttributeCiGithubWorkflowRunConclusionSkipped:
		return "skipped"
	case AttributeCiGithubWorkflowRunConclusionTimedOut:
		return "timed_out"
	case AttributeCiGithubWorkflowRunConclusionActionRequired:
		return "action_required"
	}
	return ""
}

// MapAttributeCiGithubWorkflowRunConclusion is a helper map of string to AttributeCiGithubWorkflowRunConclusion attribute value.
var MapAttributeCiGithubWorkflowRunConclusion = map[string]AttributeCiGithubWorkflowRunConclusion{
	"success":         AttributeCiGithubWorkflowRunConclusionSuccess,
	"failure":         AttributeCiGithubWorkflowRunConclusionFailure,
	"cancelled":       AttributeCiGithubWorkflowRunConclusionCancelled,
	"neutral":         AttributeCiGithubWorkflowRunConclusionNeutral,
	"null":            AttributeCiGithubWorkflowRunConclusionNull,
	"skipped":         AttributeCiGithubWorkflowRunConclusionSkipped,
	"timed_out":       AttributeCiGithubWorkflowRunConclusionTimedOut,
	"action_required": AttributeCiGithubWorkflowRunConclusionActionRequired,
}

// AttributeCiGithubWorkflowRunStatus specifies the value ci.github.workflow.run.status attribute.
type AttributeCiGithubWorkflowRunStatus int

const (
	_ AttributeCiGithubWorkflowRunStatus = iota
	AttributeCiGithubWorkflowRunStatusCompleted
	AttributeCiGithubWorkflowRunStatusInProgress
	AttributeCiGithubWorkflowRunStatusQueued
	AttributeCiGithubWorkflowRunStatusWaiting
	AttributeCiGithubWorkflowRunStatusAborted
)

// String returns the string representation of the AttributeCiGithubWorkflowRunStatus.
func (av AttributeCiGithubWorkflowRunStatus) String() string {
	switch av {
	case AttributeCiGithubWorkflowRunStatusCompleted:
		return "completed"
	case AttributeCiGithubWorkflowRunStatusInProgress:
		return "in_progress"
	case AttributeCiGithubWorkflowRunStatusQueued:
		return "queued"
	case AttributeCiGithubWorkflowRunStatusWaiting:
		return "waiting"
	case AttributeCiGithubWorkflowRunStatusAborted:
		return "aborted"
	}
	return ""
}

// MapAttributeCiGithubWorkflowRunStatus is a helper map of string to AttributeCiGithubWorkflowRunStatus attribute value.
var MapAttributeCiGithubWorkflowRunStatus = map[string]AttributeCiGithubWorkflowRunStatus{
	"completed":   AttributeCiGithubWorkflowRunStatusCompleted,
	"in_progress": AttributeCiGithubWorkflowRunStatusInProgress,
	"queued":      AttributeCiGithubWorkflowRunStatusQueued,
	"waiting":     AttributeCiGithubWorkflowRunStatusWaiting,
	"aborted":     AttributeCiGithubWorkflowRunStatusAborted,
}

type metricWorkflowJobsCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills workflow.jobs.count metric with initial data.
func (m *metricWorkflowJobsCount) init() {
	m.data.SetName("workflow.jobs.count")
	m.data.SetDescription("Number of jobs.")
	m.data.SetUnit("{job}")
	m.data.SetEmptySum()
	m.data.Sum().SetIsMonotonic(true)
	m.data.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	m.data.Sum().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricWorkflowJobsCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryNameAttributeValue string, ciGithubWorkflowJobLabelsAttributeValue string, ciGithubWorkflowJobStatusAttributeValue string, ciGithubWorkflowJobConclusionAttributeValue string, ciGithubWorkflowJobHeadBranchIsMainAttributeValue bool) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Sum().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("ci.github.workflow.job.labels", ciGithubWorkflowJobLabelsAttributeValue)
	dp.Attributes().PutStr("ci.github.workflow.job.status", ciGithubWorkflowJobStatusAttributeValue)
	dp.Attributes().PutStr("ci.github.workflow.job.conclusion", ciGithubWorkflowJobConclusionAttributeValue)
	dp.Attributes().PutBool("ci.github.workflow.job.head_branch.is_main", ciGithubWorkflowJobHeadBranchIsMainAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricWorkflowJobsCount) updateCapacity() {
	if m.data.Sum().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Sum().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricWorkflowJobsCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Sum().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricWorkflowJobsCount(cfg MetricConfig) metricWorkflowJobsCount {
	m := metricWorkflowJobsCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

type metricWorkflowRunsCount struct {
	data     pmetric.Metric // data buffer for generated metric.
	config   MetricConfig   // metric config provided by user.
	capacity int            // max observed number of data points added to the metric.
}

// init fills workflow.runs.count metric with initial data.
func (m *metricWorkflowRunsCount) init() {
	m.data.SetName("workflow.runs.count")
	m.data.SetDescription("Number of runs.")
	m.data.SetUnit("{run}")
	m.data.SetEmptySum()
	m.data.Sum().SetIsMonotonic(true)
	m.data.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	m.data.Sum().DataPoints().EnsureCapacity(m.capacity)
}

func (m *metricWorkflowRunsCount) recordDataPoint(start pcommon.Timestamp, ts pcommon.Timestamp, val int64, vcsRepositoryNameAttributeValue string, ciGithubWorkflowRunLabelsAttributeValue string, ciGithubWorkflowRunStatusAttributeValue string, ciGithubWorkflowRunConclusionAttributeValue string, ciGithubWorkflowRunHeadBranchIsMainAttributeValue bool) {
	if !m.config.Enabled {
		return
	}
	dp := m.data.Sum().DataPoints().AppendEmpty()
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(ts)
	dp.SetIntValue(val)
	dp.Attributes().PutStr("vcs.repository.name", vcsRepositoryNameAttributeValue)
	dp.Attributes().PutStr("ci.github.workflow.run.labels", ciGithubWorkflowRunLabelsAttributeValue)
	dp.Attributes().PutStr("ci.github.workflow.run.status", ciGithubWorkflowRunStatusAttributeValue)
	dp.Attributes().PutStr("ci.github.workflow.run.conclusion", ciGithubWorkflowRunConclusionAttributeValue)
	dp.Attributes().PutBool("ci.github.workflow.run.head_branch.is_main", ciGithubWorkflowRunHeadBranchIsMainAttributeValue)
}

// updateCapacity saves max length of data point slices that will be used for the slice capacity.
func (m *metricWorkflowRunsCount) updateCapacity() {
	if m.data.Sum().DataPoints().Len() > m.capacity {
		m.capacity = m.data.Sum().DataPoints().Len()
	}
}

// emit appends recorded metric data to a metrics slice and prepares it for recording another set of data points.
func (m *metricWorkflowRunsCount) emit(metrics pmetric.MetricSlice) {
	if m.config.Enabled && m.data.Sum().DataPoints().Len() > 0 {
		m.updateCapacity()
		m.data.MoveTo(metrics.AppendEmpty())
		m.init()
	}
}

func newMetricWorkflowRunsCount(cfg MetricConfig) metricWorkflowRunsCount {
	m := metricWorkflowRunsCount{config: cfg}
	if cfg.Enabled {
		m.data = pmetric.NewMetric()
		m.init()
	}
	return m
}

// MetricsBuilder provides an interface for scrapers to report metrics while taking care of all the transformations
// required to produce metric representation defined in metadata and user config.
type MetricsBuilder struct {
	config                  MetricsBuilderConfig // config of the metrics builder.
	startTime               pcommon.Timestamp    // start time that will be applied to all recorded data points.
	metricsCapacity         int                  // maximum observed number of metrics per resource.
	metricsBuffer           pmetric.Metrics      // accumulates metrics data before emitting.
	buildInfo               component.BuildInfo  // contains version information.
	metricWorkflowJobsCount metricWorkflowJobsCount
	metricWorkflowRunsCount metricWorkflowRunsCount
}

// MetricBuilderOption applies changes to default metrics builder.
type MetricBuilderOption interface {
	apply(*MetricsBuilder)
}

type metricBuilderOptionFunc func(mb *MetricsBuilder)

func (mbof metricBuilderOptionFunc) apply(mb *MetricsBuilder) {
	mbof(mb)
}

// WithStartTime sets startTime on the metrics builder.
func WithStartTime(startTime pcommon.Timestamp) MetricBuilderOption {
	return metricBuilderOptionFunc(func(mb *MetricsBuilder) {
		mb.startTime = startTime
	})
}
func NewMetricsBuilder(mbc MetricsBuilderConfig, settings receiver.Settings, options ...MetricBuilderOption) *MetricsBuilder {
	mb := &MetricsBuilder{
		config:                  mbc,
		startTime:               pcommon.NewTimestampFromTime(time.Now()),
		metricsBuffer:           pmetric.NewMetrics(),
		buildInfo:               settings.BuildInfo,
		metricWorkflowJobsCount: newMetricWorkflowJobsCount(mbc.Metrics.WorkflowJobsCount),
		metricWorkflowRunsCount: newMetricWorkflowRunsCount(mbc.Metrics.WorkflowRunsCount),
	}

	for _, op := range options {
		op.apply(mb)
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
type ResourceMetricsOption interface {
	apply(pmetric.ResourceMetrics)
}

type resourceMetricsOptionFunc func(pmetric.ResourceMetrics)

func (rmof resourceMetricsOptionFunc) apply(rm pmetric.ResourceMetrics) {
	rmof(rm)
}

// WithResource sets the provided resource on the emitted ResourceMetrics.
// It's recommended to use ResourceBuilder to create the resource.
func WithResource(res pcommon.Resource) ResourceMetricsOption {
	return resourceMetricsOptionFunc(func(rm pmetric.ResourceMetrics) {
		res.CopyTo(rm.Resource())
	})
}

// WithStartTimeOverride overrides start time for all the resource metrics data points.
// This option should be only used if different start time has to be set on metrics coming from different resources.
func WithStartTimeOverride(start pcommon.Timestamp) ResourceMetricsOption {
	return resourceMetricsOptionFunc(func(rm pmetric.ResourceMetrics) {
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
	})
}

// EmitForResource saves all the generated metrics under a new resource and updates the internal state to be ready for
// recording another set of data points as part of another resource. This function can be helpful when one scraper
// needs to emit metrics from several resources. Otherwise calling this function is not required,
// just `Emit` function can be called instead.
// Resource attributes should be provided as ResourceMetricsOption arguments.
func (mb *MetricsBuilder) EmitForResource(options ...ResourceMetricsOption) {
	rm := pmetric.NewResourceMetrics()
	ils := rm.ScopeMetrics().AppendEmpty()
	ils.Scope().SetName("github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver")
	ils.Scope().SetVersion(mb.buildInfo.Version)
	ils.Metrics().EnsureCapacity(mb.metricsCapacity)
	mb.metricWorkflowJobsCount.emit(ils.Metrics())
	mb.metricWorkflowRunsCount.emit(ils.Metrics())

	for _, op := range options {
		op.apply(rm)
	}

	if ils.Metrics().Len() > 0 {
		mb.updateCapacity(rm)
		rm.MoveTo(mb.metricsBuffer.ResourceMetrics().AppendEmpty())
	}
}

// Emit returns all the metrics accumulated by the metrics builder and updates the internal state to be ready for
// recording another set of metrics. This function will be responsible for applying all the transformations required to
// produce metric representation defined in metadata and user config, e.g. delta or cumulative.
func (mb *MetricsBuilder) Emit(options ...ResourceMetricsOption) pmetric.Metrics {
	mb.EmitForResource(options...)
	metrics := mb.metricsBuffer
	mb.metricsBuffer = pmetric.NewMetrics()
	return metrics
}

// RecordWorkflowJobsCountDataPoint adds a data point to workflow.jobs.count metric.
func (mb *MetricsBuilder) RecordWorkflowJobsCountDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryNameAttributeValue string, ciGithubWorkflowJobLabelsAttributeValue string, ciGithubWorkflowJobStatusAttributeValue AttributeCiGithubWorkflowJobStatus, ciGithubWorkflowJobConclusionAttributeValue AttributeCiGithubWorkflowJobConclusion, ciGithubWorkflowJobHeadBranchIsMainAttributeValue bool) {
	mb.metricWorkflowJobsCount.recordDataPoint(mb.startTime, ts, val, vcsRepositoryNameAttributeValue, ciGithubWorkflowJobLabelsAttributeValue, ciGithubWorkflowJobStatusAttributeValue.String(), ciGithubWorkflowJobConclusionAttributeValue.String(), ciGithubWorkflowJobHeadBranchIsMainAttributeValue)
}

// RecordWorkflowRunsCountDataPoint adds a data point to workflow.runs.count metric.
func (mb *MetricsBuilder) RecordWorkflowRunsCountDataPoint(ts pcommon.Timestamp, val int64, vcsRepositoryNameAttributeValue string, ciGithubWorkflowRunLabelsAttributeValue string, ciGithubWorkflowRunStatusAttributeValue AttributeCiGithubWorkflowRunStatus, ciGithubWorkflowRunConclusionAttributeValue AttributeCiGithubWorkflowRunConclusion, ciGithubWorkflowRunHeadBranchIsMainAttributeValue bool) {
	mb.metricWorkflowRunsCount.recordDataPoint(mb.startTime, ts, val, vcsRepositoryNameAttributeValue, ciGithubWorkflowRunLabelsAttributeValue, ciGithubWorkflowRunStatusAttributeValue.String(), ciGithubWorkflowRunConclusionAttributeValue.String(), ciGithubWorkflowRunHeadBranchIsMainAttributeValue)
}

// Reset resets metrics builder to its initial state. It should be used when external metrics source is restarted,
// and metrics builder should update its startTime and reset it's internal state accordingly.
func (mb *MetricsBuilder) Reset(options ...MetricBuilderOption) {
	mb.startTime = pcommon.NewTimestampFromTime(time.Now())
	for _, op := range options {
		op.apply(mb)
	}
}
