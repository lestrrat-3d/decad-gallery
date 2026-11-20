# Bolts and nuts

The model in `models/fastener` builds a bolt or a hex nut from a named size preset (M2 or M5) held in one catalog.

## Construction and API

`Names()` lists the catalog and `Lookup(name)` returns a copy of a preset. A `Preset` holds a `BoltSize` and the
`NutSize` of the nut that pairs with it, and `Preset.Validate()` checks it.

`BoltParams` exposes `Preset`, `Length` and an optional `Placement` frame. `DefaultBoltParams()` returns the M2x10
bolt. `Length` must be one of the preset's `Bolt.StandardLengths`. The local frame puts the center of the head's
underside at the origin, the head along +Z and the shank along -Z.
`BuildBolt(ctx, doc, params)` returns a `Bolt` with `Body` and its sketches: `Profile` and `Socket` for a socket head
cap screw, `Head` and `Profile` for a hex head bolt. Every sketch is fully constrained, and the build checks that
before it uses it. The socket head cap screw is one revolve of the section, a fillet under the head and one
hexagonal cut. Decad's boolean keeps its drill-point cone as a faceted face after that cut. The hex head bolt is a
chamfered hexagonal head from `internal/hexnut` joined to a revolved shank.

`NutParams` exposes `Preset` and an optional `Placement` frame. `BuildNut(ctx, doc, params)` returns a `Nut` with
`Body` and the `Hexagon`, `LowerChamfer` and `UpperChamfer` sketches. The construction is `internal/hexnut`'s.

The thread is not modeled, so a shank is the nominal diameter along its whole length. To add a size, see "Adding a
preset" in `models/fastener/instructions.md`.

## Build and images

```sh
go test ./models/fastener
go -C tools/render run . -model fastener -shot readme -out ../../models/fastener/images
```

The readme shot writes `hero.png`, `index.png`, `m2-bolt-head.png`, `m2-bolt-side.png`, `m2-bolt-underside.png`,
`m2-bolt-lengths.png`, `m2-nut-hero.png`, `m2-nut-top.png`, `m2-nut-side.png`, `m2-nut-low.png`,
`m2-nut-withbolt.png`, `m5-bolt-hero.png`, `m5-bolt-head.png`, `m5-bolt-side.png`, `m5-bolt-lengths.png`,
`m5-nut-hero.png`, `m5-nut-top.png`, `m5-nut-side.png` and `m5-nut-withbolt.png`.
`-model fastener -shot <name>` renders one of them, such as `-shot m2-bolt-head`.

## Verification

Tests check every preset's default bolt and nut: bounds, volume against an integral, face kinds, mesh verification and
sketches. They also check every standard length, placement, invalid inputs and the catalog values and rules.
