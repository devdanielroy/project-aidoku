# Aidoku (愛読)

> **Working name.** "Aidoku" is already used by an existing manga-reading
> iOS app, so this name will change before any public release. Treat it as
> a placeholder throughout the codebase for now.

A language-learning app that teaches through literature: users read small,
sentence-safe chunks of real public-domain books unassisted, answer three
short questions per chunk (vocab / grammar / comprehension), then get a full
breakdown before moving on. v1 targets Japanese speakers learning English.

See [AIDOKU_DESIGN.md](./AIDOKU_DESIGN.md) for the full design plan.

## Layout

- [`pipeline/`](./pipeline) — offline Go module implementing the content
  pipeline: ingest, clean (+ book catalog, trim), sentence segmentation,
  LLM chunk grouping, question generation. See
  [AIDOKU_DESIGN.md §3](./AIDOKU_DESIGN.md) for the full stage-by-stage
  design and a per-stage build/test status table.
- [`app/`](./app) — Flutter client (macOS target so far). Currently a
  vertical-slice prototype of the full core loop (library → read →
  questions → breakdown) running on hand-authored mock content, not yet
  wired to the real pipeline.

A content-serving backend (Go) and a storage layer are not started yet —
see [Milestones](#milestones).

## Milestones

**Completed**
- [x] Design plan drafted — [AIDOKU_DESIGN.md](./AIDOKU_DESIGN.md)
- [x] Pipeline: sentence segmentation (Stage A) — deterministic, no LLM; run against the real book
- [x] Pipeline: LLM chunk grouping (Stage B) — built and tested against fakes; not yet run against the real Claude API
- [x] Pipeline: question generation (vocab/grammar/comprehension) — built and tested against fakes; not yet run against the real Claude API
- [x] Pipeline: ingest, clean, book catalog, trim — run for real against Project Gutenberg; a real book (Pride and Prejudice) fully cleaned end to end, front/back matter and illustration placeholders handled
- [x] Flutter app: mock vertical slice of the full core loop, verified running natively on macOS

**Next up**
- [ ] Run chunk grouping and question generation against the real Claude API for the first time
- [ ] Breakdown generation stage (not yet designed)
- [ ] Book grading/leveling stage (not yet designed)
- [ ] Storage layer + publish stage (no DB yet)
- [ ] Wire real pipeline output into the Flutter app, replacing the hand-authored mock content
- [ ] Chapter boundary detection (deliberately deferred to the chunk-grouping stage — see design doc §7)
- [ ] Pick a real product name (see the working-name note above)

See [AIDOKU_DESIGN.md §3](./AIDOKU_DESIGN.md) for the detailed per-stage status table, and §7 for open design questions.
