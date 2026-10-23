import json
import tempfile
import unittest
from pathlib import Path

from plot_billing import aggregate


class BillingSummaryTests(unittest.TestCase):
    def test_rejects_missing_parent_or_duplicate_receipts(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            manifest = {"iterations": 10, "levels": [1, 2, 3], "rounds": 3,
                        "repeats": 2, "expectedOutputHash": "0xabc",
                        "profile": {"threshold": 2, "algorithms": [{"bands": [{"gas": 3000}]}]}}
            (directory / "manifest.json").write_text(json.dumps(manifest))
            samples = [{"iterations": 10, "callsPerTx": count, "path": path, "round": round_number,
                        "repeat": repeat, "gasUsed": 23000 + count, "status": 1,
                        "parentCalls": 0, "separatorTxHash": "0x1", "blockHash": "0x2",
                        "outputHash": "0xabc", "quotedCallGas": [3000] * count}
                       for count in manifest["levels"] for path in ("solidityCalls", "evocryptCalls")
                       for round_number in range(1, 4) for repeat in (1, 2)]

            def save():
                (directory / "samples.jsonl").write_text("\n".join(map(json.dumps, samples)) + "\n")

            save()
            _, rows = aggregate(directory)
            self.assertEqual(len(rows), 6)
            samples[0]["parentCalls"] = 1
            save()
            with self.assertRaisesRegex(ValueError, "parent/output"):
                aggregate(directory)
            samples[0]["parentCalls"] = 0
            samples[0]["repeat"] = 2
            save()
            with self.assertRaisesRegex(ValueError, "duplicate"):
                aggregate(directory)

    def test_prethreshold_quote_uses_frozen_base_gas(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            manifest = {"iterations": 10, "levels": [1], "rounds": 1, "repeats": 1,
                        "expectedOutputHash": "0xabc",
                        "profile": {"threshold": 4, "algorithms": [{"bands": [{"gas": 6000}]}]}}
            (directory / "manifest.json").write_text(json.dumps(manifest))
            samples = [{"iterations": 10, "callsPerTx": 1, "path": path, "round": 1,
                        "repeat": 1, "gasUsed": 28000, "status": 1, "parentCalls": 0,
                        "separatorTxHash": "0x1", "blockHash": "0x2", "outputHash": "0xabc",
                        "quotedCallGas": [6000]}
                       for path in ("solidityCalls", "evocryptCalls")]
            (directory / "samples.jsonl").write_text("\n".join(map(json.dumps, samples)) + "\n")
            _, rows = aggregate(directory)
            self.assertEqual(len(rows), 2)
            samples[1]["quotedCallGas"] = [3000]
            (directory / "samples.jsonl").write_text("\n".join(map(json.dumps, samples)) + "\n")
            with self.assertRaisesRegex(ValueError, "pre-threshold"):
                aggregate(directory)


if __name__ == "__main__":
    unittest.main()
