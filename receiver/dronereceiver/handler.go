package dronereceiver

import (
	"slices"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/grafana/grafana-ci-otel-collector/internal/semconv"
	"github.com/grafana/grafana-ci-otel-collector/internal/traceutils"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
	"go.uber.org/zap"
)

type RepoEvt struct {
	drone.Repo
	Build *drone.Build `json:"build"`
}

type WebhookEvent struct {
	Action       string   `json:"action"`
	Repo         *RepoEvt `json:"repo"`
	drone.System `json:"system"`
}

func handleEvent(evt WebhookEvent, config *Config, droneClient drone.Client, logger *zap.Logger) (*ptrace.Traces, *plog.Logs) {
	repo := evt.Repo
	build := evt.Repo.Build
	logger.Debug("Got request")

	// Skip unfinished builds (i.e. builds that are still running)
	// In theory, according to the docs in https://docs.drone.io/webhooks/examples/, build.Action should be "completed" when a build is completed.
	// However, in practice, it seems that build.Action is always "updated" as per https://github.com/harness/drone/issues/2977.
	// so, we check if build.Finished is set to a non-zero value to determine if the build is finished.
	// However this appears to be sent when a build completes; The structure however changes slightly as in
	// the `repo.build` seems to be absent in the first one.
	// TODO: Revisit this, we may not need the Finished check.
	if build == nil {
		logger.Warn("no build info provided from the webhook event")
		return nil, nil
	}

	if build.Finished == 0 {
		logger.Debug("build hasn't finished yet")
		return nil, nil
	}

	// Skip traces for repos that are not enabled
	allowedBranches, ok := config.ReposConfig[evt.Repo.Slug]
	if !ok {
		logger.Warn("repo not enabled, skipping", zap.String("repo", evt.Repo.Slug))
		return nil, nil
	}

	// Skip traces for branches that are not configured
	if !slices.Contains(allowedBranches, repo.Branch) {
		logger.Warn("branch not enabled, skipping", zap.String("branch", repo.Branch))
		return nil, nil
	}

	traces := ptrace.NewTraces()
	logs := plog.NewLogs()

	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()

	scopeSpans.Scope().SetName("dronereceiver")
	scopeSpans.Scope().SetVersion("0.1.0")

	resourceAttrs := resourceSpans.Resource().Attributes()
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, "drone")
	resourceAttrs.PutStr(semconv.AttributeGitRepoName, repo.Slug)
	resourceAttrs.PutStr(semconv.AttributeGitBranchName, repo.Branch)

	buildSpan := scopeSpans.Spans().AppendEmpty()
	buildSpan.SetTraceID(traceutils.NewTraceID())
	buildSpan.SetSpanID(traceutils.NewSpanID())
	buildSpan.SetParentSpanID(pcommon.NewSpanIDEmpty())
	buildAttributes := buildSpan.Attributes()

	buildAttributes.PutStr(semconv.AttributeDroneWorkflowItemKind, semconv.AttributeDroneWorkflowItemKindBuild)

	buildAttributes.PutStr(semconv.AttributeDroneWorkflowEvent, build.Event)

	buildAttributes.PutInt(semconv.AttributeDroneBuildNumber, build.Number)
	buildAttributes.PutInt(semconv.AttributeDroneBuildID, build.ID)

	// Set build title and message
	// The root span name will be the build title if it is set, otherwise it will be the build message.
	buildAttributes.PutStr(semconv.AttributeDroneWorkflowTitle, build.Title)
	buildAttributes.PutStr(semconv.AttributeDroneBuildMessage, build.Message)

	traceutils.SetStatus(build.Status, buildSpan)

	buildSpan.SetStartTimestamp(pcommon.Timestamp(build.Created * 1000000000))
	buildSpan.SetEndTimestamp(pcommon.Timestamp(build.Finished * 1000000000))

	buildAttributes.PutStr(semconv.AttributeCIVendor, semconv.AttributeCIVendorDrone)
	buildAttributes.PutStr(semconv.AttributeCIVersion, evt.System.Version)

	// --- VCS Info
	// !FIXME: the scm property seems to always be empty, we fallback to GIT for now
	vcsType := semconv.AttributeVCSTypeGit
	if repo.SCM != "" {
		vcsType = repo.SCM
	}
	buildAttributes.PutStr(semconv.AttributeVCSType, vcsType)

	if vcsType == semconv.AttributeVCSTypeGit {
		buildAttributes.PutStr(semconv.AttributeGitHTTPURL, repo.HTTPURL)
		buildAttributes.PutStr(semconv.AttributeGitSSHURL, repo.SSHURL)
		buildAttributes.PutStr(semconv.AttributeGitWWWURL, repo.Link)
	}
	// --- END VCS Info

	// --- Experimental attributes
	buildAttributes.PutStr(semconv.AttributesDroneBuildAfter, build.After)
	buildAttributes.PutStr(semconv.AttributesDroneBuildBefore, build.Before)
	buildAttributes.PutStr(semconv.AttributesDroneBuildLink, build.Link)
	buildAttributes.PutStr(semconv.AttributesDroneBuildRef, build.Ref)
	buildAttributes.PutStr(semconv.AttributesDroneBuildSource, build.Source)
	buildAttributes.PutStr(semconv.AttributesDroneBuildTarget, build.Target)
	buildAttributes.PutInt(semconv.AttributesDroneBuildParent, build.Parent)
	// --- UNKNOWN INFO

	for _, stage := range build.Stages {
		stageSpans := resourceSpans.ScopeSpans().AppendEmpty()
		stageSpan := stageSpans.Spans().AppendEmpty()
		stageAttributes := stageSpan.Attributes()

		stageAttributes.PutStr(semconv.AttributeDroneWorkflowItemKind, semconv.AttributeDroneWorkflowItemKindStage)
		stageSpan.SetTraceID(buildSpan.TraceID())
		stageSpan.SetSpanID(traceutils.NewSpanID())
		stageSpan.SetParentSpanID(buildSpan.SpanID())

		stageSpan.SetName(stage.Name)
		stageAttributes.PutStr(conventions.AttributeServiceName, stage.Name)

		traceutils.SetStatus(stage.Status, stageSpan)

		stageAttributes.PutInt(semconv.AttributeDroneStageNumber, int64(stage.Number))
		stageAttributes.PutInt(semconv.AttributeDroneStageID, stage.ID)
		stageAttributes.PutStr(semconv.AttributeDroneStageName, stage.Name)

		stageSpan.SetStartTimestamp(pcommon.Timestamp(stage.Started * 1000000000))
		stageSpan.SetEndTimestamp(pcommon.Timestamp(stage.Stopped * 1000000000))

		for _, step := range stage.Steps {
			if step.Status == "skipped" {
				continue
			}

			stepSpan := stageSpans.Spans().AppendEmpty()
			stepSpan.SetTraceID(stageSpan.TraceID())
			stepSpan.SetParentSpanID(stageSpan.SpanID())
			stepSpan.SetSpanID(traceutils.NewSpanID())

			stepAttributes := stepSpan.Attributes()

			stepAttributes.PutStr(semconv.AttributeDroneWorkflowItemKind, semconv.AttributeDroneWorkflowItemKindStep)
			stepAttributes.PutStr(semconv.AttributeDroneStageName, stage.Name)
			stepAttributes.PutInt(semconv.AttributeDroneStageID, int64(step.StageID))

			stepAttributes.PutStr(semconv.AttributeDroneStepName, step.Name)
			stepAttributes.PutInt(semconv.AttributeDroneStepID, step.ID)
			stepAttributes.PutInt(semconv.AttributeDroneStepNumber, int64(step.Number))

			traceutils.SetStatus(step.Status, stepSpan)
			stepSpan.SetName(step.Name)

			stepSpan.SetStartTimestamp(pcommon.Timestamp(step.Started * 1000000000))
			stepSpan.SetEndTimestamp(pcommon.Timestamp(step.Stopped * 1000000000))

			// TODO: Handle the error in logs retrieval better
			newLogs, err := generateLogs(droneClient, repo.Repo, *build, *stage, *step, buildSpan.TraceID(), stepSpan.SpanID())
			if err != nil {
				logger.Error("error retrieving logs", zap.Error(err))
				continue
			}

			newLogs.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
		}
	}

	return &traces, &logs
}

func generateLogs(drone drone.Client, repo drone.Repo, build drone.Build, stage drone.Stage, step drone.Step, traceId pcommon.TraceID, stepSpanId pcommon.SpanID) (plog.Logs, error) {
	logs := plog.NewLogs()

	lines, err := drone.Logs(repo.Namespace, repo.Name, int(build.Number), stage.Number, step.Number)
	if err != nil {
		return logs, err
	}

	log := logs.ResourceLogs().AppendEmpty()
	logScope := log.ScopeLogs().AppendEmpty()

	now := pcommon.NewTimestampFromTime(time.Now())

	prevLineTimestamp := int64(0)
	delta := int64(0)
	for _, line := range lines {
		if line.Timestamp == prevLineTimestamp {
			delta++
		} else {
			delta = 0
			prevLineTimestamp = line.Timestamp
		}

		record := logScope.LogRecords().AppendEmpty()
		record.SetTraceID(traceId)
		record.SetSpanID(stepSpanId)

		record.SetObservedTimestamp(now)
		record.SetTimestamp(pcommon.Timestamp((step.Started+line.Timestamp)*1000000000 + delta))
		record.Attributes().PutStr(semconv.AttributeDroneStageName, stage.Name)
		record.Attributes().PutStr(semconv.AttributeDroneStepName, step.Name)
		record.Attributes().PutInt(semconv.AttributeDroneBuildNumber, build.Number)
		record.Body().SetStr(line.Message)
	}

	return logs, nil
}
