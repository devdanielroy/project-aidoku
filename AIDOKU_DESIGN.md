# Aidoku (愛読) — Design Plan v0.1

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
2. **Clean** — strip Gutenberg boilerplate/headers, normalize whitespace/encoding, and trim to just the actual novel content using catalog-supplied first/last-line anchors (see §7d, §7e).
3. **Segment** (Stage A) — deterministic sentence segmentation, no LLM involved; see §3a.
4. **Group into chunks** (Stage B) — LLM groups sentences into ~240±60 char chunks; see §3a.
5. **Grade** — tag each book (and possibly each chunk) with a rough difficulty level, so users can self-select sensibly. Even without formal level testing, a per-book level tag is needed for the "pick by level" flow to mean anything.
6. **Generate questions** (AI, offline, per chunk):
   - Vocab question — pick a keyword/high-value word from the chunk, generate a question testing it.
   - Grammar question — identify a grammar pattern present in the chunk, generate a question testing it.
   - Comprehension question — generate a question testing whether the chunk's content/meaning was understood.
7. **Generate breakdown** (AI, offline, per chunk) — full explanation of the passage: vocab, grammar, meaning, in Japanese.
8. **QA pass** — manual review before publishing a book. This is the step that makes "AI-generated" defensible: nothing reaches a paying user unreviewed, at least at v1 scale.
9. **Publish** — chunk + 3 questions + breakdown become an immutable served unit, keyed to (book_id, chunk_index).

All users reading the same book see the same chunks, questions, and breakdowns. Zero marginal inference cost per learner after the pipeline runs once.

**Current status** (stages 3–4 were previously bundled together as a single "Chunk" item, which is what made stage 3 easy to lose track of — split apart here to match how they're actually built: two separate Go packages with very different characteristics, one deterministic and one LLM-based):

| # | Stage | Package | Status |
|---|---|---|---|
| 1 | Ingest | `internal/ingest` | ✅ built, run for real against the live Gutenberg source |
| 2 | Clean (+ Trim) | `internal/clean`, `internal/catalog` | ✅ built, run for real — see §7d, §7e |
| 3 | Segment (Stage A) | `internal/segment` | ✅ built, tested, run for real via `cmd/ingest` — see §7d |
| 4 | Group into chunks (Stage B) | `internal/chunk` | ✅ built & tested against fakes only — never called against the real Claude API or a real book |
| 5 | Grade | — | ❌ not built |
| 6 | Generate questions | `internal/question` | ✅ built & tested against fakes only — never called against the real Claude API |
| 7 | Generate breakdown | — | ❌ not built |
| 8 | QA pass | — | manual process, not code |
| 9 | Publish | — | ❌ not built — no storage layer exists yet |

## 3a. Chunking Design (Book Processor)

*Design detail for pipeline stages 3 (Segment) and 4 (Group into chunks) above.*

**Hard rule: a sentence must never be split across two chunks.** Chunk length target (~240 chars ±60 for English) is a soft target only — actual chunk length varies to respect sentence boundaries. A single long sentence becomes its own chunk even if it blows past the target range.

**Two-stage design, splitting "segment sentences" from "group into chunks" so the LLM never touches or reproduces raw text:**

**Stage A — Sentence segmentation (deterministic, no LLM).**
- Proper sentence tokenizer (not naive punctuation splitting — must correctly handle abbreviations, nested quotes, ellipses, dialogue punctuation).
- **Dialogue tags do not break a sentence.** `"Wait," she said, "I don't understand."` is one sentence, not two/three, regardless of internal quote/tag structure.
- **A paragraph break always forces a boundary, regardless of terminal punctuation.** Added after real-book testing: a Gutenberg `[Illustration: ...]` placeholder (see §7d) has no sentence-ending punctuation of its own, so without this rule it silently glues onto whatever real sentence follows it. Generally correct beyond that one case too — a sentence never legitimately spans a paragraph break — and a no-op for normal prose, which almost always ends a paragraph with real terminal punctuation anyway.
- Output: ordered list of sentences, each with exact original text (untouched) and character count.
- Zero fidelity risk — no LLM in the loop, so nothing to verify against corruption at this stage.

**Stage B — Chunk grouping (LLM, indices only, never reproduces text).**
- Feed the LLM the ordered list of sentences (with lengths) for a window of the book.
- LLM returns **grouping indices only** — e.g. "sentences 1–3 → chunk 1, sentence 4 → chunk 2, sentences 5–7 → chunk 3" — targeting ~240 chars per group.
- Model never outputs the sentence text itself, only indices — this eliminates the paraphrasing/text-corruption risk entirely (vs. an earlier delimiter-insertion approach that was rejected for this reason).
- **Verification:** confirm returned indices form a valid, complete, ordered partition of the sentence list (no gaps, no overlaps, no reordering). If invalid, reject and retry, or fall back to a simple greedy rule-based grouper (accumulate sentences until adding the next would exceed the target+tolerance, then cut).
- Chunks are reconstructed by concatenating sentences per group — guaranteed byte-identical to source text since Stage A output was untouched.
- **Dialogue-turn atomicity (open question — not yet a hard rule).** A single dialogue turn — consecutive sentences spoken by one character inside one unbroken quotation, e.g. Dracula's "I am Dracula...to my house." followed immediately by "Come in...rest." — is multiple *sentences* (Stage A correctly splits each one; that's unambiguous grammar), but grouping them into the *same chunk* wherever the length budget allows is a Stage B preference, so a reader isn't handed an uninterrupted quote sliced mid-flow between two chunk boundaries. Unlike the sentence rule in §3a (hard, no exceptions), this can't be an unconditional hard rule: a single character's speech can run to many sentences and far exceed even the oversized-chunk ceiling, so treating "the whole quote" as one unsplittable unit the way a sentence is unsplittable isn't tenable. Needs a concrete policy before Stage B implementation — options to weigh: (a) soft preference only, penalized in the grouping prompt/heuristic but overridable once length forces a cut; (b) a raised ceiling specifically for in-progress dialogue turns before allowing a split; (c) explicit permission to cut between sentences of a long dialogue turn once it exceeds some threshold, same QA-flagging treatment as an oversized single sentence. Decide when Stage B is designed — tracked in §7.

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
- **Backend**: Go.
- **Content pipeline**: separate offline batch process, Go (`pipeline/` module) — decided and built out over §7a–§7e, not just the backend's language by coincidence but a deliberate choice made once Stage B was first implemented.
- **AI generation**: Claude API, offline/batch, not in the user-facing request path.
- **Storage**: TBD — needs a DB for books/chunks/questions/breakdowns (relational fits well given the structure) plus user progress.

## 7. Open Questions / Decisions Needed

- **Corpus selection**: which specific books to launch with? Needs to be genuinely appealing to a Japanese English-learner (not just "public domain and easy" — has to feel worth reading), while being tractable in reading level.
- **Level tagging methodology**: how do we grade a book's difficulty without formal testing infrastructure? (Lexile-style heuristics? Vocabulary frequency analysis? Manual judgment?)
- **Chunk boundary algorithm**: exact rules for where a break is "logical" (sentence-final punctuation as primary rule, clause boundaries as fallback, hard character-count ceiling as last resort).
- **Explanation language**: confirm Japanese-language breakdowns for v1 (English learner, Japanese L1) — this also affects UI copy/marketing language decisions.
- **Monetization**: subscription vs. one-time per-book purchase vs. freemium (first book free, pay per additional book)?
- **i18n architecture**: build the engine bilingual-ready from day one (even though v1 only ships English-for-Japanese), so the v2 Japanese-for-English-speakers flip doesn't require a rebuild?
- **Stage B dialogue-turn atomicity policy**: how far should chunk grouping go to avoid splitting a multi-sentence dialogue turn across chunks, and what's the fallback once a single character's speech is too long to keep intact? See §3a for the options under consideration. Needs deciding before Stage B is implemented.
- **Illustration blocks downstream**: Clean condenses `[Illustration: ...]` placeholders into single atomic sentences and drops the caption-less `[Illustration]` form entirely (§7d), but nothing downstream knows the remaining captioned ones are special yet — as things stand they'd get grouped into a chunk and questions would get generated against them like any other sentence, which makes no sense for a caption fragment. Needs a way to flag/detect them (the condensed `[Illustration:` prefix is a cheap, reliable marker) so Stage B can give each one its own chunk and question generation can skip it entirely.
- **Chapter boundary detection**: deliberately deferred to Stage B rather than Clean/Segment — chunk grouping is already the stage tasked with understanding/comprehending the text, so it's the natural place to also detect chapter headings (real formatting is inconsistent even within one book — `CHAPTER II.` vs `CHAPTER XIII` with no trailing period vs `Chapter XLVI.` in mixed case — which favors judgment over a brittle regex anyway) and enforce the existing hard-break-at-chapters rule (§3a). Not yet designed.

## 7a. LLM Choice — Claude API

Using Claude via Anthropic's API directly (api.anthropic.com), not a self-hosted model. Anthropic doesn't release model weights, so "self-hosting Claude" isn't an option anywhere, including AWS — the real choice is only *who operates the inference infrastructure*:
- **Anthropic's API directly** — simplest integration, first access to new models. **Chosen for this project.**
- Amazon Bedrock / Claude Platform on AWS — exist for teams needing AWS-boundary data residency or AWS billing integration. Not relevant for a solo batch-processing pipeline with no compliance constraint, so added complexity isn't worth it here.

**Important:** a Claude.ai subscription (Pro/Max) is separate from API access. API calls need a separate Anthropic API account with its own pay-per-token billing, set up at console.anthropic.com / platform.claude.com. Not the same login/entitlement as the chat subscription.

Rationale for using a frontier hosted model over a self-hosted HuggingFace model: pipeline tasks (sentence grouping, grammar identification, question generation, Japanese explanation writing) are reasoning-heavy, not simple pattern-matching; this is a **one-time batch cost per book**, not per-user inference, so API cost is trivial relative to the value of a stronger first draft — since a human (you) QAs every generation before publishing, minimizing your own review/fix labor matters more than minimizing API spend.

## 7b. Implementation Handoff — Stage B Chunk Grouping (Go)

Implemented in `pipeline/internal/chunk`, using shared types from `pipeline/internal/types`. Built and unit-tested against fakes; not yet called against the real Claude API or a real book — see the status table in §3.

**Design summary:**
- Stage A (deterministic, no LLM) segments raw book text into sentences. Dialogue tags do not break a sentence.
- Stage B (LLM) receives the ordered sentence list for a window of the book and returns **grouping indices only** — never reproduces sentence text — eliminating paraphrasing/corruption risk entirely.
- Output is validated as a complete, ordered, non-overlapping partition before being trusted. Invalid output triggers a retry, then a deterministic greedy-grouping fallback (`GreedyGroup`).
- Target chunk length ~240 chars ±60 (English), soft target only — a sentence is never split across chunks, so actual length varies.
- Oversized chunks (single long sentence or run of sentences) are flagged for manual QA review, not auto-split.
- Book-length text is processed in overlapping windows (e.g. ~3,000 chars with overlap) to fit context limits; chapter/section breaks from source are always forced hard breaks regardless of LLM grouping (not yet implemented — chapter detection is deferred to this stage, see §7).

**Types** (`types.SentenceInput`, `types.ChunkGroup`, `types.ChunkGroupingResponse` — see `internal/types`):
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

**Function shape** (as actually built — a method on `Grouper`, not a free function, so `Client`/`Model`/`Logger` etc. are configurable per instance):
```go
type Grouper struct {
    Client    llmCaller // satisfied by *anthropic.Client; consumer-defined interface for testability
    Model     string
    MaxTokens int
    Logger    *log.Logger
}

func (g *Grouper) GroupSentencesIntoChunks(ctx context.Context, sentences []types.SentenceInput) (types.ChunkGroupingResponse, error) {
    // 1. Build prompt: system instructions + JSON-encoded sentence list
    // 2. Call Anthropic Messages API, request JSON-only output (no prose)
    // 3. Unmarshal response into ChunkGroupingResponse
    // 4. Validate via ValidatePartition
    // 5. If invalid: retry once, then fall back to GreedyGroup
    // 6. Return validated grouping; log fallback events for review
}

func ValidatePartition(sentences []types.SentenceInput, resp types.ChunkGroupingResponse) error {
    // - union of all sentence_indices == full range [0, len(sentences))
    // - no index appears twice
    // - indices within each chunk are contiguous and in order
    // - chunks themselves are in ascending order
}
```

**API call notes:**
- Uses `pipeline/internal/anthropic`, a minimal stdlib-only (`net/http`) Messages API client — deliberately no third-party SDK dependency (see §7a).
- System prompt fixes: target chunk length, sentence-boundary-only rule, dialogue-as-single-sentence rule, and the exact JSON output schema, with an explicit instruction to output JSON only, no surrounding prose.
- Requires a separate Anthropic API key (see §7a) — set as an environment variable, not committed to source.

**`BuildChunks`** (also in `internal/chunk`): the missing link between Stage A and Stage B's output — reconstructs each chunk's actual `types.Chunk` (text + char count) by concatenating its sentences per `ChunkGroup`, guaranteed byte-identical to the source since Stage A's sentence text was never rewritten. Used by §7c's question generation, which needs real chunk text, not just grouping indices.

**Not yet designed:** breakdown generation call, book-level grading/leveling step. Same general request/response/validation pattern should extend to these.

## 7c. Implementation Handoff — Question Generation (Go)

Implemented in `pipeline/internal/question`. Takes a `types.Chunk` (see §7b's `BuildChunks`, which reconstructs chunk text from Stage A sentences + Stage B's grouping) and produces its three questions — vocab, grammar, comprehension — in one LLM call, matching the shape already validated in the Flutter mock content (`app/assets/mock/*.json`): `prompt`, `options` (exactly 4, Japanese), `answer_index`, `explanation` (Japanese), and `highlight` (vocab/grammar only — the exact substring of the chunk's text underlined in the passage instead of being re-quoted in the prompt; see §2/§4 and the app's `ReadingView`).

**Deliberate deviation from §7b's stated pattern: no rule-based fallback.** Stage B can fall back to a deterministic greedy grouper because chunk grouping is mechanical — a rule-based approximation is genuinely adequate. Writing a good question is not mechanical; there's no reasonable non-LLM fallback for it. So `GenerateQuestions` retries a few times and, if it still can't produce a valid set, returns an error instead of publishing something low-quality — consistent with §7a's "nothing reaches a paying user unreviewed": a chunk that fails generation should be flagged for manual regeneration/QA, not silently patched over with a worse fallback.

## 7d. Implementation Handoff — Ingest & Clean (Go)

Implemented in `pipeline/internal/ingest` and `pipeline/internal/clean` — pipeline steps 1–2 (§3). Both deliberately have no LLM involvement (mechanical fetch/text-cleanup, nothing to reason about).

**Ingest** (`ingest.Client.FetchText`): a minimal stdlib `net/http` GET, mirroring the Anthropic client's shape (injectable `*http.Client` for testing via `httptest`, no third-party HTTP dependency). Returns the raw text completely unmodified.

**Clean** (`clean.Clean`):
- Strips everything outside Project Gutenberg's own `*** START/END OF ... PROJECT GUTENBERG EBOOK ***` markers (handles both the "THE" and "THIS" phrasing Gutenberg has used across eras). Returns an error rather than guessing if either marker is missing — an unrecognized format should surface for a human, not silently pass unknown boilerplate (or book content) through.
- Normalizes line endings (CRLF/CR → LF) and strips a leading UTF-8 BOM.
- **Dewraps hard-wrapped lines.** Gutenberg plain text is fixed-width-wrapped (~70 columns) with single newlines *within* a paragraph and a blank line *between* paragraphs. Left alone, that wrapping ends up baked into individual sentences verbatim once Stage A segments the text (e.g. a sentence containing a literal `"...in\nuncommon advantage..."`). Clean joins each paragraph's wrapped lines into one flowing line (trimming each line first, so indentation from centered/title-page text doesn't leave a gap mid-sentence) and normalizes paragraph separators to exactly one blank line.
- **Condenses illustration placeholders — or drops them, if they carry no content.** Gutenberg's convention for an image the plain text can't show is `[Illustration: <caption>]`, sometimes with a nested `[_Copyright ... _]` notice — e.g. `[Illustration:\n\n"He came down to see the place"\n\n[_Copyright 1894 by George Allen._]]`. `condenseIllustrations` finds each one (bracket-depth-aware matching, not just "up to the next `]`" — the nested copyright bracket would end the match early otherwise) and collapses its internal whitespace to one line: `[Illustration: "He came down to see the place" [_Copyright 1894 by George Allen._]]`. Nothing is stripped here — the caption/copyright text is kept in case a future feature shows the actual illustration (sourced separately) using this text to identify which one. Combined with Stage A's new paragraph-break rule (§3a), each condensed block becomes one atomic sentence, cleanly separated from the real prose around it — previously (with neither fix) it silently glued onto whatever sentence followed it, since it has no terminal punctuation of its own. A bare `[Illustration]` marker with no caption at all (a second, simpler convention this edition also uses) is instead removed outright by `removeBareIllustrations` — there's no identifying text worth keeping, so unlike the captioned form there's nothing to preserve. Downstream stages treating the remaining (captioned) illustration sentences as skippable (no question generation, etc.), and detecting chapter boundaries, are both deliberately deferred to Stage B once it's built — chunk grouping is already the stage tasked with understanding the text, so it's the natural place to reason about structure like this too, rather than teaching Clean/Segment ad hoc rules for it. See §7.
- **Emphasis underscores (`_word_`, Gutenberg's plain-text italics convention) are left untouched.** An earlier version of Clean converted them into unambiguous `__BGNCHNK__`/`__ENDCHNK__` markers so a later app-side renderer could pick them up reliably. Reverted after looking at real output on the full book — the markers made the text noticeably harder to read at this stage of the pipeline for no immediate benefit, since nothing downstream consumes them yet. Underscores just pass through as ordinary characters for now, same as any other punctuation Clean doesn't touch. Worth revisiting once there's an actual consumer (the app rendering emphasis) to justify the noise.

**Deliberate scope limit:** Clean itself does not attempt to strip front matter *within* the book — title pages, prefaces, tables of contents — that some Gutenberg editions include before the actual novel text begins. The Pride and Prejudice testdata fixture used in this package's tests (`pipeline/testdata/pg1342.txt`, the real Gutenberg ebook #1342 plain text) is a concrete example: it's an illustrated edition with a ~30KB preface by George Saintsbury before "Chapter I." Detecting that generically (vs. Gutenberg's own consistent START/END wrapper) is a much harder, edition-specific problem — no heuristic could reliably tell "preface" from "Chapter I" across arbitrarily different books. Solved a different way instead — see §7e.

**Also surfaced by testing against the real file:** the real published opening line of Pride and Prejudice has no comma before "must" (`"...in possession of a good fortune must be in want of a wife."`) — the Flutter app's hand-typed mock content (`app/assets/mock/pride_and_prejudice.json`) added one from memory that isn't actually there. Worth reconciling the mock content separately; not fixed as part of this work.

## 7e. Implementation Handoff — Book Catalog & Trim (Go)

Front/back matter (§7d's scope limit) turned out not to be worth solving generically — instead, a human supplies two anchor lines per book, once, at catalog time. This is a natural extension of the QA pass already required before publishing (§3 step 8): picking two lines is trivial compared to the rest of that review, and it sidesteps an open-ended detection problem entirely by not trying to classify front/back matter at all, just anchoring on exact text a human already identified.

**Catalog file** (`pipeline/books.txt`, parsed by `pipeline/internal/catalog`): a plain text file, one entry per book. Each entry is exactly 3 lines — Gutenberg URL, first line of the actual novel content, last line of the actual novel content — separated from other entries by a blank line; an optional `# comment` line (e.g. the book's title) may precede an entry for human readability and is ignored by the parser. See the file's own header comment for the authoritative format spec.

- `catalog.Entry` also carries a `GutenbergID`, extracted from the URL (handles the `/cache/epub/{id}/`, `/files/{id}/`, and `/ebooks/{id}` URL shapes Gutenberg serves plain text under). This is the stable identifier for a book — unlike the URL itself, which Gutenberg serves in several equivalent forms across different paths and mirrors, so the same book fetched two different ways wouldn't naturally dedup on URL alone. Earmarked as the future key for skipping books a storage layer already has, and a natural fit for the `Book` entity's `id` in §4 once that layer exists.
- Malformed entries (not exactly 3 lines, a first line that doesn't look like a URL, a URL with no extractable Gutenberg ID) are parse errors, not skipped/guessed.

**Trim** (`clean.Trim(text, firstLine, lastLine)`): slices Clean's output down to the span between the two anchors, inclusive — discarding everything outside it, whatever it is or whatever it's called. Both anchors must match **exactly once** in the text; zero or multiple matches is an error (wrong edition, a typo, or an anchor that isn't specific enough to be a safe cut point), not a guess. Kept as a separate function from `Clean` rather than folded into it: stripping Gutenberg's own wrapper is fully automatic and identical for every Gutenberg book, while trimming to the real content bounds is book-specific and needs the human-supplied anchor — deliberately not conflating a generic step with a per-book one.

**Orchestration** (`pipeline.PrepareBook`, in `pipeline/internal/pipeline`): `Fetch → Clean → Trim` for one `catalog.Entry`, returning the final novel-only text. `cmd/ingest` is the runnable entrypoint — reads `books.txt`, calls `PrepareBook` for every entry, then also runs Stage A (`segment.Segment`) on the result and writes two output files per book to `pipeline/books/` (gitignored — generated, not source): `{id}.txt` (the prepared novel text) and `{id}.sentences.txt` (its sentences, one per line, indexed with char counts, for human inspection — not a format anything downstream parses; Stage B takes `[]types.SentenceInput` directly, in memory). Continues past a single book's failure rather than aborting the whole run, but exits non-zero if anything failed. Stage B (LLM chunk grouping) and beyond are not run by this command.

Run for real against the live catalog as part of this work (not just tested against fakes): `go run ./cmd/ingest` fetched the real Pride and Prejudice text over the network. Output contained the correct opening/closing lines with zero preface, colophon, or Gutenberg-wrapper content remaining, and — after the illustration-condensing, bare-illustration-removal, and paragraph-break fixes above — 5,939 correctly-bounded sentences with no prose glued to an illustration placeholder.

## 8. Suggested Next Step

Scope v0: the smallest version that proves the full loop works end to end — one short public domain book, fully piped through (chunked, graded, questioned, broken down, QA'd), servable through a bare-bones Flutter client. Prove the pipeline and the pedagogy before building out level tagging, gamification, or monetization.

