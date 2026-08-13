# book-content: the content-serving REST API. Built from the repo root
# (see docker-compose.yml's build.context: .) rather than book-content/
# alone, because this module resolves aidoku/shared via go.work (see
# /go.work's own doc comment), not a go.mod require/replace — Go
# workspace mode needs every use'd module's go.mod present to compute
# the workspace's build list, even for modules the build doesn't
# actually import.
#
# pipeline/'s go.mod is copied for exactly that reason (workspace
# resolution) but not its source — this binary doesn't import anything
# from pipeline, and confirming that's true (a plain go.mod is enough)
# is what keeps this build context from having to include the whole
# pipeline module.
FROM golang:1.26-alpine AS build
WORKDIR /workspace

COPY go.work go.work.sum ./
COPY pipeline/go.mod pipeline/go.mod
COPY book-content/go.mod book-content/go.mod
COPY shared/go.mod shared/go.mod
RUN go mod download

COPY book-content/ book-content/
COPY shared/ shared/

RUN CGO_ENABLED=0 go build -o /aidoku-server ./book-content/cmd/server

# pgx v5 is pure Go (no libpq/CGO), so the build above is a fully
# static binary — a plain alpine base (matching postgres:16-alpine's
# convention in this compose file) is enough, no libc concerns.
FROM alpine:3.20
COPY --from=build /aidoku-server /usr/local/bin/aidoku-server
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/aidoku-server"]
