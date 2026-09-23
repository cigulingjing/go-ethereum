/* CGO 只编译本目录 .c，这里汇总与 WASM 构建相同的 Aigis-sig2 源文件。 */
#define USE_SHAKE 1

#include "ntt.c"
#include "packing.c"
#include "poly.c"
#include "polyvec.c"
#include "reduce.c"
#include "rounding.c"
#include "sign.c"
#include "fips202.c"
#include "randombytes.c"
