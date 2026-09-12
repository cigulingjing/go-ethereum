// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";
import {PolynomialMulContract} from "../src/PolynomialMul.sol";

contract PolynomialMulTest is Test {
    PolynomialMulContract internal polynomialMul;

    function setUp() public {
        polynomialMul = new PolynomialMulContract();
    }

    function test_PolynomialMul_basic() public view {
        uint256[] memory left = new uint256[](2);
        left[0] = 1;
        left[1] = 2;

        uint256[] memory right = new uint256[](2);
        right[0] = 3;
        right[1] = 4;

        uint256[] memory result = polynomialMul.PolynomialMul(left, right, 97);

        assertEq(result.length, 3);
        assertEq(result[0], 3);
        assertEq(result[1], 10);
        assertEq(result[2], 8);
    }

    function test_PolynomialMul_revertsOnEmptyInput() public {
        uint256[] memory left = new uint256[](0);
        uint256[] memory right = new uint256[](1);
        right[0] = 1;

        vm.expectRevert("empty left");
        polynomialMul.PolynomialMul(left, right, 97);
    }
}
