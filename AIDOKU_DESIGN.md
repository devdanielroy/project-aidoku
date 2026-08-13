# Aidoku (愛読) — Design Plan v0.1

## 1. Concept

A language-learning app that teaches through literature instead of graded textbook sentences. Users read real (public domain) books one small chunk at a time. Each chunk must be read unassisted first, then tested with three short questions, then unpacked with a full breakdown before moving on. The goal is to build both language competence and genuine excitement for reading in the target language — not just vocabulary drilling.

**v1 target market:** Japanese speakers learning English. Sold in Japan, where English-learning spend is high and well-established (English Buffet is proof of willingness to pay). Japanese-learners-of-English-via-literature is also a far less crowded niche than generic English apps.

**v2 (later):** Same engine, reversed — English speakers learning Japanese.

## 2. Core Loop

1. User picks a book, filtered/tagged by rough difficulty level (self-selected, not tested).
2. App serves the next chunk of that book (~240 characters ± 60, English; boundary-aware — never mid-sentence/clause if avoidable).
3. User reads the chunk unassisted.
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
3. **Segment** (Stage A) — deterministic sentence segmentation; see §3a.
4. **Claude API Invocation Steps** Send sentences to Claude API and request the following:
    1. **Group into chunks** — Group sentences into ~240±60 char chunks; see §3a.
    2. **Generate questions** (AI, offline, per chunk):
        - Vocab question — pick a keyword/high-value word from the chunk, generate a question testing it.
        - Grammar question — identify a grammar pattern present in the chunk, generate a question testing it.
        - Comprehension question — generate a question testing whether the chunk's content/meaning was understood.
    3. **Generate breakdown** full explanation of the passage: vocab, grammar, meaning, in Japanese.
5. **QA pass** — manual review before publishing a book. This is the step that makes "AI-generated" defensible: nothing reaches a paying user unreviewed, at least at v1 scale.
6. **Publish** — chunk + 3 questions + breakdown become an immutable served unit, keyed to (book_id, chunk_index).

Book grading (assigning each book a reading-comprehension level, 1–10, see the README's Reading Levels table) is decided by Daniel by weighing vocabulary difficulty, sentence complexity, and how it maps to TOEIC/CEFR. The level is recorded directly in the catalog as the book's `Level=` line. See §7e.

All users reading the same book see the same chunks, questions, and breakdowns. Zero marginal inference cost per learner after the pipeline runs once.

**Current status** (stages 3–4 were previously bundled together as a single "Chunk" item, which is what made stage 3 easy to lose track of — split apart here to match how they're actually built: two separate Go packages with very different characteristics, one deterministic and one LLM-based):

| # | Stage | Package | Status |
|---|---|---|---|
| 1 | Ingest | `internal/ingest` | ✅ built, run for real against the live Gutenberg source |
| 2 | Clean (+ Trim) | `internal/clean`, `internal/catalog` | ✅ built, run for real — see §7d, §7e |
| 3 | Segment (Stage A) | `internal/segment` | ✅ built, tested, run for real via `cmd/ingest` — see §7d |
| 4.1 | Group into chunks (Stage B) | `internal/chunk` | ✅ built, tested against fakes, and now run against the real Claude API (small test batch, not a full book yet) |
| 4.2 | Generate questions | `internal/question` | ✅ built, tested against fakes, and now run against the real Claude API (small test batch, not a full book yet) |
| 4.3 | Generate breakdown | `internal/breakdown` | ✅ built & tested against fakes — not yet called against the real Claude API |
| 5 | QA pass | — | manual process, not code |
| 6 | Publish | — | ❌ not built — no storage layer exists yet |

## 3a. Chunking Design (Book Processor)

*Design detail for pipeline stage 3 (Segment) and stage 4.1 (Group into chunks) above.*

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

**Book-length handling:** process in windows (not the whole book in one pass) to respect context limits — `chunk.SplitIntoWindows` (§7h) implements this now, targeting ~3,000 characters per window. Two refinements described here aren't implemented yet: window **overlap**, with boundary decisions near window edges reconciled using the pass that has fuller context (current windows are plain, non-overlapping cuts — see that function's own doc comment for why this is an acceptable v0 gap); and forcing chapter/section breaks from the source as hard breaks regardless of Stage B's grouping (chapter detection itself is still undesigned — see §7).

**QA flagging (not auto-splitting):** if a chunk exceeds a defined ceiling (e.g. 500 characters) due to a single long sentence or run of sentences, flag for manual review rather than silently allowing it — gives visibility into books (e.g. Victorian-era long-sentence prose) that may produce a lot of oversized chunks.

## 4. Data Model (rough sketch)

- **Book**: id, title, author, source_url (Gutenberg), level (1–10, `catalog.ReadingLevel`, assigned manually — see §7e), language, status (processing/published)
- **Chunk**: id, book_id, index (ordering), text, char_count
- **Question**: id, chunk_id, type (vocab/grammar/comprehension), prompt, options/answer, explanation
- **Breakdown**: id, chunk_id, content (Japanese explanation text)
- **UserProgress**: user_id, book_id, current_chunk_index, answers_history, streak/gamification state

See `db/schema.sql` for the actual Postgres DDL (§6) — this list stays a rough sketch on purpose; the schema file is the source of truth, so the two aren't kept in lockstep line-for-line.

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
- **Storage**: Postgres. The data model (§4) is a small number of fixed-shape, foreign-keyed relations — Book → Chunk → Question, Chunk → Breakdown (1:1), User → UserProgress — with real integrity constraints worth enforcing at the DB layer (e.g. exactly one question of each type per chunk), not the variable/nested shape a document store like MongoDB is suited to. `db/schema.sql` is the actual DDL, written to by `pipeline/internal/db` — see §7f. Local development runs it via `docker compose up -d` (`docker-compose.yml`), with a named volume so pipeline output (chunks, questions, breakdowns) survives between dev sessions instead of living only in memory. No migration tool wired up yet — a single schema file is enough until there's real backend/storage-layer code driving changes to it.

## 7. Open Questions / Decisions Needed

- **Corpus selection**: which specific books to launch with? Needs to be genuinely appealing to a Japanese English-learner (not just "public domain and easy" — has to feel worth reading), while being tractable in reading level.
- ~~**Level tagging methodology**~~ — resolved: manual judgment per book, done before the pipeline runs — not a heuristic, not a pipeline stage. See §7e.
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
- Book-length text is processed in ~3,000-char windows (`chunk.SplitIntoWindows`, §7h) to fit context limits — plain, non-overlapping cuts for now, not yet the overlap-with-reconciliation design described in §3a. Chapter/section breaks from source are always forced hard breaks regardless of LLM grouping (not yet implemented — chapter detection is deferred to this stage, see §7).

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

Breakdown generation (stage 4.3) extends this same general request/response/validation pattern — see §7g. (Book-level grading is *not* part of that pattern — see §7e: it's a manual catalog field decided before the pipeline runs, not a pipeline stage or an LLM call.)

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

Front/back matter (§7d's scope limit) turned out not to be worth solving generically — instead, a human supplies two anchor lines per book, once, at catalog time. This is a natural extension of the QA pass already required before publishing (§3 stage 5): picking two lines is trivial compared to the rest of that review, and it sidesteps an open-ended detection problem entirely by not trying to classify front/back matter at all, just anchoring on exact text a human already identified.

**Catalog file** (`pipeline/books.txt`, parsed by `pipeline/internal/catalog`): a plain text file, one entry per book. Each entry is exactly 6 lines — title, author, Gutenberg URL (the plain-text edition specifically; `Clean` depends on Gutenberg's own START/END markers, which only that edition has), first line of the actual novel content, last line of the actual novel content, `Level=X` — separated from other entries by a blank line; a `# comment` line is ignored by the parser wherever it appears (a section header, or disabling an entry — no longer needed per-entry now that title/author are real fields). See the file's own header comment, or the README's "Adding a Book to the Catalog" section, for the full spec.

- `catalog.Entry.Title` and `.Author` are shown to the user as-is — the catalog's own record of them, not derived from Gutenberg metadata or the source text.
- `catalog.Entry` also carries a `GutenbergID`, extracted from the URL (handles the `/cache/epub/{id}/`, `/files/{id}/`, and `/ebooks/{id}` URL shapes Gutenberg serves plain text under). This is the stable identifier for a book — unlike the URL itself, which Gutenberg serves in several equivalent forms across different paths and mirrors, so the same book fetched two different ways wouldn't naturally dedup on URL alone. Earmarked as the future key for skipping books a storage layer already has, and a natural fit for the `Book` entity's `id` in §4 once that layer exists.
- `catalog.Entry.Level` (type `catalog.ReadingLevel`, an `int` enum `LevelInitiate`=1 through `LevelScholar`=10, matching the README's Reading Levels table) is parsed from the entry's sixth line. Book grading is deliberately **not** one of §3's numbered pipeline stages — a human decides the level per book, before the pipeline is even run, and `catalog.Parse` just validates and carries whatever was written into the catalog; there is no book-grading code anywhere else.
- Malformed entries (not exactly 6 lines, a third line that doesn't look like a URL, a URL with no extractable Gutenberg ID, a sixth line that isn't exactly `Level=X` with X in [1,10]) are parse errors, not skipped/guessed.
- `db.NewBookFromEntry` (§7f) converts a `catalog.Entry` straight into a `db.Book` for persistence — every field a `Book` needs now has a home in the catalog.

**Trim** (`clean.Trim(text, firstLine, lastLine)`): slices Clean's output down to the span between the two anchors, inclusive — discarding everything outside it, whatever it is or whatever it's called. Both anchors must match **exactly once** in the text; zero or multiple matches is an error (wrong edition, a typo, or an anchor that isn't specific enough to be a safe cut point), not a guess. Kept as a separate function from `Clean` rather than folded into it: stripping Gutenberg's own wrapper is fully automatic and identical for every Gutenberg book, while trimming to the real content bounds is book-specific and needs the human-supplied anchor — deliberately not conflating a generic step with a per-book one.

**Orchestration** (`pipeline.PrepareBook`, in `pipeline/internal/pipeline`): `Fetch → Clean → Trim` for one `catalog.Entry`, returning the final novel-only text. `cmd/ingest` is the runnable entrypoint — reads `books.txt`, calls `PrepareBook` for every entry, then also runs Stage A (`segment.Segment`) on the result and writes two output files per book to `pipeline/books/` (gitignored — generated, not source): `{id}.txt` (the prepared novel text) and `{id}.sentences.txt` (its sentences, one per line, indexed with char counts, for human inspection — not a format anything downstream parses; Stage B takes `[]types.SentenceInput` directly, in memory). Continues past a single book's failure rather than aborting the whole run, but exits non-zero if anything failed. Stage B (LLM chunk grouping) and beyond are not run by this command.

Run for real against the live catalog as part of this work (not just tested against fakes): `go run ./cmd/ingest` fetched the real Pride and Prejudice text over the network. Output contained the correct opening/closing lines with zero preface, colophon, or Gutenberg-wrapper content remaining, and — after the illustration-condensing, bare-illustration-removal, and paragraph-break fixes above — 5,939 correctly-bounded sentences with no prose glued to an illustration placeholder.

## 7f. Implementation Handoff — Storage (Go)

Implemented in `pipeline/internal/db`. Persists pipeline output — books, chunks, questions, and breakdowns — to the Postgres schema in `db/schema.sql` (§4/§6). Nothing in this package calls the Anthropic API; it's purely the write side for what earlier stages already produced.

**`Store`** wraps a `conn` interface (`Exec`/`QueryRow`/`Begin`) rather than a concrete `*pgxpool.Pool` — same consumer-defined-interface-for-testability pattern as `llmCaller` in §7b/§7c. `pgx.Tx` satisfies `conn` too (its `Begin` opens a savepoint-based nested transaction), which is what makes the package's own tests possible without a fake: each test opens a real transaction against the local dev Postgres, wraps it in a `Store`, and rolls back at the end — so tests exercise real SQL (including the schema's CHECK/UNIQUE/FOREIGN KEY constraints) without leaving rows behind or needing a mock. Tests skip (not fail) if Postgres isn't reachable, so `go test ./...` doesn't require Docker to be running.

**Write surface:**
- `UpsertBook(ctx, Book) (id int, error)` — insert or update by `GutenbergID`. `Book` is its own type, not `catalog.Entry` (Language/Status have no catalog equivalent and default instead), but `NewBookFromEntry(catalog.Entry) Book` converts one directly now that the catalog carries title/author too (§7e).
- `SaveChunk(ctx, bookID, types.Chunk, []types.Question) (id int, error)` — upserts the chunk and replaces its questions, atomically (one transaction; a chunk left with only some of its questions written is a state nothing downstream should see). Relies on the schema's constraints to reject malformed input (e.g. a vocab question with no highlight) rather than re-validating in Go — `questions` is expected to already be `question.ValidateQuestionSet`'s output.
- `SaveBreakdown(ctx, chunkID, content string) error` — upserts by `chunk_id`. Called by breakdown generation (stage 4.3, §7g) once it has content to save.

All three are upserts, not plain inserts — re-running a pipeline stage against the same book/chunk during development is expected and should overwrite, not fail on a unique-constraint violation.

**Wired into `cmd/livetest`** (§ livetest is otherwise unrelated to real pipeline stages — see its own doc comment): after its three real API calls, it now saves the resulting book/chunk/questions/breakdown via this package, so results survive between runs instead of only ever living in stdout.

## 7g. Implementation Handoff — Breakdown Generation (Go)

Implemented in `pipeline/internal/breakdown`. Takes a `types.Chunk` (same as §7c) and produces its full Japanese breakdown: sentence structure, vocabulary, grammar, meaning/interpretation, and cultural/stylistic notes where genuinely relevant (§2 step 5). Same retry-then-error shape as §7c (`maxRetries`, no rule-based fallback — writing a good explanation isn't mechanical, same reasoning as question generation) and the same `llmCaller` consumer-defined-interface pattern as §7b/§7c/§7f.

**Deliberately simpler than §7c's request/response shape.** Question generation's output is a fixed set of typed fields (type/prompt/options/answer_index/explanation/highlight) validated field-by-field. A breakdown is one free-form text blob — matching `db/schema.sql`'s `breakdown.content` column and the Flutter app's already-established `Breakdown{id, content}` model (`app/lib/models/breakdown.dart`) — so there's no JSON envelope on the wire at all: the model is asked to output the Japanese text directly, and `GenerateBreakdown` returns it as a plain `string`.

**System prompt** matches the house style already validated in the Flutter mock content (`app/assets/mock/pride_and_prejudice.json`'s `breakdown.content` fields): Japanese section headers wrapped in 【】 (【文構造】 sentence structure, 【語彙】 vocabulary, 【文法】 grammar, 【意味】 meaning), blank-line-separated sections, exact English spans quoted from the chunk's text, `・` bullets for vocabulary lists. Includes one illustrative example (explicitly labeled as format-only) drawn from that same mock content. The prompt explicitly allows a chunk to skip sections that don't apply — a short, grammatically simple chunk doesn't need to manufacture a grammar note.

**Validation is deliberately loose**, unlike §7c's field-by-field checks: non-empty, and contains at least one Japanese character (a cheap sanity check against an accidentally-English response — the design's hard requirement per §2 step 5). It does *not* check for the specific 【...】 section structure the prompt asks for, since which sections a chunk actually needs varies — a rigid structural check would reject legitimate, thinner breakdowns for simple chunks along with genuinely malformed ones. Anything that check misses is a QA-pass (stage 5) concern, not this package's.

**Wired into `cmd/livetest`** alongside chunk grouping and question generation (§7f) — not yet run against the real Claude API (see the status table in §3).

## 7h. Implementation Handoff — Full Pipeline Orchestration (Go)

Implemented in `pipeline/cmd/process` — the "run it for real" counterpart to `cmd/ingest` (deliberately stops before Stage 4, §7d) and `cmd/livetest` (a throwaway smoke test on hand-typed sentences, not a real book, §7f). For each catalog entry: `pipeline.PrepareBook` → `segment.Segment` → `chunk.SplitIntoWindows` → Stage B per window → question + breakdown generation per resulting chunk → `internal/db`. Every book is saved with `Status` left at its default (`"processing"`) — this command never sets `"published"`; that transition is stage 5 (QA pass) and stage 6 (Publish), both still manual/not built, deliberately kept separate from bulk generation.

**`chunk.SplitIntoWindows`** (`internal/chunk`, not `cmd/process` — reusable, tested logic belongs in `internal/`, matching this project's existing split between thin `cmd/` orchestration and tested packages) turns a whole book's sentence list into ~3,000-char windows, one per Stage B call — see §3a for what's implemented (plain, non-overlapping cuts) versus not yet (overlap + edge reconciliation, forced chapter breaks). Chunk indices from each window's grouping response restart at 0 (`chunk.ValidatePartition`'s own rule) — `groupAllChunks` renumbers them sequentially across windows, so the whole book ends up with one globally-ordered chunk sequence.

**Persistence is incremental, not batched.** `groupAllChunks` saves each window's chunks (via `store.SaveChunk`, no questions yet) immediately after that window's Stage B call succeeds — not after every window in the book has been grouped. The per-chunk question/breakdown loop then re-saves the same row (same upsert-on-`(book_id, index)` semantics as everywhere else in `internal/db`) once its questions and breakdown are generated. A real API call is paid for the moment it returns, so a book-length run — potentially hundreds of sequential calls, well over an hour for a single short story — never holds already-paid-for output only in memory: a crash, an interruption, or a deliberately-stopped run loses nothing already completed. Sequential-with-immediate-persistence is the intended shape here, not a stopgap awaiting a concurrency optimization — for this pipeline, retaining completed work at every step outranks processing speed.

**Per-chunk failure is non-fatal.** A chunk whose question or breakdown generation fails after retrying is logged to stderr and skipped — the rest of the book is still saved. Safe specifically because status stays `"processing"`: nothing partially generated can reach a real user without going through the still-manual QA/publish steps first (§7a's "nothing reaches a paying user unreviewed").

**`-dry-run`** runs every free stage — fetch through windowing — for real (no Claude API calls, no Postgres writes) and prints the real window/chunk counts plus a rough, character-count-based cost estimate (`printCostEstimate`, clearly labeled as a ballpark, not a token-level measurement). Run against the real catalog as part of this work: The Vampyre (46,782 chars, 211 sentences) splits into 17 windows and an estimated ~195 chunks — ~407 real API calls, roughly $2.47 at `claude-sonnet-5` intro pricing. Meant to be checked before committing to a real run, the same way `cmd/livetest`'s cost was estimated before its first real run (see the "only the user launches anything that invokes the Claude API" rule this project works under).

## 8. Suggested Next Step

Scope v0: the smallest version that proves the full loop works end to end — one short public domain book, fully piped through (chunked, graded, questioned, broken down, QA'd), servable through a bare-bones Flutter client. Prove the pipeline and the pedagogy before building out level tagging, gamification, or monetization.

