# Spiral spring Go emission

Generate the `decad-gallery/models/spiralspring` Go implementation from this
bundle's `models/spiralspring/instructions.md` and `models/spiralspring/steps.md`.
Implement the construction, public functions, and parameter cases stated
there. Treat `instructions.md` as the geometry contract and `steps.md` as
the Decad operation recipe. Report any contradiction before emitting code.

Read only files under this prepared bundle for model-specific construction.
The bundled `internal/sketchcheck/verify.go`, `internal/edgefinish/edgefinish.go`,
`internal/placement/placement.go`, `go.mod`, and `go.sum` describe declared
dependencies. You may inspect installed Decad, Sketch, Go, and other dependency
APIs to resolve their signatures. Do not read existing `models/spiralspring/*.go`
from any checkout. Do not copy a prior implementation.

Write `params.go`, `profile.go`, and `spiralspring.go` under the bundle's
`output/` directory. Use package `spiralspring`. Keep all model geometry in
those files. You may compile them in a scratch module under the bundle that
reuses the bundled `go.mod`, `go.sum` and the bundled `internal/` packages.
Do not edit tracked files. Report the exact generated paths,
any missing instruction, and any dependency failure. The caller will stage
the candidate unchanged and run the repository's tests.
