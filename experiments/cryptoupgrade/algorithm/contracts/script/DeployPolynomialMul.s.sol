// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import {Script, console2} from "forge-std/Script.sol";
import {PolynomialMulContract} from "../src/PolynomialMul.sol";

contract DeployPolynomialMul is Script {
    function run() external {
        vm.startBroadcast();
        PolynomialMulContract deployed = new PolynomialMulContract();
        vm.stopBroadcast();
        console2.log("PolynomialMulContract deployed at:", address(deployed));
    }
}
