import { tool } from "@opencode-ai/plugin"
import { existsSync, mkdirSync, renameSync, readdirSync } from "node:fs"
import { join } from "node:path"

const VALID_TYPES = [
  "plan", "act", "investigator", "planner", "developer",
  "reviewer", "qa", "debugger", "reflector"
]

const TEMPLATES: Record<string, string> = {
  goal: `## Goal

{What success looks like. Immutable unless the user changes it.}

## Status

- Cycle: 1
- Phase: plan
- Next: {one-line description of the next increment or question}

## Known Facts

<!--
Things we've confirmed via investigation or implementation.
Format: "- {fact} (cycle {N})" — or "(pre-orchestration)" for items seeded from prior discussion at goal start.
Add here as cycles produce learnings. Never silently contradict a fact; if one becomes wrong, remove or update it and note the cycle.
-->

## Open Questions

<!--
Unknowns currently blocking progress. Each should be narrow enough for one Investigator cycle.
Format: "- {question}"
Move to Known Facts when resolved.
-->

## Decisions

<!--
Choices made and their rationale.
Format: "- {decision} — because {reason} (cycle {N})" — or "(pre-orchestration)" for decisions carried in from discussion at goal start.
-->

## Increments

<!--
Each attempted step, newest last.

### Increment 1: {title} (cycle {N})
- Hypothesis: {what we expected}
- Acceptance: {how we'd know it worked}
- Outcome: {done | failed | pivoted} — {one-line learning}
*/`,

  history: `## Plan (orchestrator)

    **Intent**: investigate | implement | verify | debug
    **Hypothesis**: {what this cycle should produce}
    **Acceptance**: {how we'll know it worked}
    **Plan**: {which agents will run, in order}

## Investigator

    **Question**: {single scoped question from the orchestrator}
    **Answer**: {1–3 sentence direct answer}
    **Evidence**:
    - \`{path}:{line}\` — {what this shows}
    **Collateral findings**:
    - {anything the orchestrator should know that came up incidentally, or "none"}
    **Still unclear**:
    - {parts of the question you couldn't answer and why, or "none"}

## Planner

    **Title**: {short increment name}
    **Hypothesis**: {what you expect this change to produce}
    **Scope**:
    - In: {files/functions in scope}
    - Out: {what is explicitly out of scope}
    **Acceptance criteria**:
    - {observable condition}
    **Assumptions**:
    - {unconfirmed assumption — flag for orchestrator, or "none"}
    **Risks**:
    - {way this could fail or reveal a plan gap, or "none"}

## Developer

    **Summary**: {one or two sentences describing what you changed}
    **Files changed**:
    - \`{path}\` — {short note}
    **Build status**: PASS | FAIL
    {if FAIL, paste the error output verbatim in a fenced code block}
    **Test status**:
    - {suite or test name} — {PASS | FAIL | not run} — {brief detail}
    **Plan gaps discovered**:
    - {anything that made the spec hard, contradictory, or impossible as written, or "none"}
    **Acceptance self-check**:
    - {criterion}: {likely met | likely unmet | unclear} — {one-line reason}

## Reviewer

    **Verdict**: APPROVED | CHANGES REQUESTED
    **Code issues**:
    - [BLOCKING] \`{path}:{line}\` — {issue}
    - [SUGGESTION] \`{path}:{line}\` — {issue}
    **Plan issues**:
    - {gap or contradiction in plan, or "none"}
    **Notes**:
    - {non-blocking observation, or "none"}

## QA

    **Verdict**: PASS | FAIL
    **Per-criterion results**:
    - {criterion} | method: {verification method} | observed: {what happened} | PASS | FAIL
    **Test suite result**:
    - {suite} — {PASS | FAIL | not run} — {brief detail}
    **Surprises**:
    - {behavior not predicted by the spec, even if it didn't cause failure, or "none"}
    **Environment issues**:
    - {anything that blocked verification, or "none"}

## Debugger

    **Symptom**: {what was observed}
    **Reproduction**: {minimum steps, or "could not reproduce — {why}"}
    **Root cause**: {underlying defect, one or two sentences}
    **Evidence**:
    - \`{path}:{line}\` — {what this shows}
    **Why the plan missed it**: {which assumption in the plan or spec turned out to be wrong}
    **Suggested next increment**: {what the orchestrator should plan next — do not implement it here}

## Reflector

    **Stuck signal**: {one-sentence reason the orchestrator dispatched Reflect}
    **Current frame**: {one sentence naming the question the last N cycles have been treating as central}
    **Measured vs inferred audit**:
    - [measured] {Known Fact} — cycle {N} — {one-line evidence}
    - [inferred] {Known Fact} — cycle {N} — rests on {prior inferred fact, or "direct from code reading"}
    **Invariants (what has not moved)**:
    - {phenomenon that persisted across every intervention}
    **Alternative framings** (2–4, each a different central question the same evidence could be reorganized around):
    1. {framing} — would predict: {what would be true if this frame is right}
    2. {framing} — would predict: {...}
    **Highest-leverage unanswered question**: {the one question whose answer would most shrink the space of viable framings — plus a one-line justification}
    **Rotation check**: {adjacent Open Questions / clusters / increments that are cheaper now, or "none obvious"}
    **Gains across the stuck stretch**: {one paragraph, stated as what is now known that was not known N cycles ago — do not frame as failures or falsifications}
    **Flags for the orchestrator**:
    - {any Decision in the plan that a reframing would invalidate, or "none"}
    - {any Known Fact tagged [inferred] that should be measured before further Do cycles, or "none"}

## Act (orchestrator)

    **Plan updates**:
    - {what moved where in the plan — e.g. "Q: 'is Session shared?' → Known Facts"}
    **Gain**: {one line — what is now known (measured or inferred) that was not known before this cycle, even if the code change was reverted}
    **Decision**: continue to cycle {N+1} | goal met | reflect | escalate
    **Reason**: {one-line justification}`,

  plan: `**Intent**: investigate | implement | verify | debug
**Hypothesis**: {what this cycle should produce}
**Acceptance**: {how we'll know it worked}
**Plan**: {which agents will run, in order}`,

  investigator: `**Question**: {single scoped question from the orchestrator}
**Answer**: {1–3 sentence direct answer}
**Evidence**:
- \`{path}:{line}\` — {what this shows}
**Collateral findings**:
- {anything the orchestrator should know that came up incidentally, or "none"}
**Still unclear**:
- {parts of the question you couldn't answer and why, or "none"}`,

  planner: `**Title**: {short increment name}
**Hypothesis**: {what you expect this change to produce}
**Scope**:
- In: {files/functions in scope}
- Out: {what is explicitly out of scope}
**Acceptance criteria**:
- {observable condition}
**Assumptions**:
- {unconfirmed assumption — flag for orchestrator, or "none"}
**Risks**:
- {way this could fail or reveal a plan gap, or "none"}`,

  developer: `**Summary**: {one or two sentences describing what you changed}
**Files changed**:
- \`{path}\` — {short note}
**Build status**: PASS | FAIL
{if FAIL, paste the error output verbatim in a fenced code block}
**Test status**:
- {suite or test name} — {PASS | FAIL | not run} — {brief detail}
**Plan gaps discovered**:
- {anything that made the spec hard, contradictory, or impossible as written, or "none"}
**Acceptance self-check**:
- {criterion}: {likely met | likely unmet | unclear} — {one-line reason}`,

  reviewer: `**Verdict**: APPROVED | CHANGES REQUESTED
**Code issues**:
- [BLOCKING] \`{path}:{line}\` — {issue}
- [SUGGESTION] \`{path}:{line}\` — {issue}
**Plan issues**:
- {gap or contradiction in plan, or "none"}
**Notes**:
- {non-blocking observation, or "none"}`,

  qa: `**Verdict**: PASS | FAIL
**Per-criterion results**:
- {criterion} | method: {verification method} | observed: {what happened} | PASS | FAIL
**Test suite result**:
- {suite} — {PASS | FAIL | not run} — {brief detail}
**Surprises**:
- {behavior not predicted by the spec, even if it didn't cause failure, or "none"}
**Environment issues**:
- {anything that blocked verification, or "none"}`,

  debugger: `**Symptom**: {what was observed}
**Reproduction**: {minimum steps, or "could not reproduce — {why}"}
**Root cause**: {underlying defect, one or two sentences}
**Evidence**:
- \`{path}:{line}\` — {what this shows}
**Why the plan missed it**: {which assumption in the plan or spec turned out to be wrong}
**Suggested next increment**: {what the orchestrator should plan next — do not implement it here}`,

  reflector: `**Stuck signal**: {one-sentence reason the orchestrator dispatched Reflect}
**Current frame**: {one sentence naming the question the last N cycles have been treating as central}
**Measured vs inferred audit**:
- [measured] {Known Fact} — cycle {N} — {one-line evidence}
- [inferred] {Known Fact} — cycle {N} — rests on {prior inferred fact, or "direct from code reading"}
**Invariants (what has not moved)**:
- {phenomenon that persisted across every intervention}
**Alternative framings** (2–4, each a different central question the same evidence could be reorganized around):
1. {framing} — would predict: {what would be true if this frame is right}
2. {framing} — would predict: {...}
**Highest-leverage unanswered question**: {the one question whose answer would most shrink the space of viable framings — plus a one-line justification}
**Rotation check**: {adjacent Open Questions / clusters / increments that are cheaper now, or "none obvious"}
**Gains across the stuck stretch**: {one paragraph, stated as what is now known that was not known N cycles ago — do not frame as failures or falsifications}
**Flags for the orchestrator**:
- {any Decision in the plan that a reframing would invalidate, or "none"}
- {any Known Fact tagged [inferred] that should be measured before further Do cycles, or "none"}`,

  act: `**Plan updates**:
- {what moved where in the plan — e.g. "Q: 'is Session shared?' → Known Facts"}
**Gain**: {one line — what is now known (measured or inferred) that was not known before this cycle, even if the code change was reverted}
**Decision**: continue to cycle {N+1} | goal met | reflect | escalate
**Reason**: {one-line justification}`,
}

const TEMPLATE_NAMES = Object.keys(TEMPLATES)

interface PlanFile {
  goal: string
  cycle: number
  plan: string
}

interface Entry {
  type: string
  timestamp: string
  body: string
}

interface CycleFile {
  cycle: number
  entries: Record<string, Entry>
}

function planPath(worktree: string, goal: string): string {
  return join(worktree, "goals", `${goal}.plan.json`)
}

function cyclePath(worktree: string, goal: string, cycle: number): string {
  return join(worktree, "goals", `${goal}-${cycle}.json`)
}

function entryId(cycle: number, type: string): string {
  return `cycle-${cycle}-${type}`
}

function parseEntryId(id: string): { cycle: number; type: string } | null {
  const m = id.match(/^cycle-(\d+)-([a-z]+)$/)
  if (!m) return null
  return { cycle: parseInt(m[1]), type: m[2] }
}

function titleCase(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

async function readPlan(worktree: string, goal: string): Promise<PlanFile> {
  const p = planPath(worktree, goal)
  if (!existsSync(p)) throw new Error(`plan not found: ${p}`)
  return Bun.file(p).json()
}

async function writePlan(worktree: string, goal: string, data: PlanFile): Promise<void> {
  const p = planPath(worktree, goal)
  const tmp = `${p}.tmp.${process.pid}`
  await Bun.write(tmp, JSON.stringify(data, null, 2) + "\n")
  renameSync(tmp, p)
}

async function readCycle(worktree: string, goal: string, cycle: number): Promise<CycleFile | null> {
  const p = cyclePath(worktree, goal, cycle)
  if (!existsSync(p)) return null
  return Bun.file(p).json()
}

async function writeCycleFile(worktree: string, goal: string, data: CycleFile): Promise<void> {
  const p = cyclePath(worktree, goal, data.cycle)
  const tmp = `${p}.tmp.${process.pid}`
  await Bun.write(tmp, JSON.stringify(data, null, 2) + "\n")
  renameSync(tmp, p)
}

function cycleFiles(worktree: string, goal: string): number[] {
  const dir = join(worktree, "goals")
  if (!existsSync(dir)) return []
  const re = new RegExp(`^${goal}-(\\d+)\\.json$`)
  const cycles: number[] = []
  for (const f of readdirSync(dir)) {
    const m = f.match(re)
    if (m) cycles.push(parseInt(m[1]))
  }
  cycles.sort((a, b) => a - b)
  return cycles
}

export default tool({
  description:
    "PDCA workflow tool for managing living plans and per-cycle history entries stored as JSON files under goals/. Use this for all PDCA bookkeeping and retrieving entry/plan templates.",
  args: {
    command: tool.schema
      .enum(["init", "plan-read", "plan-write", "complete", "cycle-create", "entry-write", "entry-read", "entry-check", "list", "last-entry", "template"])
      .describe("The pdca operation to perform"),
    goal: tool.schema.string().describe("Kebab-case goal name (e.g. 'fix-login-bug')"),
    cycle: tool.schema.number().optional().describe("Cycle number for cycle-create"),
    type: tool.schema.string().optional().describe("Entry type for [entry-write, last-entry]: plan, act, investigator, planner, developer, reviewer, qa, debugger, reflector"),
    id: tool.schema.string().optional().describe("Entry ID for [entry-read, entry-check] (e.g. 'cycle-2-investigator')"),
    body: tool.schema.string().optional().describe("Markdown body for entry-write, plan-write"),
    json: tool.schema.boolean().optional().describe("Output list as JSON"),
    emptyOnly: tool.schema.boolean().optional().describe("Show only empty entries in list"),
    name: tool.schema.string().optional().describe("Template name for template command: goal, history, plan, act, investigator, planner, developer, reviewer, qa, debugger, reflector"),
    workdir: tool.schema.string().optional().describe("Override worktree root directory (absolute path). Used when operating in a git worktree separate from the main repo."),
  },
  async execute(args, context) {
    const w = args.workdir || context.worktree
    const { command, goal } = args

    try {
      switch (command) {
        // ── init ──────────────────────────────────────────────
        case "init": {
          const p = planPath(w, goal)
          if (existsSync(p)) throw new Error(`plan already exists: ${p}`)
          mkdirSync(join(w, "goals"), { recursive: true })
          const plan: PlanFile = { goal, cycle: 1, plan: "" }
          await Bun.write(p, JSON.stringify(plan, null, 2) + "\n")
          return `Created ${p}`
        }

        // ── plan-read ─────────────────────────────────────────
        case "plan-read": {
          const plan = await readPlan(w, goal)
          return plan.plan
        }

        // ── plan-write ────────────────────────────────────────
        case "plan-write": {
          if (args.body === undefined) throw new Error("plan-write requires <body>")
          const plan = await readPlan(w, goal)
          plan.plan = args.body
          await writePlan(w, goal, plan)
          return `Plan updated: ${planPath(w, goal)}`
        }

        // ── cycle-create ──────────────────────────────────────
        case "cycle-create": {
          const cycle = args.cycle
          if (cycle == null) throw new Error("cycle-create requires <cycle>")
          const cp = cyclePath(w, goal, cycle)
          if (existsSync(cp)) throw new Error(`cycle file already exists: ${cp}`)
          const plan = await readPlan(w, goal)
          plan.cycle = cycle
          const cf: CycleFile = { cycle, entries: {} }
          await writeCycleFile(w, goal, cf)
          await writePlan(w, goal, plan)
          return `Cycle ${cycle} started`
        }

        // ── entry-write ───────────────────────────────────────
        case "entry-write": {
          const type = args.type
          // Check for extra args: if no type arg provided, agent might have split the
          // string incorrectly. Detect whether the args look correct.
          if (!type) throw new Error("entry-write requires <type> <body>")
          // Verify it's a valid, recognized type
          if (!VALID_TYPES.includes(type.toLowerCase())) {
            throw new Error(`invalid type: '${type}'. valid types: ${VALID_TYPES.join(", ")}`)
          }
          const t = type.toLowerCase()
          if (args.body === undefined) throw new Error("entry-write requires <body>")
          const plan = await readPlan(w, goal)
          const cycle = plan.cycle
          const id = entryId(cycle, t)

          // read or create cycle file
          let cf = await readCycle(w, goal, cycle)
          if (!cf) {
            mkdirSync(join(w, "goals"), { recursive: true })
            cf = { cycle, entries: {} }
          }

          if (cf.entries[id] && cf.entries[id].body.length > 0) {
            throw new Error(`entry already has content: ${id}`)
          }

          cf.entries[id] = {
            type: titleCase(t),
            timestamp: new Date().toISOString(),
            body: args.body,
          }
          await writeCycleFile(w, goal, cf)
          return id
        }

        // ── entry-read ────────────────────────────────────────
        case "entry-read": {
          if (!args.id) throw new Error("entry-read requires <id>")
          const parsed = parseEntryId(args.id)
          if (!parsed) throw new Error(`invalid entry id: ${args.id}`)
          const cf = await readCycle(w, goal, parsed.cycle)
          if (!cf || !cf.entries[args.id]) throw new Error(`entry not found: ${args.id}`)
          return cf.entries[args.id].body
        }

        // ── entry-check ───────────────────────────────────────
        case "entry-check": {
          if (!args.id) throw new Error("entry-check requires <id>")
          const parsed = parseEntryId(args.id)
          if (!parsed) return "false"
          const cf = await readCycle(w, goal, parsed.cycle)
          if (!cf || !cf.entries[args.id]) return "false"
          return cf.entries[args.id].body.length > 0 ? "true" : "false"
        }

        // ── list ──────────────────────────────────────────────
        case "list": {
          const cycles = cycleFiles(w, goal)
          const entries: Array<{
            id: string
            cycle: number
            type: string
            timestamp: string
            bodyBytes: number
            bodyLines: number
            empty: boolean
          }> = []

          for (const c of cycles) {
            const cf = await readCycle(w, goal, c)
            if (!cf) continue
            for (const [eid, e] of Object.entries(cf.entries)) {
              const entry = {
                id: eid,
                cycle: c,
                type: e.type,
                timestamp: e.timestamp,
                bodyBytes: e.body.length,
                bodyLines: e.body.length === 0 ? 0 : e.body.split("\n").length,
                empty: e.body.length === 0,
              }
              if (!args.emptyOnly || entry.empty) entries.push(entry)
            }
          }

          if (args.json) return JSON.stringify(entries, null, 2)

          if (entries.length === 0) return "(no entries)"

          return entries
            .map((e) => {
              const size = e.empty ? "(empty)" : `${e.bodyBytes}b / ${e.bodyLines} lines`
              return `${e.id}  [cycle ${e.cycle}, ${e.type}, ${size}]`
            })
            .join("\n")
        }

        // ── last-entry ────────────────────────────────────────
        case "last-entry": {
          if (!args.type) throw new Error("last-entry requires <type>")
          const t = args.type.toLowerCase()
          const cycles = cycleFiles(w, goal).reverse()
          for (const c of cycles) {
            const cf = await readCycle(w, goal, c)
            if (!cf) continue
            for (const [eid, e] of Object.entries(cf.entries)) {
              if (e.type.toLowerCase() === t) return eid
            }
          }
          throw new Error(`no "${t}" entries found`)
        }

        // ── template ──────────────────────────────────────────
        case "template": {
          const name = args.name
          if (!name) throw new Error(`template requires <name>. valid names: ${TEMPLATE_NAMES.join(", ")}`)
          const t = TEMPLATES[name.toLowerCase()]
          if (!t) throw new Error(`unknown template: '${name}'. valid names: ${TEMPLATE_NAMES.join(", ")}`)
          return t
        }

        // ── complete ──────────────────────────────────────────
        case "complete": {
          const p = planPath(w, goal)
          if (!existsSync(p)) throw new Error(`plan not found: ${p}`)
          const completedDir = join(w, "goals", "completed")
          mkdirSync(completedDir, { recursive: true })
          // Collect cycle list before moving anything (moving changes the dir contents).
          const cycles = cycleFiles(w, goal)
          const moved: string[] = []
          renameSync(p, join(completedDir, `${goal}.plan.json`))
          moved.push(`${goal}.plan.json`)
          for (const c of cycles) {
            renameSync(cyclePath(w, goal, c), join(completedDir, `${goal}-${c}.json`))
            moved.push(`${goal}-${c}.json`)
          }
          return `Completed ${goal}: archived ${moved.length} file(s) to goals/completed/`
        }

        default:
          throw new Error(`unknown command: ${command}`)
      }
    } catch (e: any) {
      return `ERROR: ${e.message}`
    }
  },
})
