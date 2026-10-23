import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from run_pbkdf2_no_termination import validated_samples


class EvoCryptTimingTests(unittest.TestCase):
    def test_validates_reference_and_sample_count(self):
        with tempfile.TemporaryDirectory() as temporary:
            result_path = Path(temporary) / "result.json"
            reference = "0x" + hashlib.pbkdf2_hmac("sha256", b"password", b"salt", 10, 32).hex()
            implementation = {"scheme": "upgrade", "outputValue": reference,
                              "latency": {"sampleMillis": [1.0] * 30, "warmup": 10}}
            result_path.write_text(json.dumps({"results": [{"outputMatched": True,
                                                                "implementations": [implementation]}]}))
            self.assertEqual(len(validated_samples(result_path, 10, 30)), 30)
            with self.assertRaises(ValueError):
                validated_samples(result_path, 20, 30)
            with self.assertRaises(ValueError):
                validated_samples(result_path, 10, 31)


if __name__ == "__main__":
    unittest.main()
