# Multi-Agent Development Workflow

A multi-agent system for Claude Code that coordinates planning, development, code review, and QA through specialized agents.

## Quick Start

### If you have a task

Just tell the orchestrator which task to work on
```
Read `agents/orchestrator.md` and start work on `tasks/task-name/plan.md`
```

### If you don't have a task

```
Read `agents/orchestrator.md` and help me plan this task: <your task description goes here>
```

or you can run the planner directly

```
Read `agents/planner.md` and help me plan this task: <your task description goes here>
```

The planner will:
1. Ask clarifying questions about requirements
2. Explore the codebase for context
3. Draft acceptance criteria
4. Get your approval
5. Write the task file to `tasks/my-feature/plan.md`

## Architecture

```
project/
├── agents/
│   ├── planner.md            # Gathers requirements, writes task files
│   ├── developer.md          # Implements code changes
│   ├── developer-review.md   # Reviews code quality
│   ├── qa.md                 # Tests functionality
│   └── orchestrator.md       # Coordinates dev/review/qa workflow
└── tasks/                    # Task definitions (one directory per task)
    ├── {task-name}/          # Auto-created per task
    │   ├── plan.md
    │   ├── status.md
    │   └── reports/
    └── completed/            # Task directories are moved here when complete
        ├── {task1}
        ├── {task2}
        └── ...

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
tasks/my-feature/reports/
├── developer-1.md   # Initial implementation
├── review-1.md      # Review found issues
├── developer-2.md   # Fixed review issues
├── review-2.md      # Approved
└── qa-2.md          # QA passed
```

### Project Standards

Two files are provided for adding project-specific standards that agents will follow:

- **`agents/CODING_STANDARDS.md`** - Coding conventions, patterns, and style rules for your project. The developer agent reads this before implementing changes, the reviewer checks code against it, and the planner references it when scoping tasks that involve code changes.

- **`agents/QA_STANDARDS.md`** - Project-specific QA instructions such as how to write integration tests, what test frameworks to use, or testing conventions. The QA agent reads this during its testing workflow.

Both files ship empty. Add your project's conventions and the agents will incorporate them into their workflows automatically.

## Customization

Modify the agent prompts in `agents/` to adjust:
- Planner questions and task template
- Review criteria
- Testing approach
- Report formats
- Workflow rules