# Seven-segment button detent-cam boundary steps

Read `button-detent-instructions.md`, `button-index-instructions.md`,
`button-index-steps.md`, `models/ballbearing/instructions.md` and the gallery
model pipeline. Do not inspect existing model Go source as an emission input.
Create new package-local production and focused test files without changing
existing manual or index builders. The emitted detent boundary is private.

## Pitch curve

The roller centre has radius R(θ) and radial derivative D=dR/dθ, in mm and
mm/radian. Use θ in [0,360), t=π/180 and slope s=2.8 mm/radian. Apply the
first matching interval below. Modulo is nonnegative.

- For θ≥354 or θ<20, let u=((θ+6) mod 360)t and L=4t. If u<L, set
  R=22+su²/(2L), D=su/L; otherwise set R=22+s(u−L/2), D=s.
- For 20≤θ<260, begin h=s(θ+4)t and D=s. For each (l,r,A) in
  (67,129,0.4), (205,259,s·14t−0.4), subtract A from h when θ≥r.
  When l<θ<r, set u=(θ−l)t, L=(r−l)t, K=2A/L; subtract
  K[u/2−L sin(2πu/L)/(4π)] from h and K sin²(πu/L) from D.
  Finally set R=22+h.
- For 260≤θ<280, let u=(θ−260)t, L=20t; set
  R=22+s(250t+u−u²/(2L)), D=s(1−u/L).
- For 280≤θ<354, let u=(θ−280)π/74, H=s·260t; set
  R=22+H(1+cos u)/2 and D=−H(90/74)sin u.

The low point is at 354°, and the highest nominal lift is 2.8·260π/180
mm. Check continuity and one-sided derivatives at every interval boundary.

## Cam geometry and key

1. At 360 integer-degree stations θ=0..359, calculate R,D, c=cos(θt),
   v=sin(θt), and L=hypot(R,D), where t=π/180. The inward normal offset for an R3 roller is
   P=(Rc−3(Rc+Dv)/L, −Rv−3(−Rv+Dc)/L).
2. Sketch the 360-point closed polygon in a local UV plane with an origin-centred
   construction reference. For each vertex create a construction ray from the
   origin; constrain its signed angle in degrees to atan2(Pv,Pu)/t and its
   length to hypot(Pu,Pv). The default Metric sketch takes angle dimensions in degrees.
   Join successive vertices by straight lines. Verify the fully constrained
   sketch, select its single closed profile and extrude local W=−1..+1.
   This explicitly specified one-degree sampled profile is the same construction
   used by the verified detent prototype; do not substitute another curve.
3. Build an origin-centred cylindrical hub R13, local W=−2.5..+2.5, and unite
   it to the cam disc. Build a separate constrained cutting sketch for the
   padded shaft/key through-opening, extruded local W=−3..+3. The opening is the
   same R2.55, ±10.10 keyed loop and 1.01/1.06 half-width lands and ramps
   specified in `button-index-steps.md`, Input shaft and crank step 2.
4. Let a=atan2(24,10), A=a+108°, and
   B=atan2(20 sin A−24,20 cos A−10). Rotate the opening tool by B in the
   local UV plane before cutting the united cam and hub. The cam lift profile
   stays clocked to +U. The undirected key axis is B modulo 180°.
5. Place the zero-phase cam at world (10,−107,52), local U=world +X,
   V=world +Z and W=world −Y. Its disc spans Y−108..−106 and its hub spans
   Y−109.5..−104.5. Make a separate Ø2×20 steel key along U at shaft station
   s=37.25 in the input shaft's B frame, centred on the shaft. Its world axis
   is (cos B,0,sin B), its centre is (10,−107,52), and its ends are ±10 mm.
   The real shaft hole is Ø2 nominal. The hub's closed key ends are ±10.10 mm.
6. Build the complete cam and key at phase zero. For a requested phase φ,
   rotate both finished bodies by φ around the world −Y axis through
   (10,−69.75,52), matching the existing index-shaft phase. Apply this rigid
   transform once. Do not perform Boolean operations in a rotated frame.

## Actual producer and consumer checks

Build the actual `buildButtonIndex` shaft at phase zero and the actual detent
cam/key in one default Decad document. Verify the cam and shaft separate,
verify the key occupies the real shaft's Ø2 hole without treating the nominal
line-to-line surfaces as positive running clearance, and verify the cam/key fit.
Force the real key ±0.30 mm along its axis in separate copies and require each
closed hub end to produce a positive overlap with the real key. The physical
press or retaining fit and load capacity of the pin remain bench checks.

At φ=0,1,10,20,109,180,251,260,271,280,301,320,331,345,354,356,358,359,
rotate the actual zero-phase cam by φ. At each phase build the purchased
`ballbearing.Build` body with bore 3, outside 6 and width 2.5 mm. Its axis is
world −Y, with its lower-cap centre at (10+R(φ)+offset,−105.75,52).
For offset +0.10 mm, require clearance from the real cam; for offset −0.10 mm,
verify contact or overlap with a bounded geometric witness. This bracket checks
both sides of the intended contact; it is not a roller-force or life test.
Verify the actual disc and hub remain one solid, and all sketches and containers
meet repository requirements. Use `VerifyAll` meshes and their actual bounds.
Keep failure diagnostics bounded.

Run this boundary's focused tests and the existing index focused checks. Document
that the incomplete follower and the complete button still require their real
consumer integration and physical tests. Do not update production gallery images
for this private boundary while the public manual display is unchanged.
