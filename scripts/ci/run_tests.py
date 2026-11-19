#!/usr/bin/env python3
"""Run the fast pull request or complete scheduled geometry test set."""

import json
import os
import re
import subprocess
import sys
import tempfile
import uuid


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

MODEL_PR_PACKAGES = {
    "cycloidal": (
        "TestDefaultParamsAreValid",
        "TestPatternedLoopClosesOnSharedPoints",
        "TestProblems",
        "TestUndercutBound",
        "TestParameterTableMatchesResolve",
        "TestBuildDefaultDrive",
        "TestCorners",
    ),
    "helicalgear": (
        "TestHelixAngleRange",
        "TestZeroHelixRejectedBeforeAddingBodies",
        "TestSmallRealHelicalGear",
        "TestSmallRealHelicalGearThroughBore",
        "TestSmallHelicalToothAndRootDiscVerifyIndividually",
        "TestDefaultRealHelicalGear",
        "TestDefaultRealHelicalGearThroughBore",
        "TestPlacedHelicalGearMatchesWorldGear",
        "TestInvalidPlacementIsRejected",
        "TestCorners",
    ),
    "herringbonegear": (
        "TestHerringboneMirrorPlaneIsHalfThickness",
        "TestOutOfRangeHelixRejectedBeforeAddingBodies",
        "TestSmallRealHerringboneGear",
        "TestSmallRealHerringboneGearWithBoreAndCapChamfer",
        "TestPlacedHerringboneGearMatchesWorldGear",
        "TestInvalidPlacementIsRejected",
        "TestCorners",
    ),
    "screwgear": (
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
    ),
    "spurgear": (
        "TestBuildRejectsInvalidParamsBeforeAddingBodies",
        "TestProfileAndFlankAPIsRejectInvalidInputs",
        "TestGearProfilesAreFullyConstrained",
        "TestBoreProfileIsFullyConstrained",
        "TestSmallGeneratedGear",
        "TestPlacedGearMatchesWorldGear",
        "TestPlacedEmbeddedGearWithRootFilletsAndBore",
        "TestInvalidPlacementIsRejected",
        "TestCorners",
    ),
    "ballbearing": (
        "TestBearing608Envelope",
        "TestBearing605Envelope",
        "TestBearingPlacedAndRepeated",
        "TestBearingCustomDimensions",
        "TestBearingRejectsInvalidInputs",
        "TestBearingCatalog",
    ),
    "pinring": (
        "TestCorners",
    ),
    "planetarygearbox": (
        "TestExplodedGearbox",
        "TestDefaultDimensionsUnchanged",
        "TestParameterTableMatchesDerive",
        "TestDerivePerRole",
        "TestSupportedCombinationsAreValid",
        "TestProblemsNameForcedMinimums",
        "TestNewFitChecks",
        "TestCorners",
        "TestSplitShafts",
    ),
    "spiralspring": (
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
    ),
    "crankhandle": (
        "TestCrankHandleCorners",
        "TestCrankHandleContainers",
        "TestCrankHandleDefault",
        "TestCrankHandleBoreFit",
        "TestCrankHandleFasteners",
        "TestCrankHandleEdgeFinishing",
        "TestCrankHandleFits",
        "TestCrankHandlePlacement",
        "TestCrankHandleRejectsInvalidInputs",
    ),
    "plaquestand": (
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
    ),
}

BEVEL_JOB_GROUPS = {
    "abgh": ("a", "b", "g", "h"),
    "cdef": ("c", "d", "e", "f"),
}
BEVEL_REMAINDER_SHARDS = 4

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


def finished_tests(lines, qualified: bool = False, module: str = ".") -> dict[str | tuple[str, str], str]:
    """Read go test -json lines, echo their output, and return each test's final action."""
    finished: dict[str | tuple[str, str], str] = {}
    record_path = os.environ.get("CI_TEST_RECORDS")
    invocation = uuid.uuid4().hex if record_path else None
    records = open(record_path, "a", encoding="utf-8") if record_path else None
    try:
        return _read_finished_tests(lines, qualified, module, invocation, records, finished)
    finally:
        if records is not None:
            records.close()


def _read_finished_tests(lines, qualified, module, invocation, records, finished):
    metadata = {"version": 1, "invocation": invocation, "module": module,
                "job": os.environ.get("GITHUB_JOB", ""), "case": os.environ.get("CI_TEST_CASE", ""),
                "revision": os.environ.get("GITHUB_SHA", ""), "run_id": os.environ.get("GITHUB_RUN_ID", ""),
                "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT", "")}

    def record(kind, **fields):
        if records is not None:
            records.write(json.dumps({**metadata, "kind": kind, **fields}, separators=(",", ":")) + "\n")

    for line in lines:
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            record("malformed", reason="invalid_json")
            sys.stdout.write(line)
            continue
        if not isinstance(event, dict):
            record("malformed", reason="not_object")
            continue
        if "Output" in event:
            sys.stdout.write(event["Output"])
        test = event.get("Test")
        if event.get("Action") in {"pass", "fail", "skip"}:
            package = event.get("Package")
            if not isinstance(package, str) or not package or (test is not None and
                                                                (not isinstance(test, str) or not test)):
                record("malformed", reason="invalid_identity")
            else:
                record("test" if test else "package", package=package, test=test, status=event["Action"])
        if test and event.get("Action") in {"pass", "fail", "skip"}:
            key = (event.get("Package", ""), test) if qualified else test
            finished[key] = event["Action"]
    record("complete")
    sys.stdout.flush()
    return finished


def go_test(args: list[str], expected: list[str] = (), env: dict[str, str] | None = None,
            package: str | None = None, allowed_skips: frozenset[str] = frozenset(),
            qualified_expected: set[tuple[str, str]] | None = None) -> None:
    """Run go test with args and fail when it fails, runs no test, or skips a test in expected."""
    command = ["go", "test", "-json", *args]
    print("+", " ".join(command), flush=True)
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True, env=env)
    qualified = package is not None or qualified_expected is not None
    finished = finished_tests(proc.stdout, qualified=qualified)
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
    if qualified_expected is not None:
        missing.extend(f"{pkg}/{name}" for pkg, name in qualified_expected
                       if finished.get((pkg, name)) != "pass")
        missing.extend(f"{pkg}/{name}" for (pkg, name), status in finished.items() if status != "pass")
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
            failures.extend(test for test, status in finished.items()
                            if status != "pass" and any(test.startswith(parent + "/") for parent in shard))
            failures = sorted(set(failures))
            results.append((returncode, failures))
        failed = [
            f"shard {index} (exit {returncode}): {', '.join(failures[:20]) or 'process failed'}"
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


def bevel_remainder_shards(available: list[str]) -> list[list[str]]:
    """Assign every otherwise unselected top-level test to one remainder shard."""
    assigned = {test for tests in BEVEL_TESTS["pr"].values() for test in tests if "/" not in test}
    remainder = sorted(set(available) - assigned)
    return [remainder[index::BEVEL_REMAINDER_SHARDS] for index in range(BEVEL_REMAINDER_SHARDS)]


def run_bevel(mode: str, shard: str) -> None:
    if mode == "pr" and shard in {f"rest-{index}" for index in range(BEVEL_REMAINDER_SHARDS)}:
        package = "./models/bevelgear"
        listing = subprocess.run(["go", "test", "-list", "^Test", package],
                                 check=True, capture_output=True, text=True)
        available = [line for line in listing.stdout.splitlines() if line.startswith("Test")]
        tests = bevel_remainder_shards(available)[int(shard.removeprefix("rest-"))]
        if tests:
            partial = [test for group in BEVEL_TESTS["pr"].values() for test in group
                       if "/" in test and test.split("/")[0] in tests]
            command = [package, "-run", run_pattern(tests), "-timeout", "60m"]
            if len(partial) == 1:
                command += ["-skip", run_pattern(partial)]
            go_test(command, tests,
                    package="github.com/lestrrat-3d/decad-gallery/models/bevelgear")
        return
    if shard in BEVEL_JOB_GROUPS:
        for member in BEVEL_JOB_GROUPS[shard]:
            run_bevel(mode, member)
        return
    tests = BEVEL_TESTS[mode][shard]
    timeout = "15m" if mode == "pr" else "60m"
    if mode == "full":
        go_test(["./models/bevelgear", "-run", run_pattern(tests), "-timeout", timeout], tests,
                package="github.com/lestrrat-3d/decad-gallery/models/bevelgear")
    else:
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
        assigned = [test for tests in WHEEL_EXPECTED.values() for test in tests]
        run_remainder("./models/spinningwheel", assigned)
        return
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
FAN_VARIANTS = {
    "flat-three": "^TestHandCrankFanVariant$/^flat-three$",
    "steep-three": "^TestHandCrankFanVariant$/^steep-three$",
}


def run_siren(case: str) -> None:
    if case == "rest":
        run_remainder("./models/handcranksiren", [test for _, tests in SIREN_CASES.values() for test in tests])
        return
    pattern, expected = SIREN_CASES[case]
    go_test(
        ["./models/handcranksiren", "-run", pattern, "-timeout", "40m", "-count=1"],
        expected, package="github.com/lestrrat-3d/decad-gallery/models/handcranksiren",
    )


def run_remainder(package: str, assigned: list[str]) -> None:
    """Run everything except paths assigned to other jobs, including new nested subtests."""
    if len(assigned) != len(set(assigned)) or any(
            other.startswith(test + "/") for test in assigned for other in assigned):
        raise SystemExit("test assignments overlap")
    listing = subprocess.run(["go", "test", "-list", "^Test", package],
                             check=True, capture_output=True, text=True)
    available = {line for line in listing.stdout.splitlines() if line.startswith("Test")}
    unknown = {test.split("/")[0] for test in assigned} - available
    if unknown:
        raise SystemExit("assigned tests missing from inventory: " + ", ".join(sorted(unknown)))
    expected = sorted(available - {test for test in assigned if "/" not in test})
    qualified = "github.com/lestrrat-3d/decad-gallery/" + package.removeprefix("./")
    go_test([package, "-run", "^Test", "-skip", run_pattern(assigned), "-timeout", "40m", "-count=1"],
            expected, package=qualified)


def run_escapement() -> None:
    """Run all Graham escapement tests in two concurrent, fully covering shards."""
    go_test_shards("./models/escapement", ESCAPEMENT_SHARDS, "40m")


def run_model_packages(names: tuple[str, ...]) -> None:
    """Run every inventoried test in the named model packages."""
    expected = set()
    for name in names:
        package = f"./models/{name}"
        listing = subprocess.run(["go", "test", "-list", "^Test", package],
                                 check=True, capture_output=True, text=True)
        tests = [line for line in listing.stdout.splitlines() if line.startswith("Test")]
        if not tests:
            raise SystemExit(f"model package {name} lists no tests")
        qualified = f"github.com/lestrrat-3d/decad-gallery/models/{name}"
        expected.update((qualified, test) for test in tests)
    go_test([*(f"./models/{name}" for name in names), "-timeout", "40m", "-count=1"],
            qualified_expected=expected)


def run_turbinefan() -> None:
    """Run every test of the turbine fan package: about 5 minutes on a busy runner, so it has its own job."""
    run_model_packages(("turbinefan",))


def run_onewayclutch() -> None:
    """Run every test of the one-way clutch package: about 12 minutes, over the 5 minute PR job limit, so it has its
    own job."""
    run_model_packages(("onewayclutch",))


def run_clamparm() -> None:
    """Run every test of the clamp arm and M5 fastener packages: about 4 minutes on a fast machine, so they have their
    own job instead of the 5 minute PR job."""
    run_model_packages(("clamparm", "m5bolt", "m5nut"))


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
        packages = MODEL_PR_PACKAGES
        qualified_expected = {(f"github.com/lestrrat-3d/decad-gallery/models/{name}", test)
                              for name, tests in packages.items() for test in tests}
        go_test(
            [
                *(f"./models/{name}" for name in packages),
                "-run",
                run_pattern(MODEL_PR_TESTS),
                "-timeout",
                "5m",
            ],
            qualified_expected=qualified_expected,
        )
        return

    run_full_models()


def run_full_models() -> None:
    """Check the root module inventory and run every test outside the bevel shards."""
    listing = subprocess.run(["go", "list", "./..."], check=True, capture_output=True, text=True)
    packages = listing.stdout.splitlines()
    if not packages:
        raise SystemExit("root module lists no packages")
    bevel = "github.com/lestrrat-3d/decad-gallery/models/bevelgear"
    assigned = [test for tests in BEVEL_TESTS["full"].values() for test in tests]
    if len(assigned) != len(set(assigned)):
        raise SystemExit("full bevel shards contain duplicate test names")
    inventory = {}
    for package in packages:
        result = subprocess.run(["go", "test", "-list", "^Test", package],
                                check=True, capture_output=True, text=True)
        inventory[package] = {line for line in result.stdout.splitlines() if line.startswith("Test")}
    unknown = sorted(set(assigned) - inventory.get(bevel, set()))
    if unknown:
        raise SystemExit("unknown full bevel shard tests: " + ", ".join(unknown))
    expected = {(package, test) for package, tests in inventory.items() if package != bevel for test in tests}
    prefix = "github.com/lestrrat-3d/decad-gallery/models/"
    expected.update({
        (prefix + "handcrankfan", "TestHandCrankFanVariant/flat-three"),
        (prefix + "handcrankfan", "TestHandCrankFanVariant/steep-three"),
        (prefix + "planetarygearbox", "TestKitGearboxes/planet608-shaft605"),
        (prefix + "planetarygearbox", "TestKitGearboxes/planet608-shaft608"),
    })
    env = os.environ.copy()
    env.update(HANDCRANKFAN_FULL="1", PLANETARY_ALL_KITS="1")
    other_packages = [package for package in packages if package != bevel]
    go_test([*other_packages, "-timeout", "60m"], env=env, qualified_expected=expected)
    remainder = sorted(inventory.get(bevel, set()) - set(assigned))
    if remainder:
        go_test([bevel, "-run", run_pattern(remainder), "-timeout", "60m"], remainder,
                env=env, package=bevel)


def run_planetary_full(shard: str) -> None:
    package = "github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
    output = subprocess.run(["go", "test", "-list", "^Test", package],
                            check=True, capture_output=True, text=True).stdout
    tests = {line for line in output.splitlines() if line.startswith("Test")}
    kit = "TestKitGearboxes"
    if kit not in tests or len(tests) < 2:
        raise SystemExit("planetary test inventory is incomplete")
    selected = [kit] if shard == "kits" else sorted(tests - {kit}) if shard == "rest" else None
    if selected is None:
        raise SystemExit("invalid planetary shard")
    expected = set(selected)
    if shard == "kits":
        expected.update(f"{kit}/{name}" for name in (
            "planet605-shaft608", "planet608-shaft605", "planet608-shaft608"))
    env = os.environ.copy()
    env["PLANETARY_ALL_KITS"] = "1"
    go_test([package, "-run", run_pattern(selected), "-timeout", "60m", "-count=1"],
            sorted(expected), env=env, package=package)


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
    if suite == "planetary" and mode == "full" and len(args) == 1:
        run_planetary_full(args[0])
        return
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
    if suite == "clamparm" and not args and mode == "all":
        run_clamparm()
        return
    if suite == "turbinefan" and not args and mode == "all":
        run_turbinefan()
        return
    if suite == "fan" and not args and mode in FAN_VARIANTS:
        run_fan_variant(mode)
        return
    raise SystemExit(
        "usage: run_tests.py cycloidal pr | model pr <name> | models <pr|full> | bevel <pr|full> <a-h|rest-0..3>"
        " | wheel <case|rest> | siren <default|spike|slots12|slots24|rest> | escapement all"
        " | onewayclutch all | clamparm all | turbinefan all | fan <flat-three|steep-three>"
    )


if __name__ == "__main__":
    main()
