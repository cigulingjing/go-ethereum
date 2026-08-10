// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract Blake2b {
    uint64 private constant IV0 = 0x6a09e667f3bcc908;
    uint64 private constant IV1 = 0xbb67ae8584caa73b;
    uint64 private constant IV2 = 0x3c6ef372fe94f82b;
    uint64 private constant IV3 = 0xa54ff53a5f1d36f1;
    uint64 private constant IV4 = 0x510e527fade682d1;
    uint64 private constant IV5 = 0x9b05688c2b3e6c1f;
    uint64 private constant IV6 = 0x1f83d9abfb41bd6b;
    uint64 private constant IV7 = 0x5be0cd19137e2179;

    function Sum256(bytes calldata data) external pure returns (bytes32) {
        require(data.length <= 128, "input too long");

        uint64[8] memory h;
        h[0] = IV0 ^ uint64(32 | (1 << 16) | (1 << 24));
        h[1] = IV1;
        h[2] = IV2;
        h[3] = IV3;
        h[4] = IV4;
        h[5] = IV5;
        h[6] = IV6;
        h[7] = IV7;

        uint64[16] memory m;
        for (uint256 i = 0; i < 16; i++) {
            m[i] = loadLE64(data, i * 8);
        }

        compress(h, m, uint64(data.length));

        return bytes32(
            (uint256(reverse64(h[0])) << 192) |
            (uint256(reverse64(h[1])) << 128) |
            (uint256(reverse64(h[2])) << 64) |
            uint256(reverse64(h[3]))
        );
    }

    function loadLE64(bytes calldata data, uint256 offset) private pure returns (uint64 v) {
        unchecked {
            for (uint256 i = 0; i < 8; i++) {
                uint256 pos = offset + i;
                if (pos < data.length) {
                    v |= uint64(uint8(data[pos])) << uint64(i * 8);
                }
            }
        }
    }

    function compress(uint64[8] memory h, uint64[16] memory m, uint64 length) private pure {
        uint64 v0 = h[0];
        uint64 v1 = h[1];
        uint64 v2 = h[2];
        uint64 v3 = h[3];
        uint64 v4 = h[4];
        uint64 v5 = h[5];
        uint64 v6 = h[6];
        uint64 v7 = h[7];
        uint64 v8 = IV0;
        uint64 v9 = IV1;
        uint64 v10 = IV2;
        uint64 v11 = IV3;
        uint64 v12 = IV4 ^ length;
        uint64 v13 = IV5;
        uint64 v14 = IV6 ^ type(uint64).max;
        uint64 v15 = IV7;

        for (uint256 r = 0; r < 12; r++) {
            uint8[16] memory s = sigma(r);
            unchecked {
                (v0, v4, v8, v12) = g(v0, v4, v8, v12, m[s[0]], m[s[4]]);
                (v1, v5, v9, v13) = g(v1, v5, v9, v13, m[s[1]], m[s[5]]);
                (v2, v6, v10, v14) = g(v2, v6, v10, v14, m[s[2]], m[s[6]]);
                (v3, v7, v11, v15) = g(v3, v7, v11, v15, m[s[3]], m[s[7]]);
                (v0, v5, v10, v15) = g(v0, v5, v10, v15, m[s[8]], m[s[12]]);
                (v1, v6, v11, v12) = g(v1, v6, v11, v12, m[s[9]], m[s[13]]);
                (v2, v7, v8, v13) = g(v2, v7, v8, v13, m[s[10]], m[s[14]]);
                (v3, v4, v9, v14) = g(v3, v4, v9, v14, m[s[11]], m[s[15]]);
            }
        }

        unchecked {
            h[0] ^= v0 ^ v8;
            h[1] ^= v1 ^ v9;
            h[2] ^= v2 ^ v10;
            h[3] ^= v3 ^ v11;
            h[4] ^= v4 ^ v12;
            h[5] ^= v5 ^ v13;
            h[6] ^= v6 ^ v14;
            h[7] ^= v7 ^ v15;
        }
    }

    function g(uint64 a, uint64 b, uint64 c, uint64 d, uint64 x, uint64 y)
        private
        pure
        returns (uint64, uint64, uint64, uint64)
    {
        unchecked {
            a = a + b + x;
            d = rotr64(d ^ a, 32);
            c = c + d;
            b = rotr64(b ^ c, 24);
            a = a + b + y;
            d = rotr64(d ^ a, 16);
            c = c + d;
            b = rotr64(b ^ c, 63);
            return (a, b, c, d);
        }
    }

    function rotr64(uint64 x, uint64 n) private pure returns (uint64) {
        unchecked {
            return (x >> n) | (x << (64 - n));
        }
    }

    function reverse64(uint64 x) private pure returns (uint64) {
        unchecked {
            return
                ((x & 0x00000000000000ff) << 56) |
                ((x & 0x000000000000ff00) << 40) |
                ((x & 0x0000000000ff0000) << 24) |
                ((x & 0x00000000ff000000) << 8) |
                ((x & 0x000000ff00000000) >> 8) |
                ((x & 0x0000ff0000000000) >> 24) |
                ((x & 0x00ff000000000000) >> 40) |
                ((x & 0xff00000000000000) >> 56);
        }
    }

    function sigma(uint256 r) private pure returns (uint8[16] memory s) {
        uint256 row = r % 10;
        if (row == 0) return [uint8(0), 2, 4, 6, 1, 3, 5, 7, 8, 10, 12, 14, 9, 11, 13, 15];
        if (row == 1) return [uint8(14), 4, 9, 13, 10, 8, 15, 6, 1, 0, 11, 5, 12, 2, 7, 3];
        if (row == 2) return [uint8(11), 12, 5, 15, 8, 0, 2, 13, 10, 3, 7, 9, 14, 6, 1, 4];
        if (row == 3) return [uint8(7), 3, 13, 11, 9, 1, 12, 14, 2, 5, 4, 15, 6, 10, 0, 8];
        if (row == 4) return [uint8(9), 5, 2, 10, 0, 7, 4, 15, 14, 11, 6, 3, 1, 12, 8, 13];
        if (row == 5) return [uint8(2), 6, 0, 8, 12, 10, 11, 3, 4, 7, 15, 1, 13, 5, 14, 9];
        if (row == 6) return [uint8(12), 1, 14, 4, 5, 15, 13, 10, 0, 6, 9, 8, 7, 3, 2, 11];
        if (row == 7) return [uint8(13), 7, 12, 3, 11, 14, 1, 9, 5, 15, 8, 2, 0, 4, 6, 10];
        if (row == 8) return [uint8(6), 14, 11, 0, 15, 9, 3, 8, 12, 13, 1, 10, 2, 7, 4, 5];
        return [uint8(10), 8, 7, 1, 2, 4, 6, 5, 15, 9, 3, 13, 11, 14, 12, 0];
    }
}
