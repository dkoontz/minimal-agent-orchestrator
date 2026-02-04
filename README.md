# Multi-Agent Development Workflow

A multi-agent system for Claude Code that coordinates planning, development, code review, and QA through specialized agents.

## Quick Start

```
Read `agents/orchestrator.md`
```

If you don't tell the orchestrator what task to work on it will prompt you for what to do next. You can also directly plan out tasks in advance.

### Option 1: Use the Planner (Recommended)

Have the planner agent help define your task:

```
"Read agents/planner.md and help me plan a task.
```

The planner will:
1. Ask clarifying questions about requirements
2. Explore the codebase for context
3. Draft acceptance criteria
4. Get your approval
5. Write the task file to `tasks/my-feature.md`

### Option 2: Write Task Directly

Create a task file manually in `tasks/`:

```bash
cp tasks/example.md tasks/my-feature.md
# Edit tasks/my-feature.md with your requirements
```

### Run the Orchestrator

Once you have a task file, run the orchestrator:

```
Read agents/orchestrator.md and begin work on my-feature
```

## Architecture

```
project/
├── agents/
│   ├── planner.md            # Gathers requirements, writes task files
│   ├── developer.md          # Implements code changes
│   ├── developer-review.md   # Reviews code quality
│   ├── qa.md                 # Tests functionality
│   └── orchestrator.md       # Coordinates dev/review/qa workflow
├── tasks/                     # Task definitions (one file per task)
│   └── example.md
└── workspaces/                # Auto-created per task
    └── {task-name}/
        ├── status.md
        └── reports/
```

## Full Workflow

```
Planner → [writes task file] → Orchestrator → Developer → Review → QA → Complete
                                                  ↑          |       |
                                                  └──────────┴───────┘
                                                       (on failure)
```

1. **Planner** - Interacts with user to define requirements and write task file
2. **Orchestrator** - Coordinates the development workflow
3. **Developer** - Implements the requirements, runs build and tests
4. **Review** - Checks code quality and identifies issues
5. **QA** - Verifies functionality through testing

## Task File Format

Task files follow a standard template:

```markdown
# Add user logout button

## Summary
Add a logout button to the application header that clears the user session.

## Requirements
- Add a logout button to the header component
- Clicking the button should clear the session
- User should be redirected to /login after logout

## Acceptance Criteria
- [ ] Button is visible when user is logged in
- [ ] Button is hidden when user is logged out
- [ ] Clicking button clears session storage
- [ ] User is redirected to /login after logout

## Out of Scope
- Logout confirmation dialog
- Remember me functionality

## Technical Context

### Files to Modify
- `src/components/Header.tsx` - Add logout button
- `src/hooks/useAuth.ts` - Add logout function

### Related Files
- `src/pages/Login.tsx` - Redirect target

### Patterns to Follow
- Use existing Button component from design system
- Follow useAuth hook pattern for session management

## Testing Requirements
- Unit test for logout function
- Integration test for redirect behavior

## Notes
- Session is stored in localStorage under 'auth_token' key
```

## Running Multiple Tasks

Start separate Claude Code sessions:

```bash
# Terminal 1: Plan a task
claude "Read agents/planner.md and help me plan a task.

Parameters:
TASK_NAME: feature-a
DESCRIPTION: Add dark mode support"

# Terminal 2: Execute a different task
claude "Read agents/orchestrator.md and execute the workflow.

Parameters:
TASK_FILE: tasks/feature-b.md"
```

## Agent Reports

Reports are numbered by iteration:

```
workspaces/my-feature/reports/
├── developer-1.md   # Initial implementation
├── review-1.md      # Review found issues
├── developer-2.md   # Fixed review issues
├── review-2.md      # Approved
└── qa-2.md          # QA passed
```

## Running Individual Agents

For debugging, run agents directly:

```bash
# Run planner
claude "Read agents/planner.md and help me plan a task.

Parameters:
TASK_NAME: my-feature
DESCRIPTION: Brief description of what you want"

# Run developer only
claude "Read agents/developer.md and execute your workflow.

Parameters:
TASK_FILE: tasks/my-feature.md
STATUS_FILE: workspaces/my-feature/status.md
REPORT_FILE: workspaces/my-feature/reports/developer-1.md"

# Run review only
claude "Read agents/developer-review.md and execute your workflow.

Parameters:
TASK_FILE: tasks/my-feature.md
DEV_REPORT: workspaces/my-feature/reports/developer-1.md
REPORT_FILE: workspaces/my-feature/reports/review-1.md"

# Run QA only
claude "Read agents/qa.md and execute your workflow.

Parameters:
TASK_FILE: tasks/my-feature.md
DEV_REPORT: workspaces/my-feature/reports/developer-1.md
REPORT_FILE: workspaces/my-feature/reports/qa-1.md"
```

## Customization

Modify the agent prompts in `agents/` to adjust:
- Planner questions and task template
- Review criteria
- Testing approach
- Report formats
- Workflow rules
