# Crank handle

The model in `models/crankhandle` constructs a printed hand crank for a D-flat shaft: an arm that pinches the shaft, a
pin, a handle that turns on the pin, the M2 x 10 bolt and nut that clamp the pin in the arm, the two M2 x 6 cross bolts
and nuts that clamp the pin's two printed halves together, and the two M2 x 16 clamp bolts and nuts that close the arm's slit
on the shaft. `models/handcranksiren` builds its crank from it.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the throw, `DoubleFlat` (a second bore flat
opposite the first, on for `Params6082RS()`) and an optional `Placement` frame. The arm, pin and handle keep fixed
sizes, exported as constants (the arm's hub is `HubLength` 17 long, joined to the 8 mm arm by a `RampAngle` 30 ramp)
with the handle's meridian `HandleProfile`, the cross bolts' positions `CrossZ` and the clamp bolts' positions `ClampZ`. `Params605ZZ()` and `Params6082RS()` read the shaft diameter from
`ballbearing.Params605ZZ()` and `ballbearing.Params6082RS()`, and the bore clearance from
`ballbearing.PressFitClearance`. `DefaultParams()` returns `Params605ZZ()`. `Params6082RS()` is the siren's crank.
`Params.Derive()` returns the first failure in the order `instructions.md` lists, or `Dimensions`: the bore's radius,
flat, `BoreBack` and outline length, `RampFoot`, the pin bolt's `m2fastener.Site`, the two cross bolts' sites in
`CrossSites`, and the clamp's layout: `ClampY`, `SideEnd`, `Reach` and the two clamp bolts' sites `ClampSites`, with
`CornerCenter()` and `EndCenter()` for its arcs.

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.01 mm. The pin's cuts after its union
refuse a plain `decad.New()` document with a chain-depth error. `BuildParts(ctx, doc, params)` adds the arm, the pin
and the handle and returns them with every sketch, the pin bolt's and the cross bolts' sites in the world frame, and
`SplitNormal`, the world normal of the plane that splits the pin into its printed halves, and `ClampSites`. It files
them in no container, so an assembly files them with its own parts. `Build(ctx, doc, params)` adds those three, the
five bolts and the five nuts (13 bodies), in the `Arm`, `Pin`, `Handle`, `Bolts` and `Nuts` containers. A rejected build adds no
body.

The arm's outline runs from an arc round the pin down two straight sides, past the shaft, into two 4.5 mm corners and
one 22 mm end arc, every join tangent. It is extruded to the hub's 17 mm and both faces' outlines are chamfered; the chamfer needs decad
902e347 or later, which admits the outline's solver-placed tangent arcs in the Booleans that follow. A relief cut
leaves the 8 mm arm beyond a 30° ramp, and the new outer face's outline and the ramp's top edge are chamfered (decad
`aa32b3694405` or later). Then the arm is cut by the slit, the blind bore (with two flats for `DoubleFlat`), the
hexagonal pocket for the pin's stub, the pin bolt's head pocket, and each clamp bolt's hexagonal nut pocket in the -X
side face, head pocket and hole. Last, the bore's rim in the back face, which the slit opens into two arcs and the
flat or flats, is chamfered by 0.5 mm. The nut pocket keeps sharp hexagon corners and
its floor is rounded on the tool. The pin joins a hexagonal stub to a revolved shank and head, then
takes the double flats, the pin bolt's hole and nut cavity with rounded ends (`m2fastener.RoundedHoleTool` and
`RoundedCavityTool`), and each cross bolt's hole, head pocket and hexagonal nut pocket open to the -Y flat, with the
floor rounded by `NutPocketRound` as for the clamp. The handle is a revolved meridian.

## Build and images

```sh
go test ./models/crankhandle
go -C tools/render run . -model crankhandle -shot readme -out ../../models/crankhandle/images
```

The readme shot writes `hero.png`, `index.png`, `side.png`, `back.png`, `cutaway.png`, `exploded.png`, `split.png`,
`print.png`, `sizes.png`, `clamp.png`, `clamp-nut.png`, and `parts/arm.png`, `pin.png`, `pin-half-heads.png`,
`pin-half-nuts.png`, `handle.png`, `bolt.png`, `cross-bolt.png`, `clamp-bolt.png` and `nut.png`. `clamp` cuts the
arm's hub along the clamp bolt, and `clamp-nut` shows the hub from the nut's side; they are written only by the readme
shot. The presets `crankhandle/hero`, `side`, `back`, `cutaway`, `exploded`,
`split`, `print` and `sizes` render one view each. `split` and `print` clip the real pin mesh by the seam with
`solidlens.ClipMesh`; the model keeps the pin whole.

## Verification

Tests build four cases: both presets, a 0.35 mm bore clearance, and an 80 mm throw. For each they check:

- every sketch's full constraint, the thirteen bodies and their containers;
- `cornercheck` on the arm and the handle (axis +Z, with the hexagonal pocket's corners, clamp pocket 1's floor round
  and the clamp head pockets' floor rims listed) and on the pin with the split normal as its print
  axis (the cross pockets' corners listed);
- the bore: every point lies `BoreClearance` outside the shaft's D (or double-flat) section, it stops at its floor apart
  from the slit's round end, and the wall round it is at least 1.6 mm, or 1.05 mm over a clamp bolt's hole with a single
  D bore;
- the slit: one section loop runs from the bore down the slit's walls, 1 mm apart, to the end arc, and the slit ends
  in the floor 1 mm past the bore's wall;
- each clamp bolt: an M2 x 16 with its head flush with the +X side face and inside its pocket, crossing the slit, its
  tip 2 mm inside the -X face, its nut inside its hexagonal pocket below that face, and no overlap with the arm in
  sections along and across it; the pocket open to the -X face with a corner up and one down, its flats closer to the axis than the nut's corners
  and its corners clear of them; and every wall round the clamp's cuts at least 1 mm, also for a 16.6 mm shaft;
- the face chamfers: 0.25 mm inside each face the section stands 0.25 mm in from the pin end, the end arc and both
  sides, and a section 0.1 mm under the boss top shows the ramp top's chamfer;
- the grip (`TestCrankHandleGrip`): two clamp sites at `ClampZ`, the bore's depth, both bore flats with `DoubleFlat`, and
  the capacity estimate's engagement (at least 15.25 mm) and torque;
- the bore's rim chamfer: sections 0.05, 0.25 and 0.45 mm above the back face run along the arc of radius
  `r_b + 0.5 - z` and the flat (both flats with `DoubleFlat`) at `f_b + 0.5 - z`, and the bevel meets each slit wall where that arc does;
- the fits: the stub in its pocket (Tight), the handle round the shank (Regular), and the handle's gaps to the pin and
  the arm in a cut through the pin axis (Regular);
- the pin bolt: its head flush with the arm's back face, its tip inside the pin, the nut inside its cavity;
- each cross bolt: its head flush with the +Y flat and inside its pocket, its tip at the -Y flat, its nut inside its
  hexagonal pocket below that flat, the pocket open to the flat with its corners across the pin and its flats closer to the bolt than the nut's
  corners, the bolt crossing the seam, no overlap with the pin in cuts parallel to the seam, and 1 mm or
  more of pin round every pocket and cavity, except the listed 0.73 mm wall round each nut pocket's corners at the
  -Y flat (its value is pinned);
- `splitcheck` on the pin with the split normal and both cross bolts' lines exempt: no violation, bridges no wider
  than 6 mm, and each half as high as the head's radius.

The default's bounds and `VerifyAll` meshes, the placed crank and the validation errors are checked too. The siren's
tests check its crank through the assembly.
