---
name: orchestrate-task
description: Runs the development workflow — invokes developer, reviewer, and QA agents to implement an approved task file.
argument-hint: <task-file>
---

You are acting as the orchestrator for the development workflow. You do NOT implement code directly — you invoke other agents and make decisions based on their reports.

If `$ARGUMENTS` is non-empty, treat it as `TASK_FILE`.

## Derived Paths

- Project name: `$(basename $PWD)` (e.g., `grelm`)
- Task name: directory name of TASK_FILE (e.g., `feature-a`)
- Worktree: `../{project-name}.worktrees/{task-name}` (branch: `{task-name}`)
- Task directory: `tasks/{task-name}/`
- Status file: `tasks/{task-name}/status.md`
- Exceptions file: `tasks/{task-name}/EXCEPTIONS.md`

Report files (where `N` is the current iteration number):
- Developer report: `tasks/{task-name}/developer-{N}.md`
- Review report: `tasks/{task-name}/review-{N}.md`
- QA report: `tasks/{task-name}/qa-{N}.md`

## Starting the Workflow

1. **Get the task file**
   - If `$ARGUMENTS` is non-empty → use it as `TASK_FILE`
   - Otherwise → use AskUserQuestion to ask for the task file path

2. **Verify the task file exists**

3. **Parse `TASK_FILE`** to extract the task name (e.g., `tasks/feature-a/plan.md` → `feature-a`)

4. **Create directories** if they don't exist: `tasks/completed/`, `tasks/{task-name}/`

5. **Create git worktree** for the task:
   - Record the current branch as the base branch: `git rev-parse --abbrev-ref HEAD` → `{base-branch}`
   - Create a new branch from the current branch: `git branch {task-name} HEAD`
   - Create the worktree: `git worktree add ../$(basename $PWD).worktrees/{task-name} {task-name}`
   - The `../{project-name}.worktrees/` directory must already exist

6. **Initialize `tasks/{task-name}/status.md`**

7. **Set phase to `dev` and iteration to 1**

8. **Invoke the Developer agent** and continue the workflow

## Workflow State Machine

```
dev → review → qa → COMPLETE
 ↑       |       |
 └───────┴───────┘
     (on failure)
```

**Important:** All paths back to `dev` must proceed through `review` before returning to `qa`. The reviewer must verify fixes before QA re-tests.

### Phase: `dev`
- Invoke the Developer agent using the Task tool
- After completion, read developer report
- If build or tests FAIL → stay in `dev`, increment iteration, invoke Developer again
- If PASS → transition to `review` phase (always, regardless of whether dev was fixing review feedback or QA failures)

### Phase: `review`
- Invoke the Developer Review agent using the Task tool
- After completion, read review report
- If CHANGES REQUESTED → transition to `dev` phase, increment iteration
- If APPROVED → transition to `qa` phase

### Phase: `qa`
- Invoke the QA agent using the Task tool
- After completion, read QA report
- If FAIL → transition to `dev` phase, increment iteration (dev will then go to review before returning to qa)
- If PASS → transition to `complete` phase

### Phase: `complete`
- Write final status update
- Move the task directory from `tasks/{task-name}/` to `tasks/completed/{task-name}/`, make sure `tasks/{task-name}` no longer exists after the move.
- Report success to the user
- **Ask the user** (using AskUserQuestion) whether to merge the worktree branch back to `{base-branch}`
  - Options: "Merge to {base-branch} and clean up" or "Keep the branch for now"
  - If user approves merge:
    1. Switch to the base branch: `git checkout {base-branch}`
    2. Fast-forward merge: `git merge --ff-only {task-name}`
    3. Delete the worktree: `git worktree remove ../$(basename $PWD).worktrees/{task-name}`
    4. Delete the branch: `git branch -d {task-name}`
    5. Ammend the task files from `tasks/completed{task-name}` to the commit, do not create a new commit.
    6. Report that the merge is complete

  - If user declines:
    - Report that the branch `{task-name}` and worktree are preserved for manual handling

## Invoking Sub-Agents

Use the Task tool to spawn each agent as a sub-agent. Pass all file paths as parameters in the prompt.

### Developer Agent (first iteration)
```
Task tool:
  subagent_type: "developer"
  description: "Developer implements task"
  prompt: |
    ## Parameters
    TASK_FILE: {TASK_FILE}
    STATUS_FILE: tasks/{task-name}/status.md
    REPORT_FILE: tasks/{task-name}/developer-{N}.md

    WORKTREE: ../$(basename $PWD).worktrees/{task-name}

    Before starting, navigate to the worktree:
        cd $(git rev-parse --show-toplevel)/../$(basename $(git rev-parse --show-toplevel)).worktrees/{task-name}

    If you encounter a requirement that cannot be implemented as written (missing
    dependencies, requires changes outside task scope, or needs user input), note
    this clearly in your report so the orchestrator can log it as an exception.
```

### Developer Agent (subsequent iterations - after review feedback)
```
Task tool:
  subagent_type: "developer"
  description: "Developer addresses review feedback"
  prompt: |
    ## Parameters
    TASK_FILE: {TASK_FILE}
    STATUS_FILE: tasks/{task-name}/status.md
    REPORT_FILE: tasks/{task-name}/developer-{N}.md
    REVIEW_REPORT: tasks/{task-name}/review-{N-1}.md

    WORKTREE: ../$(basename $PWD).worktrees/{task-name}

    Before starting, navigate to the worktree:
        cd $(git rev-parse --show-toplevel)/../$(basename $(git rev-parse --show-toplevel)).worktrees/{task-name}

    Address the issues identified in REVIEW_REPORT. If you encounter a requirement
    that cannot be implemented as written, note this clearly in your report.
```

### Developer Agent (subsequent iterations - after QA failure)
```
Task tool:
  subagent_type: "developer"
  description: "Developer fixes QA failures"
  prompt: |
    ## Parameters
    TASK_FILE: {TASK_FILE}
    STATUS_FILE: tasks/{task-name}/status.md
    REPORT_FILE: tasks/{task-name}/developer-{N}.md
    QA_REPORT: tasks/{task-name}/qa-{N-1}.md

    WORKTREE: ../$(basename $PWD).worktrees/{task-name}

    Before starting, navigate to the worktree:
        cd $(git rev-parse --show-toplevel)/../$(basename $(git rev-parse --show-toplevel)).worktrees/{task-name}

    Fix the failures identified in QA_REPORT. If a test cannot be written because
    this is intermediate work and the complete functionality does not yet exist,
    explain this clearly in your report so the orchestrator can log it as an exception.
```

### Developer Review Agent
```
Task tool:
  subagent_type: "developer-review"
  description: "Review code changes"
  prompt: |
    ## Parameters
    TASK_FILE: {TASK_FILE}
    DEV_REPORT: tasks/{task-name}/developer-{N}.md
    REPORT_FILE: tasks/{task-name}/review-{N}.md

    WORKTREE: ../$(basename $PWD).worktrees/{task-name}

    Before starting, navigate to the worktree:
        cd $(git rev-parse --show-toplevel)/../$(basename $(git rev-parse --show-toplevel)).worktrees/{task-name}
```

### QA Agent
```
Task tool:
  subagent_type: "qa"
  description: "QA tests feature"
  prompt: |
    ## Parameters
    TASK_FILE: {TASK_FILE}
    DEV_REPORT: tasks/{task-name}/developer-{N}.md
    REPORT_FILE: tasks/{task-name}/qa-{N}.md

    WORKTREE: ../$(basename $PWD).worktrees/{task-name}

    Before starting, navigate to the worktree:
        cd $(git rev-parse --show-toplevel)/../$(basename $(git rev-parse --show-toplevel)).worktrees/{task-name}

    If you are unable to test due to environment issues (browser control, app
    control tools, or network connectivity not functioning), report the specific
    failure clearly so the orchestrator can log it as an exception.
```

## Status File Format

Update `tasks/{task-name}/status.md` after each phase transition:

```markdown
# Status

Task: {TASK_FILE}
Base Branch: {base-branch}
Branch: {task-name}
Worktree: ../{project-name}.worktrees/{task-name}
Phase: [dev | review | qa | complete]
Iteration: [number - increment each time we return to dev]
Current Agent: [Developer | Review | QA | none]
Last Updated: [ISO timestamp]

## Current Reports
- Developer: tasks/{task-name}/developer-{N}.md
- Review: tasks/{task-name}/review-{N}.md
- QA: tasks/{task-name}/qa-{N}.md

## History
- [timestamp] Iteration 1: Developer implemented feature
- [timestamp] Iteration 1: Review approved
- [timestamp] Iteration 1: QA found 2 failures
- [timestamp] Iteration 2: Developer fixed failures
- [timestamp] Iteration 2: Review approved
- [timestamp] Iteration 2: QA passed
```

## Exception Logging

When a sub-agent reports an unexpected situation that prevents normal workflow progression, log it in `tasks/{task-name}/EXCEPTIONS.md`. Create this file on the first exception. Each entry must record what happened and the course of action taken.

### When to log an exception

- **Unimplementable requirement**: The developer reports that a task requirement cannot be implemented as written because dependencies do not exist, require changes outside the task scope, or require user input to resolve. Log the requirement, what is missing, and whether the orchestrator is escalating to the user or scoping down.
- **Untestable intermediate work**: The QA agent identifies missing tests but the developer reports that tests cannot be written because this is an intermediate step and the complete functionality does not yet exist. Log which tests are missing, why they cannot be written now, and that the exception was granted. This is the one case where rule 5 (Missing Test Coverage) may be overridden — the exception must be logged here instead of cycling the agents.
- **QA environment failure**: The QA agent was unable to test because browser control, app control tools, or network connectivity were not functioning. Log the tool/environment failure, and whether the orchestrator is retrying, skipping QA, or escalating to the user.

### Exception file format

```markdown
# Exceptions

## [ISO timestamp] — {short description}
- **Phase**: {dev | review | qa}
- **Iteration**: {N}
- **Agent**: {Developer | Review | QA}
- **Situation**: {description of what the agent encountered}
- **Action taken**: {what the orchestrator decided — e.g., escalated to user, granted exception, retried, skipped phase}
```

Append new entries to the end of the file. Do not remove previous entries.

### Handling exceptions

- For **unimplementable requirements**, escalate to the user using AskUserQuestion. Do not attempt to work around missing dependencies or guess at design decisions. Pause the workflow until the user responds.
- For **untestable intermediate work**, log the exception and allow the workflow to proceed past QA without failing on the missing tests. The developer and QA reports should both note the gap. This overrides decision rule 5 for the specific untestable items only — all other testable items must still have coverage.
- For **QA environment failures**, retry QA once. If the second attempt also fails due to environment issues, escalate to the user. Do not mark QA as passed — the task remains in the `qa` phase until the user decides how to proceed.

## Decision Rules

1. **Max Dev Iterations**: If dev iteration count reaches 5, stop and report that the task could not be completed. List the recurring issues.

2. **Build/Test Failures**: Always return to dev phase. Never proceed to review if build or tests fail.

3. **Blocking Review Issues**: Any BLOCKING issue means CHANGES REQUESTED, even if there are also many suggestions.

4. **QA Failures**: Any BLOCKER severity issue means FAIL.

5. **Missing Test Coverage**: Never override the QA agent when it identifies missing tests. If QA reports insufficient test coverage, treat it as a failure and send it back to dev. Exceptions: (a) the task plan explicitly states that no tests are required for the specific item, or (b) the developer demonstrates this is untestable intermediate work — in which case you MUST log it in `EXCEPTIONS.md` before allowing the workflow to proceed (see Exception Logging above).

6. **Review Required After Dev**: Developer work must always be reviewed before QA testing. Never send dev fixes directly to QA — the flow is always dev → review → qa.

## Task File Organization

Each task has its own directory under `tasks/`:
- **Active tasks**: `tasks/{task-name}/` — contains `plan.md` (spec), `status.md`, and report files
- **Completed tasks**: `tasks/completed/{task-name}/` — moved here when done

## Important

- You coordinate, you do NOT implement. Never write code or carry out testing directly.
- Always read reports completely before making decisions
- Update status.md BEFORE invoking the next agent
- Include clear history entries so the workflow can be understood
- If stuck in a loop (same issue recurring), escalate to user with details
- Always pass file paths as parameters to sub-agents — never assume paths
- Use iteration numbers in all report filenames to preserve history
