import copy
import json
import unittest
from pathlib import Path

from compare import compare, validate_input


ROOT = Path(__file__).resolve().parents[2] / "results" / "gas-complexity"


def fixtures():
    calibration = (ROOT / "2026-10-22-calibration.json").read_bytes()
    model = json.loads((ROOT / "2026-10-22-solidity-range-model.json").read_text())
    baselines = [json.loads((ROOT / f"2026-10-22-solidity-{part}.json").read_text())
                 for part in ("first", "middle", "dh", "tail")]
    return calibration, model, baselines


class ComparisonTests(unittest.TestCase):
    def test_compare_range_and_receipts(self):
        result = compare(*fixtures())
        self.assertEqual(len(result["algorithms"]), 8)
        self.assertEqual(result["algorithms"][0]["solidityReceiptGas"], 21738)
        self.assertEqual(result["algorithms"][0]["grid"][0]["projectedStandaloneGas"], 22336)
        self.assertEqual(result["algorithms"][3]["cheaperGridPoints"], 25)
        self.assertIn("unavailable", result["algorithms"][5])
        self.assertFalse(result["algorithms"][4]["inputEncodingComparable"])

    def test_rejects_old_or_failed_receipt(self):
        calibration, model, baselines = fixtures()
        old = copy.deepcopy(baselines)
        old[0]["referenceInputs"] = False
        with self.assertRaisesRegex(ValueError, "-reference-inputs"):
            compare(calibration, model, old)
        old = copy.deepcopy(baselines)
        old[0]["results"][0]["contractExecution"]["status"] = 0
        with self.assertRaisesRegex(ValueError, "failed or unmatched"):
            compare(calibration, model, old)

    def test_rejects_mismatched_input_or_calibration(self):
        calibration, model, baselines = fixtures()
        with self.assertRaisesRegex(ValueError, "does not match"):
            validate_input("Sha256", {"dataHex": "0x73"})
        changed = copy.deepcopy(model)
        changed["calibrationSHA256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "digest mismatch"):
            compare(calibration, changed, baselines)


if __name__ == "__main__":
    unittest.main()
