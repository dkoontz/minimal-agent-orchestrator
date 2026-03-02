# Multi-Agent Development Workflow

A multi-agent system for Claude Code that coordinates planning, development, code review, and QA through specialized agents.

Two versions are provided:

|               | [Plain Agents](#plain-agents)                                        | [Claude Sub-Agents](#claude-sub-agents)                                       |
| ------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| How it works  | Agent instructions are loaded as text snippets into Claude's context | Agents are registered as Claude Code sub-agents with defined tools and models |
| Orchestration | Read `agents/orchestrator.md` as a snippet                           | `/orchestrate-plan` and `/orchestrate-task` slash commands                    |
| Extras        | —                                                                    | Typed tool access per agent                                                   |
| Installation  | Copy `plain-agents/` as `agents/` to your project                    | Copy files into `.claude/agents/` and `.claude/commands/`                     |

---

## Full Workflow

The **Orchestrator** manages the entire workflow. It invokes each agent in sequence and decides what to do based on their reports.

**Planning workflow** (creates a task file):
```
Orchestrator → Planner → [Plan Reviewer ↔ Planner] → [User Approval]
```

**Development workflow** (executes a task file):
```
Orchestrator → Developer → Review → QA → Complete
                   ↑          |       |
                   └──────────┴───────┘
                         (on failure)
```

1. **Planner** — Interacts with the user to define requirements and write the task file
2. **Technical Plan Reviewer** — Validates the plan; the orchestrator loops it back to the planner if changes are needed
3. **Developer** — Implements the requirements, runs build and tests
4. **Reviewer** — Checks code quality and identifies issues
5. **QA** — Verifies functionality through testing

---

## Plain Agents

Agents work as text snippets — you paste or reference the agent file in your prompt and Claude follows the instructions.

### Installation

Copy `plain-agents/` into your project as `agents/`:

### Usage

If you have a task file:

```
Read `agents/orchestrator.md` and start work on `tasks/task-name/plan.md`
```

If you don't have a task yet:

```
Read `agents/orchestrator.md` and help me plan this task: <your task description>
```

### Project structure

```
your-project/
├── CODING_STANDARDS.md              # Your project's coding conventions (fill this in)
├── QA_STANDARDS.md                  # Your project's QA conventions (fill this in)
├── agents/
│   ├── orchestrator.md              # Coordinates the workflow
│   ├── planner.md                   # Gathers requirements, writes task files
│   ├── developer.md                 # Implements code changes
│   ├── developer-review.md          # Reviews code quality
│   ├── qa.md                        # Tests functionality
│   └── technical-plan-reviewer.md   # Reviews plans before development
└── tasks/
    ├── {task-name}/
    │   ├── plan.md
    │   ├── status.md
    │   └── *.md              # iteration reports
    └── completed/
```

### Customization

Modify the agent files in `agents/` to adjust review criteria, testing approach, report formats, or workflow rules.

---

## Claude Sub-Agents

Agents are registered as Claude Code sub-agents — they have defined tool access, a specified model, and are invoked natively by the Task tool rather than loaded as text snippets. The workflow is started with the `/orchestrate-plan` or `/orchestrate-task` slash commands.

### Installation

Copy the agent and command directories into your project's `.claude/` directory:

```
path/to/your-project/.claude/agents/
path/to/your-project/.claude/commands/
```

Add the CODING/QA standards templates to your project root (agents read these at runtime).

Alternatively you can install these files in your global `~/.claude` directory to make them available for all projects.

### Usage

Plan a new task (produces a task file):

```
/orchestrate-plan add a logout button
```

Or run without arguments to be prompted:

```
/orchestrate-plan
```

Once you have a task file, start development:

```
/orchestrate-task tasks/add-logout-button/plan.md
```

Or run without arguments to be prompted for the task file path:

```
/orchestrate-task
```

### Project structure

```
your-project/
├── .claude/
│   ├── agents/
│   │   ├── planner.md                  # Gathers requirements, writes task files
│   │   ├── developer.md                # Implements code changes
│   │   ├── developer-review.md         # Reviews code quality
│   │   ├── qa.md                       # Tests functionality
│   │   └── technical-plan-reviewer.md  # Reviews plans before development
│   └── commands/
│       ├── orchestrate-plan.md         # /orchestrate-plan slash command
│       └── orchestrate-task.md         # /orchestrate-task slash command
├── CODING_STANDARDS.md       # Your project's coding conventions (fill this in)
├── QA_STANDARDS.md           # Your project's QA conventions (fill this in)
└── tasks/
    ├── {task-name}/
    │   ├── plan.md
    │   ├── status.md
    │   └── *.md              # iteration reports
    └── completed/
```

### Customization

Modify the agent files in `.claude/agents/` and the command in `.claude/commands/` to adjust behavior. The YAML frontmatter in each agent file controls the model and available tools.

---

## Task File Format

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

---

## Agent Reports

Reports are numbered by iteration and written to the task directory:

```
tasks/my-feature/
├── plan.md
├── status.md
├── developer-1.md   # Initial implementation
├── review-1.md      # Review found issues
├── developer-2.md   # Fixed review issues
├── review-2.md      # Approved
└── qa-2.md          # QA passed
```

---

## Project Standards

Fill in these files with your project's conventions — agents read them automatically:

- **`CODING_STANDARDS.md`** — Coding conventions, patterns, and style rules. The developer reads this before implementing, the reviewer checks code against it, and the planner references it when scoping tasks.
  - Plain agents: lives at `CODING_STANDARDS.md`
  - Claude sub-agents: lives at the project root

- **`QA_STANDARDS.md`** — Project-specific QA instructions: test framework, how to run tests, integration test conventions. The QA agent reads this during its workflow.
  - Plain agents: lives at `QA_STANDARDS.md`
  - Claude sub-agents: lives at the project root

Both files ship empty. The more detail you add, the better agents will align with your project's expectations.
