#!/usr/bin/env python3
"""Run the fast pull request or complete scheduled geometry test set."""

import subprocess
import sys


BEVEL_TESTS = {
    "pr": {
        "a": ["TestRealEightToothFullPatternJoin"],
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
    "TestDefaultParamsAreValid",
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
                "./models/spurgear",
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
    raise SystemExit("usage: run_tests.py models <pr|full> | bevel <pr|full> <a-h>")


if __name__ == "__main__":
    main()
