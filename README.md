# PDCA Agent Orchestrator

A multi-agent development workflow for [opencode](https://opencode.ai) that drives a goal
to completion through repeated **Plan → Do → Check → Act** cycles against a *living plan*.

A single primary agent — the **orchestrator** — coordinates nine specialized subagents.
The orchestrator never writes code or runs tests itself; it routes work to the right agent,
records what was learned each cycle, and keeps the plan up to date. All plan and history
state flows through one tool, `pdca`, so the on-disk layout is an implementation detail the
agents never touch directly.

- Each goal gets its own git **branch + worktree**; every cycle is committed there, so the
  base branch stays clean until you choose to merge.
- A **living plan** accumulates known facts, open questions, decisions, and increments
  across cycles — the system gets smarter about the goal as it iterates.
- Stuck loops are detected automatically and routed to a read-only **reflector** before
  escalating to you.

## Agents

| Agent | Mode | Edits code? | Role |
|---|---|---|---|
| `pdca-orchestrator` | primary | no | Runs the whole workflow: drafts the plan with you, creates the worktree, drives the PDCA loop, commits each cycle, then rebases the goal branch onto base and squash-merges it in. |
| `pdca-investigator` | subagent | no (read-only) | Answers **one** scoped Open Question using read-only code search; cites files and lines. |
| `pdca-planner` | subagent | no (read-only) | Turns the next step into a single increment spec: hypothesis, scope, acceptance criteria. Does not design the whole feature. |
| `pdca-critic` | subagent | no (read-only) | Strategic skeptic. Attacks whether the plan's approach is justified by evidence in the plan and history — unsupported strategies, unwarranted assumptions, premature solutions. Runs before the Developer as a gate. |
| `pdca-synthesizer` | subagent | no (read-only) | Decision agent. Weighs the plan against the critique in light of the goal and history, then decides: **accept** the plan, **revise** it, or **investigate first**. Read-only — the orchestrator carries out the decision. |
| `pdca-developer` | subagent | **yes** | Implements exactly **one** increment per spec, verifies the build/tests, reports changes and any plan gaps. Does not commit. |
| `pdca-reviewer` | subagent | no (read-only) | Reviews the developer's diff for quality, scope adherence, and plan consistency. Separates code issues from plan issues. |
| `pdca-qa` | subagent | no (read-only) | Verifies the increment against its acceptance criteria, runs tests, reports observed vs. expected. |
| `pdca-debugger` | subagent | no (read-only) | Root-causes an unexpected failure from a failing entry. Diagnoses only — never applies a fix. |
| `pdca-reflector` | subagent | no (read-only) | Dispatched when the loop is stuck. Re-reads the plan and recent cycles, tags facts as measured vs. inferred, and proposes alternative framings and the highest-leverage next question. |

The orchestrator is `mode: primary` (you interact with it directly). The other nine are
`mode: subagent` — the orchestrator dispatches them with the `Task` tool, one at a time,
and reads each one's written entry before proceeding.

## Workflow graph

```mermaid
flowchart TD
    Start([User states a goal]) --> Draft["Draft initial plan<br/>from the conversation"]
    Draft --> Confirm{"User confirms?"}
    Confirm -->|"Revise"| Draft
    Confirm -->|"Cancel"| Stop(["Exit — no side effects"])
    Confirm -->|"Proceed"| Bootstrap["Create worktree + branch<br/>pdca init / plan-write / commit"]

    Bootstrap --> Loop["PDCA cycle N"]
    Loop --> Plan["Plan — read the living plan"]
    Plan --> PlanChoice{"What does the plan need?"}
    PlanChoice -->|"Open question blocks"| Investigator[pdca-investigator]
    PlanChoice -->|"Increment clear"| NeedsSpecification{"Needs a spec?"}
    PlanChoice -->|"Stuck signal"| Reflector[pdca-reflector]
    PlanChoice -->|"Goal met"| Completion[Completion]

    NeedsSpecification -->|"Yes"| Planner[pdca-planner]
    NeedsSpecification -->|"No (inline)"| Critic[pdca-critic]
    Planner --> Critic
    Critic --> Synthesizer[pdca-synthesizer]
    Synthesizer -->|"Accept plan"| Developer[pdca-developer]
    Synthesizer -->|"Revise / Investigate"| Act

    Investigator --> Check
    Developer --> Check
    Reflector --> Check

    Check{"Check — what to verify?"}
    Check -->|"Code changed"| Reviewer[pdca-reviewer]
    Check -->|"Observable behavior"| QA[pdca-qa]
    Check -->|"Failure, unclear cause"| Debugger[pdca-debugger]
    Check -->|"Nothing to check"| Act

    Reviewer --> Act
    QA --> Act
    Debugger --> Act

    Act["Act — update plan, commit"] --> Decide{"Next?"}
    Decide -->|"Continue"| Loop
    Decide -->|"Goal met"| Completion

    Completion --> Rebase["Rebase goal branch onto base<br/>(automatic)"]
    Rebase --> Squash["Squash-merge into base"]
```

### Which agent, when

| Situation | Phase | Agent dispatched |
|---|---|---|
| An Open Question blocks progress | Plan / Do | `pdca-investigator` |
| Next increment is clear but needs a spec | Plan / Do | `pdca-planner` → `pdca-critic` → `pdca-synthesizer` → `pdca-developer` |
| Next increment is obvious | Plan / Do | `pdca-critic` → `pdca-synthesizer` → `pdca-developer` (inline spec) |
| A stuck signal fires | Plan / Do | `pdca-reflector` (read-only cycle) |
| Code changed and needs review | Check | `pdca-reviewer` |
| Observable behavior to verify | Check | `pdca-qa` |
| Unexpected failure, unclear root cause | Check | `pdca-debugger` |
| Goal met, no blocking questions | — | Completion (no agent) |

**Critique gate.** On the Implement path the plan always passes through `pdca-critic`
then `pdca-synthesizer` before any code is written. The Synthesizer returns one of three
decisions: **accept plan** (proceed to the Developer), **revise plan** (fold the required
changes into the plan and re-plan in the next cycle), or **investigate first** (a
question must be answered before planning can continue). Revise and investigate skip the
Check phase — no code was written — and go straight to Act. Only one critique round runs
per cycle; a revise becomes a fresh cycle rather than re-looping the gate.

**Stuck signals** (any one forces a Reflect cycle before more Do cycles): 
- Two consecutive cycles on the same question with no progress; a new Decision added in two of the last three cycles 
- Certainty-then-revert language repeating across cycles
- A fix added and reverted in 2 of the last 5 cycles

The orchestrator reflects before it escalates to you.

## Starting the orchestrator

The workflow runs inside opencode.

1. **Launch opencode** in the repository where the work should happen.
2. **Switch to the `pdca-orchestrator` agent** — press `Tab` to cycle agents until it's active.
3. **State your goal** in plain language, e.g.:
   > make the login form show a validation error when the email field is empty

   If you've already discussed the task in the session, the orchestrator distills that
   conversation (goal, known facts, decisions, open questions) into the seed plan.
4. **Confirm the plan.** The orchestrator drafts a living plan and asks you to
   **Proceed / Revise / Cancel**. Revise freely — steps run in-session and nothing touches
   git until you choose Proceed.
5. **On Proceed**, it creates a `{goal-name}` branch + worktree, writes the plan, and enters
   the PDCA loop. It returns to you only at **completion**: it rebases the goal branch
   onto base and squash-merges it as a single clean commit (no prompt). Other in-flight
   worktrees are left untouched.

You can course-correct at any time — the orchestrator also asks you via a prompt before
adjusting scope (it never silently drops or shrinks a planned feature).

## Repository layout

```
opencode/
├── agents/
│   ├── pdca-orchestrator.md     # primary — runs the loop
│   ├── pdca-investigator.md     # subagent — answers one open question
│   ├── pdca-planner.md          # subagent — specs one increment
│   ├── pdca-critic.md           # subagent — critiques the plan's approach
│   ├── pdca-synthesizer.md      # subagent — decides accept / revise / investigate
│   ├── pdca-developer.md        # subagent — implements one increment
│   ├── pdca-reviewer.md         # subagent — reviews the diff
│   ├── pdca-qa.md               # subagent — verifies acceptance criteria
│   ├── pdca-debugger.md         # subagent — root-causes a failure
│   └── pdca-reflector.md        # subagent — reframes when stuck
└── tools/
    └── pdca.ts                  # the `pdca` tool: living-plan + history I/O
```

All plan and cycle state is read and written through the `pdca` tool (`plan-read`,
`plan-write`, `complete`, and the `entry-*` / `list` / `template` commands) — the agents
never reference or hand-edit the underlying files.
