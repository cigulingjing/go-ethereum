// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

interface ICodeStorageConsistency {
    function callFunc(string calldata name, bytes calldata input) external returns (bytes memory);
}

contract UpgradeConsistencyProbe {
    address internal constant CODE_STORAGE = 0x0000000000000000000000000000000000000043;

    uint256 public lastOutput;
    event ProbeExecuted(uint256 indexed blockNumber, uint256 output);

    function probe(uint256 left, uint256 right) external returns (uint256 output) {
        bytes memory raw = ICodeStorageConsistency(CODE_STORAGE).callFunc(
            "UpgradeConsistencyProbe",
            abi.encode(left, right)
        );
        output = abi.decode(raw, (uint256));
        lastOutput = output;
        emit ProbeExecuted(block.number, output);
    }
}
