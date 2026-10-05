/* CGO 单独编译 blst server.c。x86_64 的 no_asm 路径需要双宽度 llimb_t。 */
#if defined(__x86_64__) || defined(__aarch64__)
typedef unsigned __int128 llimb_t;
#endif
#include "server.c"
