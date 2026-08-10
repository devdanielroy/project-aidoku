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

- [`pipeline/`](./pipeline) — offline Go module that turns a raw public-domain
  text into published chunks, questions, and breakdowns. Currently
  implements Stage A (deterministic sentence segmentation); Stage B (LLM
  chunk grouping) and later stages (grading, question generation, breakdown
  generation) are next.

Backend (Go, content-serving API) and frontend (Flutter client) will be
added as separate top-level directories once the pipeline proves the loop
end to end (see design doc §8).

## Status

Pre-v0. Nothing is published yet — the goal right now is one short book
piped fully through chunking → grading → questions → breakdown → QA.
