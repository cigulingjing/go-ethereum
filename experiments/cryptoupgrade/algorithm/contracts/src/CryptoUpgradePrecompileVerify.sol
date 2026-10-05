// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

/// @notice Solidity-side adapter used to exercise cryptoupgrade precompiles.
/// The precompile receives ABI-encoded bytes arguments and returns ABI bool.
contract CryptoUpgradePrecompileVerify {
    function verify(
        address precompile,
        bytes calldata first,
        bytes calldata second,
        bytes calldata third
    ) external view returns (bool) {
        (bool success, bytes memory output) = precompile.staticcall(abi.encode(first, second, third));
        require(success, "cryptoupgrade precompile call failed");
        require(output.length == 32, "invalid cryptoupgrade precompile output");
        return abi.decode(output, (bool));
    }
}
