import csv
import tempfile
import unittest
from pathlib import Path

from run_pbkdf2 import LEVELS, SCHEMES, aggregate


class SummaryTests(unittest.TestCase):
    def test_missing_round_remains_incomplete(self):
        with tempfile.TemporaryDirectory() as temporary:
            observations = []
            for round_number in range(1, 4):
                if round_number == 2:
                    continue
                observations.append({"iterations": 1, "round": round_number, "path": "upgrade",
                                     "measurementKind": "eth_call_rpc", "durationNs": 1_000_000})
                observations.append({"iterations": 1, "round": round_number, "path": "upgrade",
                                     "measurementKind": "receipt_gas", "gasUsed": 48000})
            aggregate(Path(temporary), observations, 3, 1, 1)
            with (Path(temporary) / "summary.csv").open(encoding="utf-8") as handle:
                rows = list(csv.DictReader(handle))
            self.assertEqual(len(rows), len(LEVELS) * len(SCHEMES))
            self.assertEqual(rows[0]["path"], "upgrade")
            self.assertEqual(rows[0]["complete"], "False")
            self.assertEqual(rows[0]["medianRpcMs"], "")


if __name__ == "__main__":
    unittest.main()
