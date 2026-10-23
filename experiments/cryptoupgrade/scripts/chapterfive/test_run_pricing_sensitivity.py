import copy
import json
import unittest
from fractions import Fraction
from math import ceil

from run_pricing_sensitivity import DEFAULT_ROOT, check_matrix


class SensitivityMatrixTests(unittest.TestCase):
    def test_changes_one_factor_per_case(self):
        matrix = json.loads((DEFAULT_ROOT / "matrix.json").read_text())
        check_matrix(matrix)
        modified = copy.deepcopy(matrix)
        modified["cases"][0]["threshold"] = 4
        with self.assertRaisesRegex(ValueError, "multiple factors"):
            check_matrix(modified)

    def test_frozen_grid_fits_transaction_limit_without_retuning(self):
        matrix = json.loads((DEFAULT_ROOT / "matrix.json").read_text())
        maximum_quote_sum = 0
        for case in matrix["cases"]:
            alpha = Fraction(case["alphaNumerator"], case["alphaDenominator"])
            quotes = [ceil(case["baseGas"] * (1 + alpha) ** max(0, index - case["threshold"]))
                      for index in range(1, 17)]
            quote_sum = sum(quotes)
            maximum_quote_sum = max(maximum_quote_sum, quote_sum)
            self.assertLess(quote_sum, matrix["controls"]["receiptGasLimit"], case["name"])
        self.assertEqual(maximum_quote_sum, 17685053)


if __name__ == "__main__":
    unittest.main()
