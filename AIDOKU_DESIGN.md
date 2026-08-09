# Aidoku (架読) — Design Plan v0.1

## 1. Concept

A language-learning app that teaches through literature instead of graded textbook sentences. Users read real (public domain) books one small chunk at a time. Each chunk must be read unassisted first, then tested with three short questions, then unpacked with a full breakdown before moving on. The goal is to build both language competence and genuine excitement for reading in the target language — not just vocabulary drilling.

**v1 target market:** Japanese speakers learning English. Sold in Japan, where English-learning spend is high and well-established (English Buffet is proof of willingness to pay). Japanese-learners-of-English-via-literature is also a far less crowded niche than generic English apps.

**v2 (later):** Same engine, reversed — English speakers learning Japanese.

## 2. Core Loop

1. User picks a book, filtered/tagged by rough difficulty level (self-selected, not tested).
2. App serves the next chunk of that book (~240 characters ± 60, English; boundary-aware — never mid-sentence/clause if avoidable).
3. User reads the chunk unassisted. No dictionary, no hints, no translation visible.
4. User answers 3 questions tied to that chunk:
   - **Vocabulary** — a keyword or notable word from the chunk.
   - **Grammar** — a grammar pattern/structure used in the chunk.
   - **Comprehension** — did they understand what actually happened/was said.
5. After answering, the app reveals a full breakdown of the passage (vocab notes, grammar notes, meaning, maybe cultural/stylistic notes) — explained in Japanese, since the learner's L1 is Japanese and grammar explanation lands better in L1 for this level.
6. Move to next chunk. Repeat until book complete.

This is deliberately slow and effortful by design — the friction (forced unaided read, forced retrieval before help) is the pedagogy, not a UX flaw to smooth away.

## 3. Content Pipeline (the core IP)

All content is **pre-generated, not generated on-the-fly**. One-time processing cost per book; infinite reuse across all users reading that book. This is the main technical/cost advantage of the whole product.

**Pipeline stages:**
1. **Ingest** — pull a public domain text (source: Project Gutenberg for English; unambiguous copyright status).
2. **Clean** — strip Gutenberg boilerplate/headers, normalize whitespace/encoding.
3. **Chunk** — see §3a, two-stage sentence-safe chunking.
4. **Grade** — tag each book (and possibly each chunk) with a rough difficulty level, so users can self-select sensibly. Even without formal level testing, a per-book level tag is needed for the "pick by level" flow to mean anything.
5. **Generate questions** (AI, offline, per chunk):
   - Vocab question — pick a keyword/high-value word from the chunk, generate a question testing it.
   - Grammar question — identify a grammar pattern present in the chunk, generate a question testing it.
   - Comprehension question — generate a question testing whether the chunk's content/meaning was understood.
6. **Generate breakdown** (AI, offline, per chunk) — full explanation of the passage: vocab, grammar, meaning, in Japanese.
7. **QA pass** — manual review before publishing a book. This is the step that makes "AI-generated" defensible: nothing reaches a paying user unreviewed, at least at v1 scale.
8. **Publish** — chunk + 3 questions + breakdown become an immutable served unit, keyed to (book_id, chunk_index).

All users reading the same book see the same chunks, questions, and breakdowns. Zero marginal inference cost per learner after the pipeline runs once.

## 3a. Chunking Design (Book Processor)

**Hard rule: a sentence must never be split across two chunks.** Chunk length target (~240 chars ±60 for English) is a soft target only — actual chunk length varies to respect sentence boundaries. A single long sentence becomes its own chunk even if it blows past the target range.

**Two-stage design, splitting "segment sentences" from "group into chunks" so the LLM never touches or reproduces raw text:**

**Stage A — Sentence segmentation (deterministic, no LLM).**
- Proper sentence tokenizer (not naive punctuation splitting — must correctly handle abbreviations, nested quotes, ellipses, dialogue punctuation).
- **Dialogue tags do not break a sentence.** `"Wait," she said, "I don't understand."` is one sentence, not two/three, regardless of internal quote/tag structure.
- Output: ordered list of sentences, each with exact original text (untouched) and character count.
- Zero fidelity risk — no LLM in the loop, so nothing to verify against corruption at this stage.

**Stage B — Chunk grouping (LLM, indices only, never reproduces text).**
- Feed the LLM the ordered list of sentences (with lengths) for a window of the book.
- LLM returns **grouping indices only** — e.g. "sentences 1–3 → chunk 1, sentence 4 → chunk 2, sentences 5–7 → chunk 3" — targeting ~240 chars per group.
- Model never outputs the sentence text itself, only indices — this eliminates the paraphrasing/text-corruption risk entirely (vs. an earlier delimiter-insertion approach that was rejected for this reason).
- **Verification:** confirm returned indices form a valid, complete, ordered partition of the sentence list (no gaps, no overlaps, no reordering). If invalid, reject and retry, or fall back to a simple greedy rule-based grouper (accumulate sentences until adding the next would exceed the target+tolerance, then cut).
- Chunks are reconstructed by concatenating sentences per group — guaranteed byte-identical to source text since Stage A output was untouched.

**Book-length handling:** process in overlapping windows (not the whole book in one pass) to respect context limits — e.g. ~3,000-character windows with overlap, reconciling boundary decisions near window edges using the pass that has fuller context. Chapter/section breaks from the source are always forced hard breaks, regardless of Stage B's grouping.

**QA flagging (not auto-splitting):** if a chunk exceeds a defined ceiling (e.g. 500 characters) due to a single long sentence or run of sentences, flag for manual review rather than silently allowing it — gives visibility into books (e.g. Victorian-era long-sentence prose) that may produce a lot of oversized chunks.

## 4. Data Model (rough sketch)

- **Book**: id, title, author, source_url (Gutenberg), level_tag, language, status (processing/published)
- **Chunk**: id, book_id, index (ordering), text, char_count
- **Question**: id, chunk_id, type (vocab/grammar/comprehension), prompt, options/answer, explanation
- **Breakdown**: id, chunk_id, content (Japanese explanation text)
- **UserProgress**: user_id, book_id, current_chunk_index, answers_history, streak/gamification state

## 5. Gamification (light touch, TBD in detail)

- Streaks for daily reading.
- Progress bar per book (chunks completed / total).
- Possibly badges for finishing a book, or accuracy milestones.
- Deliberately not central to the pitch — the book itself and the "I can read real English literature" feeling is the hook, gamification is scaffolding around it, not the point.

## 6. Tech Stack

- **Frontend**: Flutter (single codebase, iOS/Android).
- **Backend**: Go (new language for ダニエル — deliberate learning goal, also a reasonable fit for a straightforward CRUD + content-serving API).
- **Content pipeline**: separate offline batch process (language TBD — Python is the pragmatic choice given AI/text-processing tooling, but not yet decided; doesn't need to match the backend language).
- **AI generation**: Claude API, offline/batch, not in the user-facing request path.
- **Storage**: TBD — needs a DB for books/chunks/questions/breakdowns (relational fits well given the structure) plus user progress.

## 7. Open Questions / Decisions Needed

- **Corpus selection**: which specific books to launch with? Needs to be genuinely appealing to a Japanese English-learner (not just "public domain and easy" — has to feel worth reading), while being tractable in reading level.
- **Level tagging methodology**: how do we grade a book's difficulty without formal testing infrastructure? (Lexile-style heuristics? Vocabulary frequency analysis? Manual judgment?)
- **Chunk boundary algorithm**: exact rules for where a break is "logical" (sentence-final punctuation as primary rule, clause boundaries as fallback, hard character-count ceiling as last resort).
- **Explanation language**: confirm Japanese-language breakdowns for v1 (English learner, Japanese L1) — this also affects UI copy/marketing language decisions.
- **Monetization**: subscription vs. one-time per-book purchase vs. freemium (first book free, pay per additional book)?
- **i18n architecture**: build the engine bilingual-ready from day one (even though v1 only ships English-for-Japanese), so the v2 Japanese-for-English-speakers flip doesn't require a rebuild?

## 7a. LLM Choice — Claude API

Using Claude via Anthropic's API directly (api.anthropic.com), not a self-hosted model. Anthropic doesn't release model weights, so "self-hosting Claude" isn't an option anywhere, including AWS — the real choice is only *who operates the inference infrastructure*:
- **Anthropic's API directly** — simplest integration, first access to new models. **Chosen for this project.**
- Amazon Bedrock / Claude Platform on AWS — exist for teams needing AWS-boundary data residency or AWS billing integration. Not relevant for a solo batch-processing pipeline with no compliance constraint, so added complexity isn't worth it here.

**Important:** a Claude.ai subscription (Pro/Max) is separate from API access. API calls need a separate Anthropic API account with its own pay-per-token billing, set up at console.anthropic.com / platform.claude.com. Not the same login/entitlement as the chat subscription.

Rationale for using a frontier hosted model over a self-hosted HuggingFace model: pipeline tasks (sentence grouping, grammar identification, question generation, Japanese explanation writing) are reasoning-heavy, not simple pattern-matching; this is a **one-time batch cost per book**, not per-user inference, so API cost is trivial relative to the value of a stronger first draft — since a human (you) QAs every generation before publishing, minimizing your own review/fix labor matters more than minimizing API spend.

## 7b. Implementation Handoff — Stage B Chunk Grouping (Go)

This is the first pipeline stage ready for implementation. Language: Go.

**Design summary:**
- Stage A (deterministic, no LLM) segments raw book text into sentences. Dialogue tags do not break a sentence.
- Stage B (LLM) receives the ordered sentence list for a window of the book and returns **grouping indices only** — never reproduces sentence text — eliminating paraphrasing/corruption risk entirely.
- Output is validated as a complete, ordered, non-overlapping partition before being trusted. Invalid output triggers a retry, then a deterministic greedy-grouping fallback.
- Target chunk length ~240 chars ±60 (English), soft target only — a sentence is never split across chunks, so actual length varies.
- Oversized chunks (single long sentence or run of sentences) are flagged for manual QA review, not auto-split.
- Book-length text is processed in overlapping windows (e.g. ~3,000 chars with overlap) to fit context limits; chapter/section breaks from source are always forced hard breaks regardless of LLM grouping.

**Types:**
```go
type SentenceInput struct {
    Index     int    `json:"index"`
    Text      string `json:"text"`
    CharCount int    `json:"char_count"`
}

type ChunkGroup struct {
    ChunkIndex      int   `json:"chunk_index"`
    SentenceIndices []int `json:"sentence_indices"`
}

type ChunkGroupingResponse struct {
    Chunks []ChunkGroup `json:"chunks"`
}
```

**Function shape:**
```go
func GroupSentencesIntoChunks(sentences []SentenceInput) (ChunkGroupingResponse, error) {
    // 1. Build prompt: system instructions + JSON-encoded sentence list
    // 2. Call Anthropic Messages API, request JSON-only output (no prose)
    // 3. Unmarshal response into ChunkGroupingResponse
    // 4. Validate via ValidatePartition
    // 5. If invalid: retry once, then fall back to greedy rule-based grouper
    // 6. Return validated grouping; log fallback events for review
}

func ValidatePartition(sentences []SentenceInput, resp ChunkGroupingResponse) error {
    // - union of all sentence_indices == full range [0, len(sentences))
    // - no index appears twice
    // - indices within each chunk are contiguous and in order
    // - chunks themselves are in ascending order
}
```

**API call notes:**
- Use Anthropic's Messages API (Go: plain HTTP client or an unofficial Go SDK — no first-party Anthropic Go SDK as of this writing, worth confirming current status before building).
- System prompt should fix: target chunk length, sentence-boundary-only rule, dialogue-as-single-sentence rule, and the exact JSON output schema, with an explicit instruction to output JSON only, no surrounding prose.
- Requires a separate Anthropic API key (see §7a) — set as an environment variable, not committed to source.

**Not yet designed (next stages after this one):** vocab/grammar/comprehension question generation calls, breakdown generation call, book-level grading/leveling step. Same general pattern (structured JSON request/response, validation, fallback) should extend to these.

## 8. Suggested Next Step

Scope v0: the smallest version that proves the full loop works end to end — one short public domain book, fully piped through (chunked, graded, questioned, broken down, QA'd), servable through a bare-bones Flutter client. Prove the pipeline and the pedagogy before building out level tagging, gamification, or monetization.

