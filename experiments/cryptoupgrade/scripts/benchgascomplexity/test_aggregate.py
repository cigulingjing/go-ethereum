import copy
import unittest

from aggregate import REFERENCE_LEVELS, aggregate


def sample_run():
    measurements = []
    for name, reference in REFERENCE_LEVELS.items():
        for level in (reference // 2, reference, reference * 2, reference * 4):
            measurements.append({
                "algorithm": name,
                "metric": "work",
                "value": level,
                "inputBytes": 128,
                "outputSha256": f"{name}-{level}",
                "medianNs": 101 if name == "Add" and level == reference else 301,
            })
    return {
        "anchorGas": 100,
        "anchorAlgorithm": "Add",
        "anchorMedianNs": 101,
        "wasmSha256": {name: name for name in REFERENCE_LEVELS},
        "measurements": measurements,
    }


class AggregateTests(unittest.TestCase):
    def test_ratio_rounded_up(self):
        result = aggregate([sample_run(), sample_run()])
        self.assertEqual(result["algorithms"]["Add"]["referenceBaseGas"], 100)
        self.assertEqual(result["algorithms"]["Sha256"]["referenceBaseGas"], 299)

    def test_reject_mismatched_wasm(self):
        run = sample_run()
        different = copy.deepcopy(run)
        different["wasmSha256"]["Add"] = "other"
        with self.assertRaises(ValueError):
            aggregate([run, different])

    def test_reject_mismatched_input(self):
        run = sample_run()
        different = copy.deepcopy(run)
        different["measurements"][0]["inputBytes"] += 1
        with self.assertRaises(ValueError):
            aggregate([run, different])


if __name__ == "__main__":
    unittest.main()
