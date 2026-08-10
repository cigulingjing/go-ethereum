// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

import "./BigMod.sol";

contract PedersenCommitContract is BigMod {
    function PedersenCommit(bytes calldata message, bytes calldata blinding) external view returns (bytes memory) {
        return pedersenCommit(message, blinding);
    }

    function pedersenCommit(bytes calldata message, bytes calldata blinding) private view returns (bytes memory) {
        bytes memory p = dh2048Prime();
        bytes memory h = pedersenH(p);
        bytes memory gm = modExp(hex"04", message, p);
        bytes memory hr = modExp(h, blinding, p);
        return mulModBytes(gm, hr, p);
    }
}
