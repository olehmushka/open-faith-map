// This go.mod exists only to wall web/ off as its own Go module boundary, so `go build ./...` /
// `go test ./...` from the repo root never descends into web/node_modules — the go tool skips
// directories named "." / "_"-prefixed / "testdata" but NOT "node_modules", and at least one npm
// dependency here (flatted) happens to bundle a real .go file that would otherwise get picked up
// as a stray package. No real Go code lives in this module.
module github.com/olehmushka/open-faith-map/web

go 1.26.3
