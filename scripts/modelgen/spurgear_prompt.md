# Spur gear Go emission

Generate the `decad-gallery/models/spurgear` Go implementation from this
bundle's `models/spurgear/instructions.md` and `models/spurgear/steps.md`.
Implement the construction, public functions, and parameter cases stated
there. Treat `instructions.md` as the geometry contract and `steps.md` as
the Decad operation recipe. Report any contradiction before emitting code.

Read only files under this prepared bundle for model-specific construction.
The bundled `internal/sketchcheck/verify.go`, `go.mod`, and `go.sum` describe
declared dependencies. You may inspect installed Decad, Sketch, Go, and
other dependency APIs to resolve their signatures. Do not read existing
`models/spurgear/*.go` from any checkout. Do not copy a prior implementation.

Write `involute.go`, `spurgear.go`, and `tooth_section.go` under the bundle's
`output/` directory. Use package `spurgear`. Keep all model geometry in
those files. Do not edit tracked files. Report the exact generated paths,
any missing instruction, and any dependency failure. The caller will stage
the candidate unchanged and run the repository's tests.
