# Judge research harness

A reusable multi-agent harness for investigating questions about this repo (and
the wider ecosystem) with **verified**, cited answers instead of one model's
first guess. It fans work out to cheap specialist agents, then puts every finding
in front of an adversarial **judge** before anything reaches the final report.

It lives in `.claude/workflows/judge-research.mjs` and runs through Claude Code's
`Workflow` tool.

## Model roles

The roles are fixed by design — cheap models do the legwork, the expensive model
only orchestrates and judges:

| Model  | Role                        | Why |
|--------|-----------------------------|-----|
| **haiku**  | Web research            | Fast/cheap fan-out over many subtopics via WebSearch/WebFetch |
| **sonnet** | Local code study        | Strong code reading, cheaper than opus for crawling files |
| **opus**   | Orchestrator + developer + **judge** | Plans the decomposition, adversarially verifies findings, writes the report, and does the actual code changes |
| **fable**  | Alternate orchestrator  | Swap in for the plan/synthesize steps via `orchestrator: "fable"` |

## Pipeline

```
Plan (opus/fable)                decompose question -> web subtopics + code areas
   │
Research (parallel)              haiku web researchers ── sonnet code crawlers
   │                             each returns findings = {claim, evidence, source}
Judge (opus, per finding)        try to REFUTE each claim -> supported|refuted|uncertain
   │
Synthesize (opus/fable)          cited report from SUPPORTED findings + caveats
```

Only judge-**supported** findings become load-bearing claims; refuted/uncertain
ones are surfaced in a Caveats section so nothing is silently dropped.

## Usage

From a Claude Code session in this repo, invoke the `Workflow` tool with the
saved workflow name and a question:

```
Workflow(name: "judge-research",
         args: "Which VictoriaMetrics vminsert versions are safe to keep in the CI matrix, and why?")
```

Steer it (skip planning, force subtopics/code areas, or use fable to orchestrate):

```
Workflow(name: "judge-research", args: {
  question: "Does vmprober's push path stay compatible across vminsert v1.97 -> v1.147?",
  subtopics: ["vminsert /api/v1/import changelog", "vmselect latencyOffset defaults"],
  codeAreas: ["internal/adapter", "tests/e2e"],
  orchestrator: "opus"
})
```

The workflow returns `{ question, counts, report, supported, rejected }` — read
`report` for the answer and `rejected` for what didn't hold up. Live progress is
visible via `/workflows`.

## When to use it

- Ecosystem/version questions where a wrong fact is expensive (e.g. picking
  image tags for the E2E matrix — this harness caught that the LTS *patch* tags
  are enterprise-only and would fail CI).
- "Is X actually true of our code?" audits that mix web docs + local code.
- Any research you want **checked** before acting on it.

For a single quick lookup, just ask directly — the harness is for questions worth
fanning out and adjudicating.
