// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

abstract contract BigMod {
    function modExp(bytes memory base, bytes memory exponent, bytes memory modulus) internal view returns (bytes memory result) {
        if (exponent.length == 0) {
            exponent = hex"00";
        }
        bytes memory input = abi.encodePacked(uint256(base.length), uint256(exponent.length), uint256(modulus.length), base, exponent, modulus);
        result = new bytes(modulus.length);
        bool success;
        assembly {
            success := staticcall(gas(), 0x05, add(input, 32), mload(input), add(result, 32), mload(modulus))
        }
        require(success, "modexp failed");
    }

    function mulModBytes(bytes memory a, bytes memory b, bytes memory modulus) internal view returns (bytes memory) {
        bytes memory two = hex"02";
        bytes memory sum = addModBytes(a, b, modulus);
        bytes memory sumSquare = modExp(sum, two, modulus);
        bytes memory aSquare = modExp(a, two, modulus);
        bytes memory bSquare = modExp(b, two, modulus);
        bytes memory doubled = subModBytes(subModBytes(sumSquare, aSquare, modulus), bSquare, modulus);
        return halfModBytes(doubled, modulus);
    }

    function addModBytes(bytes memory a, bytes memory b, bytes memory modulus) internal pure returns (bytes memory) {
        (bytes memory sum, bool carry) = addSameLength(a, b);
        if (carry) {
            bytes memory wide = new bytes(a.length + 1);
            wide[0] = 0x01;
            copyBytes(sum, 0, wide, 1, sum.length);
            bytes memory wideModulus = new bytes(a.length + 1);
            copyBytes(modulus, 0, wideModulus, 1, modulus.length);
            bytes memory reduced = subSameLength(wide, wideModulus);
            return sliceMemory(reduced, 1, a.length);
        }
        if (compareBytes(sum, modulus) >= 0) {
            return subSameLength(sum, modulus);
        }
        return sum;
    }

    function subModBytes(bytes memory a, bytes memory b, bytes memory modulus) internal pure returns (bytes memory) {
        if (compareBytes(a, b) >= 0) {
            return subSameLength(a, b);
        }
        bytes memory diff = subSameLength(b, a);
        return subSameLength(modulus, diff);
    }

    function halfModBytes(bytes memory value, bytes memory modulus) internal pure returns (bytes memory) {
        if (uint8(value[value.length - 1]) & 1 == 0) {
            return shiftRightOne(value, value.length);
        }
        (bytes memory sum, bool carry) = addSameLength(value, modulus);
        bytes memory wide = new bytes(value.length + 1);
        wide[0] = carry ? bytes1(uint8(1)) : bytes1(0);
        copyBytes(sum, 0, wide, 1, sum.length);
        return shiftRightOne(wide, value.length);
    }

    function addSameLength(bytes memory a, bytes memory b) internal pure returns (bytes memory sum, bool carryOut) {
        require(a.length == b.length, "length mismatch");
        sum = new bytes(a.length);
        uint256 carry;
        for (uint256 i = a.length; i > 0; i--) {
            uint256 v = uint8(a[i - 1]) + uint8(b[i - 1]) + carry;
            sum[i - 1] = bytes1(uint8(v));
            carry = v >> 8;
        }
        carryOut = carry != 0;
    }

    function subSameLength(bytes memory a, bytes memory b) internal pure returns (bytes memory diff) {
        require(a.length == b.length, "length mismatch");
        require(compareBytes(a, b) >= 0, "negative result");
        diff = new bytes(a.length);
        uint256 borrow;
        for (uint256 i = a.length; i > 0; i--) {
            uint256 av = uint8(a[i - 1]);
            uint256 bv = uint8(b[i - 1]) + borrow;
            if (av >= bv) {
                diff[i - 1] = bytes1(uint8(av - bv));
                borrow = 0;
            } else {
                diff[i - 1] = bytes1(uint8(256 + av - bv));
                borrow = 1;
            }
        }
    }

    function shiftRightOne(bytes memory value, uint256 outLength) internal pure returns (bytes memory out) {
        bytes memory shifted = new bytes(value.length);
        uint256 carry;
        for (uint256 i = 0; i < value.length; i++) {
            uint256 v = uint8(value[i]);
            shifted[i] = bytes1(uint8((carry << 7) | (v >> 1)));
            carry = v & 1;
        }
        out = sliceMemory(shifted, shifted.length - outLength, outLength);
    }

    function validScalar(bytes memory x) internal pure returns (bool) {
        bytes memory padded = leftPad(x, 256);
        return compareBytes(padded, oneBytes(256)) > 0 && compareBytes(padded, dh2048PMinusOne()) < 0;
    }

    function groupElement(bytes memory x) internal pure returns (bool) {
        return x.length == 256 && compareBytes(x, oneBytes(256)) > 0 && compareBytes(x, dh2048PMinusOne()) < 0;
    }

    function pedersenH(bytes memory p) internal view returns (bytes memory) {
        return modExp(abi.encodePacked(sha256(abi.encodePacked("cryptoupgrade-pedersen-h-", bytes1(0)))), hex"02", p);
    }

    function compareBytes(bytes memory a, bytes memory b) internal pure returns (int256) {
        require(a.length == b.length, "length mismatch");
        for (uint256 i = 0; i < a.length; i++) {
            if (uint8(a[i]) < uint8(b[i])) {
                return -1;
            }
            if (uint8(a[i]) > uint8(b[i])) {
                return 1;
            }
        }
        return 0;
    }

    function leftPad(bytes memory value, uint256 length) internal pure returns (bytes memory out) {
        require(value.length <= length, "too long");
        out = new bytes(length);
        copyBytes(value, 0, out, length - value.length, value.length);
    }

    function oneBytes(uint256 length) internal pure returns (bytes memory out) {
        out = new bytes(length);
        out[length - 1] = 0x01;
    }

    function copyBytes(bytes memory src, uint256 srcOffset, bytes memory dst, uint256 dstOffset, uint256 length) internal pure {
        for (uint256 i = 0; i < length; i++) {
            dst[dstOffset + i] = src[srcOffset + i];
        }
    }

    function sliceMemory(bytes memory data, uint256 offset, uint256 length) internal pure returns (bytes memory out) {
        require(offset + length <= data.length, "slice out of bounds");
        out = new bytes(length);
        copyBytes(data, offset, out, 0, length);
    }

    function sliceCalldata(bytes calldata data, uint256 offset, uint256 length) internal pure returns (bytes memory out) {
        require(offset + length <= data.length, "slice out of bounds");
        out = new bytes(length);
        for (uint256 i = 0; i < length; i++) {
            out[i] = data[offset + i];
        }
    }

    function dh2048Prime() internal pure returns (bytes memory) {
        return hex"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF";
    }

    function dh2048Q() internal pure returns (bytes memory) {
        return hex"7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF";
    }

    function dh2048PMinusOne() internal pure returns (bytes memory) {
        return hex"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFE";
    }
}
