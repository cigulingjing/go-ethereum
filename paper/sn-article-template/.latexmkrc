# sn-mathphys-num.bst 等样式在 ./bst，不在系统 TeX 树中。
# 不设置 BSTINPUTS 时 bibtex 失败、不生成 .bbl，natbib 会报 Citation undefined。
ensure_path('BSTINPUTS', './bst');
