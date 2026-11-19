#!/usr/bin/env python3
"""Run the fast pull request or complete scheduled geometry test set."""

import json
import os
import re
import subprocess
import sys
import tempfile


BEVEL_TESTS = {
    "pr": {
        "a": [
            "TestRealEightToothFullPatternJoin",
            "TestBlankPairSharesOneSketch",
            "TestSecondPairGetsOwnContainers",
            "TestShaftSectionIsExactOnItsAxis",
            "TestLocalShaftKeepsItsSection",
        ],
        "b": ["TestSourceFourFourStraightPairAutoBore"],
        "c": ["TestStraightOptionsBuildMembers"],
        "d": ["TestSpiralInputsMembers/module_0_5"],
        "e": ["TestSpiralSerialPair"],
        "f": ["TestSpiralSerialRouteBuildsMembers/options/tooth_spacing"],
        "g": ["TestSpiralCutterRadiusBuildsMember"],
        "h": ["TestPlacedMemberMatchesWorldMember", "TestPlacedPairMovesBothMembers"],
    },
    "full": {
        "a": [
            "TestRealEightToothFullPatternJoin",
            "TestJoinedEightToothPairThroughBore",
            "TestSourceFourFourStraightMember",
            "TestSourceFourFourStraightMemberAutoBore",
            "TestSourceFourFourStraightPairAutoBore",
            "TestBlankPairSharesOneSketch",
            "TestSecondPairGetsOwnContainers",
            "TestShaftSectionIsExactOnItsAxis",
            "TestLocalShaftKeepsItsSection",
        ],
        "b": [
            "TestSourceSixteenTwelveStraightPair",
            "TestSourceSixteenTwelveStraightPairAutoBore",
            "TestSourceThirtyTwoSixteenStraightPair",
            "TestStraightPairShaftAngle",
        ],
        "c": [
            "TestStraightOptionsBuildMembers",
            "TestStraightPairMateOptions",
            "TestStraightShaftAnglesBuildMembers",
        ],
        "d": ["TestSpiralInputsMembers"],
        "e": ["TestSpiralInputsPair", "TestSpiralSerialPair"],
        "f": [
            "TestSpiralSerialRouteBuildsMembers",
            "TestSpiralSerialRouteBuildsUnverifiedMembers",
        ],
        "g": [
            "TestSpiralHandWidthAndBaseHeight",
            "TestSpiralCutterRadiusBuildsMember",
            "TestSpiralPairCutterRadius",
        ],
        "h": [
            "TestSpiralPairHandWidthAndBaseHeight",
            "TestSourceDefaultSpiralPair",
            "TestPlacedMemberMatchesWorldMember",
            "TestPlacedPairMovesBothMembers",
        ],
    },
}

MODEL_PR_TESTS = [
    "TestBearing608Envelope",
    "TestBearing605Envelope",
    "TestBearingPlacedAndRepeated",
    "TestBearingCustomDimensions",
    "TestBearingRejectsInvalidInputs",
    "TestDefaultParamsAreValid",
    "TestPatternedLoopClosesOnSharedPoints",
    "TestProblems",
    "TestUndercutBound",
    "TestParameterTableMatchesResolve",
    "TestBuildDefaultDrive",
    "TestHelixAngleRange",
    "TestZeroHelixRejectedBeforeAddingBodies",
    "TestSmallRealHelicalGear",
    "TestSmallRealHelicalGearThroughBore",
    "TestSmallHelicalToothAndRootDiscVerifyIndividually",
    "TestDefaultRealHelicalGear",
    "TestDefaultRealHelicalGearThroughBore",
    "TestHerringboneMirrorPlaneIsHalfThickness",
    "TestOutOfRangeHelixRejectedBeforeAddingBodies",
    "TestSmallRealHerringboneGear",
    "TestSmallRealHerringboneGearWithBoreAndCapChamfer",
    "TestBuildRejectsInvalidParamsBeforeAddingBodies",
    "TestProfileAndFlankAPIsRejectInvalidInputs",
    "TestGearProfilesAreFullyConstrained",
    "TestBoreProfileIsFullyConstrained",
    "TestSmallGeneratedGear",
    "TestPlacedGearMatchesWorldGear",
    "TestPlacedEmbeddedGearWithRootFilletsAndBore",
    "TestPlacedHelicalGearMatchesWorldGear",
    "TestPlacedHerringboneGearMatchesWorldGear",
    "TestInvalidPlacementIsRejected",
    "TestBuildPlacementMovesTheWholeMechanism",
    "TestBuildRejectsInvalidParams",
    "TestFramePairAtTheDefaults",
    "TestFrameAngleAndSectionCoordinates",
    "TestFramePlacementCarriesThePair",
    "TestFrameBoreRoofsAtTheDefaults",
    "TestParamsDefaultsAreTheSpecTable",
    "TestParamsDerivedValues",
    "TestParamsValidateRangeChecks",
    "TestParamsValidateOrder",
    "TestParamsValidateSleeveBounds",
    "TestSleeveChannelSeparationAtTheDefaults",
    "TestSleeveChannelSeparationShrinksWithTheWall",
    "TestSleeveWindowFacingsFollowTheCrossingAngle",
    "TestWindowSearchAtTheDefaults",
    "TestWindowSearchKeepsCollarWallToTheFlankingBores",
    "TestWindowSearchRoomRefusesTheBandWithNoWidth",
    "TestSectionPair/default/cuts-both-ribbons",
    "TestSectionPair/default/plane-outside-gear-a",
    "TestSectionPair/placed",
    "TestSectionPair/default/clearance",
    "TestRibbonDefaultIsOneVerifiedLoft",
    "TestExplodedGearbox",
    "TestBearingCatalog",
    "TestDefaultDimensionsUnchanged",
    "TestParameterTableMatchesDerive",
    "TestDerivePerRole",
    "TestSupportedCombinationsAreValid",
    "TestProblemsNameForcedMinimums",
    "TestNewFitChecks",
    "TestCorners",
    "TestSplitShafts",
    "TestSpiralSpringDefault",
    "TestSpiralSpringBranches",
    "TestSpiralSpringFits",
    "TestSpiralSpringPlacement",
    "TestSpiralSpringRejectsInvalidInputs",
    "TestSpiralSpringClearance",
    "TestSpiralSpringParamsDefaults",
    "TestSpiralSpringParamsThicknessFloor",
    "TestSpiralSpringParamsTurnBoundary",
    "TestSpiralSpringParamsRoot",
    "TestSpiralSpringParamsRootRoom",
    "TestSpiralSpringCorners",
    "TestSpiralSpringRootThickness",
    "TestSpiralSpringParamsEyeFloor",
    "TestCrankHandleCorners",
    "TestCrankHandleContainers",
    "TestCrankHandleDefault",
    "TestCrankHandleBoreFit",
    "TestCrankHandleFasteners",
    "TestCrankHandleEdgeFinishing",
    "TestCrankHandleFits",
    "TestCrankHandlePlacement",
    "TestCrankHandleRejectsInvalidInputs",
    "TestPlaqueStandParamsDefaults",
    "TestPlaqueStandDeriveCases",
    "TestPlaqueStandDeriveRejects",
    "TestPlaqueStandOutlineCases",
    "TestPlaqueStandBuildDefault",
    "TestPlaqueStandBuildCases",
    "TestPlaqueStandBuildRejectsInvalidInputs",
    "TestPlaqueStandClearance",
    "TestPlaqueStandStability",
    "TestPlaqueStandCorners",
    "TestPlaqueStandRailFit",
    "TestPlaqueStandFasteners",
]

BEVEL_FULL_TESTS = [
    test
    for shard in BEVEL_TESTS["full"].values()
    for test in shard
]
BEVEL_JOB_GROUPS = {
    "abgh": ("a", "b", "g", "h"),
    "cdef": ("c", "d", "e", "f"),
}

ESCAPEMENT_SHARDS = (
    (
        "TestArborCollars",
        "TestCentreLine",
        "TestCorners",
        "TestEdgeFinishing",
        "TestFits",
        "TestInstallability",
        "TestInventory",
        "TestPlacement",
        "TestPressFitWalls",
        "TestSetJoinsEveryPart",
        "TestSketches",
        "TestSplitShafts",
        "TestSpringFits",
    ),
    (
        "TestActionAnalytic",
        "TestActionAnalyticWide",
        "TestAnchor",
        "TestArbors",
        "TestBuildValidates",
        "TestCoupling",
        "TestEscapeWheel",
        "TestEscapementAction",
        "TestFasteners",
        "TestFrontPlateSpringAnchor",
        "TestGearMesh",
        "TestKnob",
        "TestLayout",
        "TestMainspring",
        "TestMotion",
        "TestPawlCarrier",
        "TestPendulumParts",
        "TestPlateMaterial",
        "TestPlateWindows",
        "TestPlates",
        "TestPoseNoOverlap",
        "TestRatchet",
        "TestSketchesBranches",
        "TestSpringLayout",
        "TestSpringRoot",
        "TestValidate",
    ),
)


def run_pattern(tests: list[str]) -> str:
    """Build a go test -run pattern that selects exactly the named tests and subtests.

    Go splits a -run pattern on "/" and "|" only outside parentheses, so each test is one top-level
    alternative and each "/"-separated level is anchored on its own.
    """
    return "|".join(
        "/".join("^" + re.escape(part) + "$" for part in test.split("/"))
        for test in tests
    )


def finished_tests(lines, qualified: bool = False) -> dict[str | tuple[str, str], str]:
    """Read go test -json lines, echo their output, and return each test's final action."""
    finished: dict[str | tuple[str, str], str] = {}
    for line in lines:
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            sys.stdout.write(line)
            continue
        if "Output" in event:
            sys.stdout.write(event["Output"])
        test = event.get("Test")
        if test and event.get("Action") in {"pass", "fail", "skip"}:
            key = (event.get("Package", ""), test) if qualified else test
            finished[key] = event["Action"]
    sys.stdout.flush()
    return finished


def go_test(args: list[str], expected: list[str] = (), env: dict[str, str] | None = None,
            package: str | None = None, allowed_skips: frozenset[str] = frozenset()) -> None:
    """Run go test with args and fail when it fails, runs no test, or skips a test in expected."""
    command = ["go", "test", "-json", *args]
    print("+", " ".join(command), flush=True)
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True, env=env)
    finished = finished_tests(proc.stdout, qualified=package is not None)
    if proc.wait() != 0:
        raise SystemExit(proc.returncode)
    if not any(action != "skip" for action in finished.values()):
        raise SystemExit("go test ran no tests; check the -run pattern")
    missing = [
        f"{package}/{name}" if package else name
        for name in expected
        if finished.get((package, name) if package else name) != "pass"
        and not (name in allowed_skips and finished.get((package, name) if package else name) == "skip")
    ]
    if package:
        missing.extend(f"{pkg}/{name}" for (pkg, name), status in finished.items()
                       if status != "pass" and not (pkg == package and status == "skip" and name in allowed_skips))
    missing = sorted(set(missing))
    if missing:
        suffix = f" (and {len(missing) - 20} more)" if len(missing) > 20 else ""
        raise SystemExit("go test did not pass: " + ", ".join(missing[:20]) + suffix)


def go_test_shards(package: str, shards: tuple[tuple[str, ...], ...], timeout: str) -> None:
    """Run exact, disjoint test shards concurrently and require complete package coverage."""
    flattened = [test for shard in shards for test in shard]
    if len(flattened) != len(set(flattened)):
        raise SystemExit("go test shards contain duplicate test names")

    listing = subprocess.run(
        ["go", "test", "-list", "^Test", package],
        check=True,
        capture_output=True,
        text=True,
    )
    available = {line for line in listing.stdout.splitlines() if line.startswith("Test")}
    missing = sorted(available - set(flattened))
    unknown = sorted(set(flattened) - available)
    if missing or unknown:
        details = []
        if missing:
            details.append("unassigned tests: " + ", ".join(missing))
        if unknown:
            details.append("unknown tests: " + ", ".join(unknown))
        raise SystemExit("go test shard coverage does not match package: " + "; ".join(details))

    env = os.environ.copy()
    env["GOMAXPROCS"] = "2"
    processes = []
    try:
        for index, shard in enumerate(shards, start=1):
            command = [
                "go",
                "test",
                "-json",
                package,
                "-run",
                run_pattern(list(shard)),
                "-timeout",
                timeout,
                "-count=1",
            ]
            print(f"+ shard {index}/{len(shards)}: {' '.join(command)}", flush=True)
            output = tempfile.TemporaryFile(mode="w+t")
            try:
                proc = subprocess.Popen(command, stdout=output, stderr=subprocess.STDOUT, text=True, env=env)
            except BaseException:
                output.close()
                raise
            processes.append((proc, output, shard))

        results = []
        for proc, output, shard in processes:
            returncode = proc.wait()
            output.seek(0)
            finished = finished_tests(output)
            failures = [test for test in shard if finished.get(test) != "pass"]
            results.append((returncode, failures))
        failed = [
            f"shard {index} (exit {returncode}): {', '.join(failures) or 'process failed'}"
            for index, (returncode, failures) in enumerate(results, start=1)
            if returncode != 0 or failures
        ]
        if failed:
            raise SystemExit("go test shard failure: " + "; ".join(failed))
    finally:
        for proc, output, _ in processes:
            if proc.poll() is None:
                proc.terminate()
                proc.wait()
            output.close()


def run_bevel(mode: str, shard: str) -> None:
    if mode == "pr" and shard == "rest":
        package = "./models/bevelgear"
        listing = subprocess.run(["go", "test", "-list", "^Test", package],
                                 check=True, capture_output=True, text=True)
        assigned = {test for tests in BEVEL_TESTS["pr"].values() for test in tests if "/" not in test}
        tests = [line for line in listing.stdout.splitlines() if line.startswith("Test") and line not in assigned]
        if tests:
            go_test([package, "-run", run_pattern(tests), "-timeout", "60m"], tests,
                    package="github.com/lestrrat-3d/decad-gallery/models/bevelgear")
        return
    if shard in BEVEL_JOB_GROUPS:
        for member in BEVEL_JOB_GROUPS[shard]:
            run_bevel(mode, member)
        return
    tests = BEVEL_TESTS[mode][shard]
    timeout = "15m" if mode == "pr" else "60m"
    go_test(["./models/bevelgear", "-run", run_pattern(tests), "-timeout", timeout], tests)


# The spinning wheel builds roughly 190 solids and verifies every pair, so each case gets its own job and
# its own time limit. "rest" holds every other spinning wheel test.
#
# Which job runs which check. checkSpinningWheel (models/spinningwheel/build_test.go) runs two subtests
# for the cases that check the linkage, each with its own Build:
#   <case>-verify   everything except requireLinkageFrameClearances: Build, part checks, Sketch
#                   verification, requireNoMaterialOverlap, brake captivity, wheel features, head
#                   adjustment, flyer sweep and drive band checks.
#   <case>-linkage  Build, then requireLinkageFrameClearances only.
# The default, small-seven and large-nine cases are split this way, so the two jobs of one case together
# run exactly the checks the unsplit test ran. The twelve and small-seventeen cases skip the linkage check
# on purpose (it repeats a scale already checked), so one job runs all of their verify checks.
# "native" runs the native and connector default builds. "rest" runs every other package test.
WHEEL_CASES = {
    "default-verify": "^TestSpinningWheelDefault$/^verify$",
    "default-linkage": "^TestSpinningWheelDefault$/^linkage$",
    "native": "^(TestSpinningWheelNativeDefault|TestSpinningWheelConnectorDefault)$",
    "small-seven-verify": "^TestSpinningWheelVariants$/^small-seven$/^verify$",
    "small-seven-linkage": "^TestSpinningWheelVariants$/^small-seven$/^linkage$",
    "large-nine-verify": "^TestSpinningWheelVariants$/^large-nine$/^verify$",
    "large-nine-linkage": "^TestSpinningWheelVariants$/^large-nine$/^linkage$",
    "twelve": "^TestSpinningWheelVariants$/^twelve$",
    "small-seventeen": "^TestSpinningWheelVariants$/^small-seventeen$",
}
WHEEL_HEAVY = "^(TestSpinningWheelDefault|TestSpinningWheelNativeDefault|TestSpinningWheelConnectorDefault|TestSpinningWheelVariants)$"
WHEEL_EXPECTED = {
    "default-verify": ["TestSpinningWheelDefault/verify"],
    "default-linkage": ["TestSpinningWheelDefault/linkage"],
    "native": ["TestSpinningWheelNativeDefault", "TestSpinningWheelConnectorDefault"],
    "small-seven-verify": ["TestSpinningWheelVariants/small-seven/verify"],
    "small-seven-linkage": ["TestSpinningWheelVariants/small-seven/linkage"],
    "large-nine-verify": ["TestSpinningWheelVariants/large-nine/verify"],
    "large-nine-linkage": ["TestSpinningWheelVariants/large-nine/linkage"],
    "twelve": ["TestSpinningWheelVariants/twelve"],
    "small-seventeen": ["TestSpinningWheelVariants/small-seventeen"],
}


def run_wheel(case: str) -> None:
    command = ["./models/spinningwheel", "-timeout", "40m", "-count=1"]
    if case == "rest":
        command += ["-run", "^Test", "-skip", WHEEL_HEAVY]
    else:
        command += ["-run", WHEEL_CASES[case]]
    go_test(command, WHEEL_EXPECTED.get(case, ()),
            package="github.com/lestrrat-3d/decad-gallery/models/spinningwheel")


# The hand-crank siren builds 73 solids and verifies every pair; each case gets its own job.
SIREN_CASES = {
    "default": (
        "^(TestHandCrankSirenDefault|TestValidate|TestMotion|TestPitch|TestGearboxParams)$",
        ["TestHandCrankSirenDefault", "TestValidate", "TestMotion", "TestPitch", "TestGearboxParams"],
    ),
    "spike": (
        "^TestGearboxSpike$",
        ["TestGearboxSpike"],
    ),
    "slots12": (
        "^TestHandCrankSirenVariant$/^slots_12$",
        ["TestHandCrankSirenVariant/slots_12"],
    ),
    "slots24": (
        "^TestHandCrankSirenVariant$/^slots_24$",
        ["TestHandCrankSirenVariant/slots_24"],
    ),
}
SIREN_ASSIGNED = {test.split("/")[0] for _, tests in SIREN_CASES.values() for test in tests}

FAN_VARIANTS = {
    "flat-three": "^TestHandCrankFanVariant$/^flat-three$",
    "steep-three": "^TestHandCrankFanVariant$/^steep-three$",
}


def run_siren(case: str) -> None:
    if case == "rest":
        package = "./models/handcranksiren"
        listing = subprocess.run(
            ["go", "test", "-list", "^Test", package], check=True, capture_output=True, text=True
        )
        tests = [line for line in listing.stdout.splitlines()
                 if line.startswith("Test") and line not in SIREN_ASSIGNED]
        if not tests:
            raise SystemExit("siren remainder lists no tests")
        go_test([package, "-run", run_pattern(tests), "-timeout", "40m", "-count=1"], tests,
                package="github.com/lestrrat-3d/decad-gallery/models/handcranksiren")
        return
    pattern, expected = SIREN_CASES[case]
    go_test(
        ["./models/handcranksiren", "-run", pattern, "-timeout", "40m", "-count=1"],
        expected,
    )


def run_escapement() -> None:
    """Run all Graham escapement tests in two concurrent, fully covering shards."""
    go_test_shards("./models/escapement", ESCAPEMENT_SHARDS, "40m")


def run_turbinefan() -> None:
    """Run every test of the turbine fan package: about 5 minutes on a busy runner, so it has its own job."""
    go_test(["./models/turbinefan", "-timeout", "40m", "-count=1"])


def run_onewayclutch() -> None:
    """Run every test of the one-way clutch package: about 12 minutes, over the 5 minute PR job limit, so it has its
    own job."""
    go_test(["./models/onewayclutch", "-timeout", "40m", "-count=1"])


def run_fan_variant(variant: str) -> None:
    env = os.environ.copy()
    env["HANDCRANKFAN_FULL"] = "1"
    go_test(
        [
            "./models/handcrankfan",
            "-run",
            FAN_VARIANTS[variant],
            "-timeout",
            "40m",
            "-count=1",
        ],
        [f"TestHandCrankFanVariant/{variant}"],
        env,
    )


def run_models(mode: str) -> None:
    if mode == "pr":
        go_test(
            [
                "./models/cycloidal",
                "./models/helicalgear",
                "./models/herringbonegear",
                "./models/screwgear",
                "./models/spurgear",
                "./models/ballbearing",
                "./models/pinring",
                "./models/planetarygearbox",
                "./models/spiralspring",
                "./models/crankhandle",
                "./models/plaquestand",
                "-run",
                run_pattern(MODEL_PR_TESTS),
                "-timeout",
                "5m",
            ],
            MODEL_PR_TESTS,
        )
        return

    pattern = "^(" + "|".join(BEVEL_FULL_TESTS) + ")$"
    subprocess.run(["go", "test", "-skip", pattern, "-timeout", "60m", "./..."], check=True)


def run_model_pr(name: str) -> None:
    """Run every listed test in one model package and check its qualified results."""
    if not re.fullmatch(r"[a-z][a-z0-9]*", name):
        raise SystemExit("invalid model package name")
    package = f"./models/{name}"
    listing = subprocess.run(
        ["go", "test", "-list", "^Test", package], check=True, capture_output=True, text=True
    )
    tests = [line for line in listing.stdout.splitlines() if line.startswith("Test")]
    if not tests:
        raise SystemExit(f"model package {name} lists no tests")
    qualified_package = f"github.com/lestrrat-3d/decad-gallery/models/{name}"
    allowed_skips = set()
    if name == "handcrankfan" and not os.environ.get("HANDCRANKFAN_FULL"):
        allowed_skips.update(("TestHandCrankFanDefault", "TestHandCrankFanVariant"))
    if name == "planetarygearbox" and not os.environ.get("PLANETARY_ALL_KITS"):
        allowed_skips.update(("TestKitGearboxes/planet608-shaft605", "TestKitGearboxes/planet608-shaft608"))
    go_test([package, "-timeout", "60m", "-count=1"], tests,
            package=qualified_package, allowed_skips=frozenset(allowed_skips))
    for test in tests:
        if test not in allowed_skips:
            print(f"PASS {qualified_package} {test}")


def run_cycloidal_pr() -> None:
    run_model_pr("cycloidal")


def main() -> None:
    _, suite, mode, *args = sys.argv
    if suite == "cycloidal" and mode == "pr" and not args:
        run_cycloidal_pr()
        return
    if suite == "model" and mode == "pr" and len(args) == 1:
        run_model_pr(args[0])
        return
    if suite == "models" and mode in {"pr", "full"} and not args:
        run_models(mode)
        return
    if suite == "bevel" and mode in BEVEL_TESTS and len(args) == 1:
        run_bevel(mode, args[0])
        return
    if suite == "wheel" and not args and (mode in WHEEL_CASES or mode == "rest"):
        run_wheel(mode)
        return
    if suite == "siren" and not args and (mode in SIREN_CASES or mode == "rest"):
        run_siren(mode)
        return
    if suite == "escapement" and not args and mode == "all":
        run_escapement()
        return
    if suite == "onewayclutch" and not args and mode == "all":
        run_onewayclutch()
        return
    if suite == "turbinefan" and not args and mode == "all":
        run_turbinefan()
        return
    if suite == "fan" and not args and mode in FAN_VARIANTS:
        run_fan_variant(mode)
        return
    raise SystemExit(
        "usage: run_tests.py cycloidal pr | model pr <name> | models <pr|full> | bevel <pr|full> <a-h> | wheel <case|rest>"
        " | siren <default|spike|slots12|slots24|rest> | escapement all | onewayclutch all | turbinefan all | fan <flat-three|steep-three>"
    )


if __name__ == "__main__":
    main()
