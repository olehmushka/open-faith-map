// This go.mod exists only to wall web/ off as its own Go module boundary, so `go build ./...` /
// `go test ./...` from the repo root never descends into web/apps/*/node_modules — the go tool
// skips directories named "." / "_"-prefixed / "testdata" but NOT "node_modules", and at least one
// npm dependency (flatted) happens to bundle a real .go file that would otherwise get picked up as
// a stray package. Covers both web/apps/web and web/apps/admin (M2.1, D-AdminSurface) — one module
// root one level up from both is sufficient; no real Go code lives in this module.
module github.com/olehmushka/open-faith-map/web

go 1.26.3
