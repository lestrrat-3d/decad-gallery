# Planetary gearbox with 8 mm shafts (608ZZ cover bearings)

[Planetary gearbox](README.md)

![The two-stage planetary gearbox of the planet605-shaft608 kit, fully assembled, with the ring gear and both end covers cut open to show the planets, carriers, shafts and bearings inside](images/kit-planet605-shaft608/hero.png)

This kit keeps the 605ZZ planet bearings of the default gearbox and holds the input and output shafts in 608ZZ cover
bearings. Each shaft is 8 mm across where it passes a bearing or a sun, the bore of a 608ZZ, so it has no clearance in
the bearing. The carrier pins stay 5 mm across, the bore of a 605ZZ. A wider printed shaft carries more torque.

The kit has a 14-tooth sun, three 28-tooth planets and a 70-tooth ring. The default 13/29/71 teeth do not fit here: the
8.1 mm sun bore needs a sun of at least 14 teeth at module 1. One stage slows the output to 1 ÷ 6 of the input speed
(1 + 70 ÷ 14), and two stages slow it to 1 ÷ 36. The images show two stages in the same ring. The ring and both covers
are cut open along a plane through the axis in the hero image; every other part is shown whole.

## Assembled

The complete two-stage gearbox with nothing cut away, from the output end, from the input end and from the side.

| From the output end | From the input end | From the side |
| --- | --- | --- |
| <img src="images/kit-planet605-shaft608/assembled-output.png" alt="Assembled two-stage gearbox seen from the output end, with an 8 mm output shaft in a 608ZZ bearing boss" width="320"> | <img src="images/kit-planet605-shaft608/assembled-input.png" alt="Assembled two-stage gearbox seen from the input end, with an 8 mm input shaft in a 608ZZ bearing boss" width="320"> | <img src="images/kit-planet605-shaft608/assembled-side.png" alt="Assembled two-stage gearbox seen from the side, with the two 8 mm shaft ends and the wide cover bearing bosses" width="320"> |

## Exploded

<img src="images/kit-planet605-shaft608/exploded.png" alt="The two-stage gearbox pulled apart along its axis: the front bearing below the front cover, the input shaft above it, then the ring, two stages of planets, suns with the planet bearings, carriers and carrier pins with a link shaft between them, then the output shaft, the rear cover and the rear bearing">

The picture pulls the gearbox apart along its axis. It leaves out the bolts and nuts, which are the same as in the
default gearbox.

The parts, from the bottom of the picture:

1. The front bearing, a 608ZZ, off the input shaft's outer end.
2. The front cover, with its wide bearing boss.
3. The input shaft, out of the front cover's inner side.
4. The ring.
5. The first stage: the planets; the sun, beside the 605ZZ planet bearings, which come up out of the planets; the
   carrier; and above it the carrier pins, beside the link shaft.
6. The second stage: the planets; the sun, beside the planet bearings; the carrier pins, below the carrier; the
   carrier; and the output shaft, out of the top of the carrier.
7. The rear cover, then the rear bearing, a 608ZZ, off the output shaft's outer end.

## What changes from the default

| Dimension | Default | This kit |
| --- | --- | --- |
| Sun, planet and ring teeth | 13, 29, 71 | 14, 28, 70 |
| One-stage ratio | 6.46 | 6 |
| Shaft diameter | 5 mm | 8 mm |
| Sun bore | 5.1 mm | 8.1 mm |
| Shaft collar and thick part | 7.4 mm | 10.4 mm |
| Cover bearing pocket | 14.1 mm | 22.1 mm |
| Cover lip bore | 10 mm | 18 mm |
| Cover bearing boss | 20.1 mm | 28.1 mm |
| Cover hub | 26.1 mm | 34.1 mm |
| Ring outer radius | 44.1 mm | 43.6 mm |

The carrier pins, the planet pockets and lips, the pin boss and the cover spoke width are the same as the default's.

## Parts

The gearbox has the same parts as the default. The two cover bearings are 608ZZ and the three planet bearings are
605ZZ, so this kit lists two bearings. Each is drawn as its nominal envelope.

| Image | Name |
| --- | --- |
| <img src="images/kit-planet605-shaft608/parts/ring.png" alt="Ring gear" width="240"> | Ring gear |
| <img src="images/kit-planet605-shaft608/parts/cover.png" alt="Cover with a 28.1 mm bearing boss" width="240"> | Cover (front and rear) |
| <img src="images/kit-planet605-shaft608/parts/sun.png" alt="Sun gear with 14 teeth" width="240"> | Sun gear |
| <img src="images/kit-planet605-shaft608/parts/planet.png" alt="Planet gear with 28 teeth" width="240"> | Planet gear |
| <img src="images/kit-planet605-shaft608/parts/planet-bearing.png" alt="605ZZ planet bearing" width="240"> | Planet bearing (605ZZ) |
| <img src="images/kit-planet605-shaft608/parts/cover-bearing.png" alt="608ZZ cover bearing" width="240"> | Cover bearing (608ZZ) |
| <img src="images/kit-planet605-shaft608/parts/input-shaft.png" alt="Input shaft, 8 mm across, with its collar" width="240"> | Input shaft |
| <img src="images/kit-planet605-shaft608/parts/carrier.png" alt="Carrier" width="240"> | Carrier |
| <img src="images/kit-planet605-shaft608/parts/carrier-pin.png" alt="Carrier pin" width="240"> | Carrier pin |
| <img src="images/kit-planet605-shaft608/parts/link-shaft.png" alt="Link shaft with its 10.4 mm thick part" width="240"> | Link shaft |
| <img src="images/kit-planet605-shaft608/parts/output-shaft.png" alt="Output shaft with its 10.4 mm thick part" width="240"> | Output shaft |

## Building it

```go
p, err := planetarygearbox.DefaultParamsFor("planet605-shaft608")
if err != nil {
	return err
}
p.Stages = 2
box, err := planetarygearbox.Build(ctx, planetarygearbox.NewDocument(), p)
```

`DefaultParamsFor` returns the default parameters with `PlanetBearing` 605ZZ, `ShaftBearing` 608ZZ and the 14/28/70
teeth. For the functions and checks, see the [developer notes](../../docs/models/planetarygearbox.md).
