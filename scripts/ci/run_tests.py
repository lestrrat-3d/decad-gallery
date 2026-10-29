#!/usr/bin/env python3
"""Run the fast pull request or complete scheduled geometry test set."""

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
]

DECAD_MODULE = "github.com/lestrrat-3d/decad"
GO_MOD_FILES = ["go.mod", "tools/render/go.mod"]
# The long twisted loft only the full suite otherwise runs. A decad bump must pass it before merge.
DECAD_BUMP_TEST = "TestRibbonDefaultIsOneVerifiedLoft"

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


def decad_requirement(go_mod: str) -> str | None:
    """Return the version go.mod requires for decad, or None when it has no such requirement."""
    match = re.search(r"^\s*(?:require\s+)?" + re.escape(DECAD_MODULE) + r"\s+(\S+)", go_mod, re.MULTILINE)
    return match.group(1) if match else None


def git_show(base: str, path: str) -> str:
    result = subprocess.run(["git", "show", f"{base}:{path}"], capture_output=True, text=True)
    return result.stdout if result.returncode == 0 else ""


def decad_bumped(base: str) -> bool:
    """Report whether the decad requirement in any go.mod differs from the one at base."""
    for path in GO_MOD_FILES:
        try:
            with open(path, encoding="utf-8") as f:
                current = decad_requirement(f.read())
        except FileNotFoundError:
            current = None
        if current != decad_requirement(git_show(base, path)):
            return True
    return False


def run_decad_bump(base: str) -> None:
    if not decad_bumped(base):
        print(f"decad requirement unchanged against {base}; skipping {DECAD_BUMP_TEST}")
        return
    print(f"decad requirement changed against {base}; running {DECAD_BUMP_TEST}")
    subprocess.run(
        ["go", "test", "./models/screwgear", "-run", f"^{DECAD_BUMP_TEST}$", "-timeout", "10m"],
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
                "./models/screwgear",
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
    if suite == "models" and mode == "decad-bump" and len(args) <= 1:
        run_decad_bump(args[0] if args else "HEAD^1")
        return
    if suite == "bevel" and mode in BEVEL_TESTS and len(args) == 1:
        run_bevel(mode, args[0])
        return
    raise SystemExit("usage: run_tests.py models <pr|full|decad-bump [base]> | bevel <pr|full> <a-h>")


if __name__ == "__main__":
    main()
