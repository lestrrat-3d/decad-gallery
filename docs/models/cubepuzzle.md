# Cube Puzzle model

`models/cubepuzzle` builds a fixed 60 mm cube puzzle. Use `cubepuzzle.NewDocument()` and call
`cubepuzzle.Build(ctx, doc, cubepuzzle.DefaultParams())`. The returned assembly contains the hub and slices of blocks
and mount components. The document groups bodies by part type.

The focused checks build and tessellate the actual bodies, verify the assembled clearances and fastener positions,
read the printed-part sections, and sample face turns and insertion positions. Two extraction witnesses establish
specific blocked motions. The samples do not establish every continuous motion or physical print performance.

Run the model checks with:

```sh
go test ./models/cubepuzzle
```

Regenerate its images with:

```sh
go -C tools/render run . -all -model cubepuzzle
```
