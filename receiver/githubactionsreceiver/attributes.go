// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func setWorkflowRunEventAttributes(attrs pcommon.Map, e *github.WorkflowRunEvent, config *Config) {
	if e == nil || e.GetRepo() == nil || e.GetWorkflowRun() == nil {
		return // Skip attribute setting if fields are nil
	}
	
	serviceName := generateServiceName(config, e.GetRepo().GetFullName())
	attrs.PutStr("service.name", serviceName)

	if actor := e.GetWorkflowRun().GetActor(); actor != nil {
		attrs.PutStr("ci.github.workflow.run.actor.login", actor.GetLogin())
	}

	attrs.PutStr("ci.github.workflow.run.conclusion", e.GetWorkflowRun().GetConclusion())
	attrs.PutStr("ci.github.workflow.run.created_at", e.GetWorkflowRun().GetCreatedAt().Format(time.RFC3339))
	attrs.PutStr("ci.github.workflow.run.display_title", e.GetWorkflowRun().GetDisplayTitle())
	attrs.PutStr("ci.github.workflow.run.event", e.GetWorkflowRun().GetEvent())
	attrs.PutStr("ci.github.workflow.run.head_branch", e.GetWorkflowRun().GetHeadBranch())
	attrs.PutStr("ci.github.workflow.run.head_sha", e.GetWorkflowRun().GetHeadSHA())
	attrs.PutStr("ci.github.workflow.run.html_url", e.GetWorkflowRun().GetHTMLURL())
	attrs.PutInt("ci.github.workflow.run.id", e.GetWorkflowRun().GetID())
	attrs.PutStr("ci.github.workflow.run.name", e.GetWorkflowRun().GetName())
	attrs.PutStr("ci.github.workflow.run.path", e.GetWorkflow().GetPath())

	if e.GetWorkflowRun().GetPreviousAttemptURL() != "" {
		htmlURL := transformGitHubAPIURL(e.GetWorkflowRun().GetPreviousAttemptURL())
		attrs.PutStr("ci.github.workflow.run.previous_attempt_url", htmlURL)
	}

	if len(e.GetWorkflowRun().ReferencedWorkflows) > 0 {
		var referencedWorkflows []string
		for _, workflow := range e.GetWorkflowRun().ReferencedWorkflows {
			referencedWorkflows = append(referencedWorkflows, workflow.GetPath())
		}
		attrs.PutStr("ci.github.workflow.run.referenced_workflows", strings.Join(referencedWorkflows, ";"))
	}

	attrs.PutInt("ci.github.workflow.run.run_attempt", int64(e.GetWorkflowRun().GetRunAttempt()))
	attrs.PutStr("ci.github.workflow.run.run_started_at", e.GetWorkflowRun().RunStartedAt.Format(time.RFC3339))
	attrs.PutStr("ci.github.workflow.run.status", e.GetWorkflowRun().GetStatus())
	if sender := e.GetSender(); sender != nil {
		attrs.PutStr("ci.github.workflow.run.sender.login", sender.GetLogin())
	}
	
	if triggeringActor := e.GetWorkflowRun().GetTriggeringActor(); triggeringActor != nil {
		attrs.PutStr("ci.github.workflow.run.triggering_actor.login", triggeringActor.GetLogin())
	}
	attrs.PutStr("ci.github.workflow.run.updated_at", e.GetWorkflowRun().GetUpdatedAt().Format(time.RFC3339))

	attrs.PutStr("ci.system", "github")

	attrs.PutStr("scm.system", "git")

	attrs.PutStr("scm.git.head_branch", e.GetWorkflowRun().GetHeadBranch())
	
	if headCommit := e.GetWorkflowRun().GetHeadCommit(); headCommit != nil {
		if author := headCommit.GetAuthor(); author != nil {
			attrs.PutStr("scm.git.head_commit.author.email", author.GetEmail())
			attrs.PutStr("scm.git.head_commit.author.name", author.GetName())
		}
		if committer := headCommit.GetCommitter(); committer != nil {
			attrs.PutStr("scm.git.head_commit.committer.email", committer.GetEmail())
			attrs.PutStr("scm.git.head_commit.committer.name", committer.GetName())
		}
		attrs.PutStr("scm.git.head_commit.message", headCommit.GetMessage())
		attrs.PutStr("scm.git.head_commit.timestamp", headCommit.GetTimestamp().Format(time.RFC3339))
	}
	attrs.PutStr("scm.git.head_sha", e.GetWorkflowRun().GetHeadSHA())

	if len(e.GetWorkflowRun().PullRequests) > 0 {
		var prUrls []string
		for _, pr := range e.GetWorkflowRun().PullRequests {
			prUrls = append(prUrls, convertPRURL(pr.GetURL()))
		}
		attrs.PutStr("scm.git.pull_requests.url", strings.Join(prUrls, ";"))
	}

	attrs.PutStr("scm.git.repo", e.GetRepo().GetFullName())
}
