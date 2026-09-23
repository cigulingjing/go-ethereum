#ifndef PQMAGIC_CONFIG_H
#define PQMAGIC_CONFIG_H

/* 四个验签算法共用 SHAKE 符号前缀；由各模块 CFLAGS 提供 PQC_NAMESPACE。 */
#ifndef PQC_NAMESPACE
#error "define PQC_NAMESPACE, e.g. -DPQC_NAMESPACE=aigis_sig2_verify"
#endif
#define PQC_NS_CONCAT(a, b) a##_##b
#define PQC_NS_EXPAND(a, b) PQC_NS_CONCAT(a, b)
#define GLOBAL_NAMESPACE(s) PQC_NS_EXPAND(PQC_NAMESPACE, s)

#endif
