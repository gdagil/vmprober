export const meta = {
  name: 'judge-research',
  description:
    'Fan-out research (haiku web + sonnet code) -> opus judge adversarially verifies each finding -> orchestrator synthesizes a cited report',
  phases: [
    { title: 'Plan', detail: 'orchestrator decomposes the question' },
    { title: 'Research', detail: 'haiku web + sonnet code, in parallel' },
    { title: 'Judge', detail: 'opus refutes/confirms each finding' },
    { title: 'Synthesize', detail: 'orchestrator writes the cited report' },
  ],
}

// ---------------------------------------------------------------------------
// Role assignment (fixed by design; see .claude/workflows/README.md):
//   haiku  -> web research         sonnet -> local code study
//   opus   -> orchestrate/develop/judge     fable -> alternate orchestrator
//
// Usage: pass a question string, or an object to skip/steer planning:
//   args: "Which vminsert versions are safe to drop from the CI matrix?"
//   args: { question: "...", subtopics: ["..."], codeAreas: ["..."], orchestrator: "fable" }
// ---------------------------------------------------------------------------

const input = typeof args === 'string' ? { question: args } : (args || {})
const question = input.question
if (!question) {
  throw new Error('judge-research: provide a question via args (a string, or { question })')
}
const orchestrator = input.orchestrator === 'fable' ? 'fable' : 'opus'

const PLAN_SCHEMA = {
  type: 'object',
  required: ['subtopics', 'codeAreas'],
  properties: {
    subtopics: { type: 'array', items: { type: 'string' }, description: '3-6 web-research subtopics' },
    codeAreas: { type: 'array', items: { type: 'string' }, description: '2-4 local code areas to study' },
  },
}

const FINDINGS_SCHEMA = {
  type: 'object',
  required: ['findings'],
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        required: ['claim', 'evidence'],
        properties: {
          claim: { type: 'string' },
          evidence: { type: 'string', description: 'quote/summary backing the claim' },
          source: { type: 'string', description: 'URL (web) or file:line (code)' },
        },
      },
    },
  },
}

const VERDICT_SCHEMA = {
  type: 'object',
  required: ['verdict', 'reason'],
  properties: {
    verdict: { type: 'string', enum: ['supported', 'refuted', 'uncertain'] },
    reason: { type: 'string' },
    confidence: { type: 'number' },
  },
}

// -- Plan -------------------------------------------------------------------
phase('Plan')
let subtopics = input.subtopics
let codeAreas = input.codeAreas
if (!subtopics || !codeAreas) {
  const plan = await agent(
    `Decompose this research question into (a) 3-6 focused WEB-research subtopics and ` +
      `(b) 2-4 LOCAL code areas worth studying to answer it. Question:\n${question}`,
    { model: orchestrator, phase: 'Plan', label: 'plan', schema: PLAN_SCHEMA },
  )
  subtopics = subtopics || plan.subtopics
  codeAreas = codeAreas || plan.codeAreas
}
log(`Planned ${subtopics.length} web subtopics + ${codeAreas.length} code areas`)

// -- Research (fan-out) -----------------------------------------------------
phase('Research')
const webThunks = subtopics.map((t) => () =>
  agent(
    `Web-research this subtopic using WebSearch/WebFetch. Return grounded findings, ` +
      `each with a source URL. Prefer primary/official docs.\nSubtopic: ${t}\nParent question: ${question}`,
    { model: 'haiku', phase: 'Research', label: `web:${String(t).slice(0, 36)}`, schema: FINDINGS_SCHEMA },
  ),
)
const codeThunks = codeAreas.map((a) => () =>
  agent(
    `Study this area of the LOCAL codebase (read the files). Return grounded findings, ` +
      `each with file:line evidence. Do not speculate beyond what the code shows.\nArea: ${a}\nParent question: ${question}`,
    { model: 'sonnet', phase: 'Research', label: `code:${String(a).slice(0, 36)}`, schema: FINDINGS_SCHEMA },
  ),
)
const research = (await parallel([...webThunks, ...codeThunks])).filter(Boolean)
const findings = research.flatMap((r) => r.findings || [])
log(`Collected ${findings.length} findings; judging...`)

// -- Judge (adversarial verify, per finding) --------------------------------
phase('Judge')
const judged = (
  await parallel(
    findings.map((f) => () =>
      agent(
        `You are an adversarial JUDGE. Try to REFUTE the claim below; mark it "supported" ` +
          `only if the evidence genuinely holds, "refuted" if it does not, "uncertain" if you cannot tell. ` +
          `Be skeptical of unsourced or overreaching claims.\n` +
          `Claim: ${f.claim}\nEvidence: ${f.evidence}\nSource: ${f.source || 'n/a'}`,
        { model: 'opus', phase: 'Judge', label: `judge:${String(f.claim).slice(0, 36)}`, schema: VERDICT_SCHEMA },
      ).then((v) => ({ ...f, ...v })),
    ),
  )
).filter(Boolean)
const supported = judged.filter((f) => f.verdict === 'supported')
const rejected = judged.filter((f) => f.verdict !== 'supported')
log(`Judge: ${supported.length} supported, ${rejected.length} refuted/uncertain`)

// -- Synthesize -------------------------------------------------------------
phase('Synthesize')
const report = await agent(
  `Write a concise, cited research report answering the question. Use ONLY the judge-supported ` +
    `findings as load-bearing claims (cite each source). Add a short "Caveats" section for the ` +
    `refuted/uncertain findings so the reader knows what did NOT hold up.\n\n` +
    `Question: ${question}\n\n` +
    `SUPPORTED:\n${JSON.stringify(supported, null, 2)}\n\n` +
    `REFUTED/UNCERTAIN:\n${JSON.stringify(rejected, null, 2)}`,
  { model: orchestrator, phase: 'Synthesize', label: 'synthesize' },
)

return {
  question,
  counts: { findings: findings.length, supported: supported.length, rejected: rejected.length },
  report,
  supported,
  rejected,
}
