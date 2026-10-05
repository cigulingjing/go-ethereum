/* CGO 只编译本目录 .c，这里汇总与 WASM 构建相同的 Aigis-sig2 源文件。 */
#define USE_SHAKE 1

#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/ntt.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/packing.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/poly.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/polyvec.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/reduce.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/rounding.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/sign.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/fips202.c"
#include "../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/randombytes.c"
