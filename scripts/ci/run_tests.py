#!/usr/bin/env python3
"""Run the fast pull request or complete scheduled geometry test set."""

import json
import os
import re
import subprocess
import sys


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
]

BEVEL_FULL_TESTS = [
    test
    for shard in BEVEL_TESTS["full"].values()
    for test in shard
]


def run_pattern(tests: list[str]) -> str:
    """Build a go test -run pattern that selects exactly the named tests and subtests.

    Go splits a -run pattern on "/" and "|" only outside parentheses, so each test is one top-level
    alternative and each "/"-separated level is anchored on its own.
    """
    return "|".join(
        "/".join("^" + re.escape(part) + "$" for part in test.split("/"))
        for test in tests
    )


def finished_tests(lines) -> dict[str, str]:
    """Read go test -json lines, echo their output, and return each test's final action."""
    finished: dict[str, str] = {}
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
            finished[test] = event["Action"]
    sys.stdout.flush()
    return finished


def go_test(args: list[str], expected: list[str] = (), env: dict[str, str] | None = None) -> None:
    """Run go test with args and fail when it fails, runs no test, or skips a test in expected."""
    command = ["go", "test", "-json", *args]
    print("+", " ".join(command), flush=True)
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, text=True, env=env)
    finished = finished_tests(proc.stdout)
    if proc.wait() != 0:
        raise SystemExit(proc.returncode)
    if not any(action != "skip" for action in finished.values()):
        raise SystemExit("go test ran no tests; check the -run pattern")
    missing = [name for name in expected if finished.get(name) not in {"pass", "fail"}]
    if missing:
        raise SystemExit("go test did not run: " + ", ".join(missing))


def run_bevel(mode: str, shard: str) -> None:
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
# "native" runs the native and connector default builds. "rest" runs every other TestSpinningWheel test.
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


def run_wheel(case: str) -> None:
    command = ["./models/spinningwheel", "-timeout", "40m", "-count=1"]
    if case == "rest":
        command += ["-run", "^TestSpinningWheel", "-skip", WHEEL_HEAVY]
    else:
        command += ["-run", WHEEL_CASES[case]]
    go_test(command)


# The hand-crank siren builds 73 solids and verifies every pair; each case gets its own job.
SIREN_CASES = {
    "default": "^(TestHandCrankSirenDefault|TestValidate|TestMotion|TestPitch)$",
    "variants": "^TestHandCrankSirenVariant$",
}

FAN_VARIANTS = {
    "flat-three": "^TestHandCrankFanVariant$/^flat-three$",
    "steep-three": "^TestHandCrankFanVariant$/^steep-three$",
}


def run_siren(case: str) -> None:
    go_test(["./models/handcranksiren", "-run", SIREN_CASES[case], "-timeout", "40m", "-count=1"])


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
                "./models/planetarygearbox",
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


def main() -> None:
    _, suite, mode, *args = sys.argv
    if suite == "models" and mode in {"pr", "full"} and not args:
        run_models(mode)
        return
    if suite == "bevel" and mode in BEVEL_TESTS and len(args) == 1:
        run_bevel(mode, args[0])
        return
    if suite == "wheel" and not args and (mode in WHEEL_CASES or mode == "rest"):
        run_wheel(mode)
        return
    if suite == "siren" and not args and mode in SIREN_CASES:
        run_siren(mode)
        return
    if suite == "fan" and not args and mode in FAN_VARIANTS:
        run_fan_variant(mode)
        return
    raise SystemExit(
        "usage: run_tests.py models <pr|full> | bevel <pr|full> <a-h> | wheel <case|rest>"
        " | siren <default|variants> | fan <flat-three|steep-three>"
    )


if __name__ == "__main__":
    main()
