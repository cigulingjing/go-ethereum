/* CGO 不允许 -D 值中带连字符，把 SLH_DSA_MODE 写进归档头文件。 */
#ifndef SLH_DSA_MODE
#define SLH_DSA_MODE slh-dsa-shake-192f
#endif
#ifndef THASH
#define THASH simple
#endif
#ifndef SLH_DSA_HASH_MODE_NAMESPACE
#define SLH_DSA_HASH_MODE_NAMESPACE shake_192f
#endif
