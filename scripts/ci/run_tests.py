#!/usr/bin/env python3
"""Run the fast pull request or complete scheduled geometry test set."""

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
        "b": ["TestSourceFourFourStraightPairAutoBore", "TestStraightPairBuildsMembersInTheirContainers"],
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
            "TestStraightPairBuildsMembersInTheirContainers",
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
    "TestSectionPairCutsBothRibbons",
    "TestSectionPairOnePlaneOutsideGearAReadsEmpty",
    "TestSectionPairFollowsThePlacement",
    "TestSectionPairClearanceOfTheDefaultPair",
    "TestRibbonDefaultIsOneVerifiedLoft",
]

BEVEL_FULL_TESTS = [
    test
    for shard in BEVEL_TESTS["full"].values()
    for test in shard
]


def run_bevel(mode: str, shard: str) -> None:
    tests = BEVEL_TESTS[mode][shard]
    pattern = "^(" + "|".join(tests) + ")$"
    timeout = "15m" if mode == "pr" else "60m"
    subprocess.run(
        ["go", "test", "./models/bevelgear", "-run", pattern, "-timeout", timeout],
        check=True,
    )


# The spinning wheel builds roughly 190 solids and verifies every pair, so each case gets its own job and
# its own time limit. "rest" holds every other spinning wheel test.
WHEEL_CASES = {
    "default": "^TestSpinningWheelDefault$",
    "native": "^(TestSpinningWheelNativeDefault|TestSpinningWheelConnectorDefault)$",
    "small-seven": "^TestSpinningWheelVariants$/^small-seven$",
    "large-nine": "^TestSpinningWheelVariants$/^large-nine$",
    "twelve": "^TestSpinningWheelVariants$/^twelve$",
    "small-seventeen": "^TestSpinningWheelVariants$/^small-seventeen$",
}
WHEEL_HEAVY = "^(TestSpinningWheelDefault|TestSpinningWheelNativeDefault|TestSpinningWheelConnectorDefault|TestSpinningWheelVariants)$"


def run_wheel(case: str) -> None:
    command = ["go", "test", "./models/spinningwheel", "-timeout", "40m", "-count=1"]
    if case == "rest":
        command += ["-run", "^TestSpinningWheel", "-skip", WHEEL_HEAVY]
    else:
        command += ["-run", WHEEL_CASES[case]]
    subprocess.run(command, check=True)


def run_models(mode: str) -> None:
    if mode == "pr":
        pattern = "^(" + "|".join(MODEL_PR_TESTS) + ")$"
        subprocess.run(
            [
                "go",
                "test",
                "./models/cycloidal",
                "./models/helicalgear",
                "./models/herringbonegear",
                "./models/screwgear",
                "./models/spurgear",
                "./models/ballbearing",
                "-run",
                pattern,
                "-timeout",
                "5m",
            ],
            check=True,
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
    raise SystemExit("usage: run_tests.py models <pr|full> | bevel <pr|full> <a-h> | wheel <case|rest>")


if __name__ == "__main__":
    main()
