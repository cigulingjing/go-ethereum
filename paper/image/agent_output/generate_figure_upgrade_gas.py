#!/usr/bin/env python3
"""Generate figure_upgrade_gas_comparison (EvoCrypt vs Solidity upgrade gas).

数据源为 paper/实验数据.xlsx 的“升级消耗Gas”工作表；输出 PDF/PNG/SVG 到本脚本
所在目录，并在导出前执行多面板对齐门禁（单面板记录 NOT APPLICABLE）。
历史版本的纵轴标签曾被重复绘制导致 collision audit FAIL，本脚本保证
set_ylabel 只调用一次。
"""

from __future__ import annotations

import os
import sys
from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
import numpy as np
from openpyxl import load_workbook

OUT_DIR = Path(__file__).resolve().parent
# 数据源为 paper/image/实验数据.xlsx：paper/ 下的同名旧文件中“升级消耗Gas”表
# 被误填为与“执行消耗gas”完全相同的数值，本文件中的升级 gas 才是真实测量值
# （与 experiments/cryptoupgrade/results/tx-gas 原始回执一致）。
DATA_XLSX = OUT_DIR.parent / "实验数据.xlsx"
STEM = "figure_upgrade_gas_comparison"

# 与论文其余实验图保持同一调色板：EvoCrypt 为信号色（hero），Solidity 为中性基线。
PALETTE = {
    "evocrypt": "#0F4D92",
    "solidity": "#767676",
}

ALGO_DISPLAY = {
    "Add": "Add",
    "Sha256": "SHA-256",
    "Blake2bSum256": "BLAKE2b",
    "Pbkdf2Sha256": "PBKDF2",
    "Dh2048Secret": "DH-2048",
    "PedersenCommit": "Pedersen",
    "SchnorrVerify": "Schnorr",
    "PolynomialMul": "PolyMul",
}


def apply_style() -> None:
    mpl.rcParams.update(
        {
            "font.family": "sans-serif",
            "font.sans-serif": ["Arial", "Helvetica", "DejaVu Sans", "sans-serif"],
            "svg.fonttype": "none",
            "pdf.fonttype": 42,
            "font.size": 7,
            "axes.spines.right": False,
            "axes.spines.top": False,
            "axes.linewidth": 0.8,
            "legend.frameon": False,
            "xtick.major.width": 0.8,
            "ytick.major.width": 0.8,
        }
    )


def load_upgrade_gas() -> tuple[list[str], list[float], list[float]]:
    wb = load_workbook(DATA_XLSX, data_only=True, read_only=True)
    ws = wb["升级消耗Gas"]
    algos, upgrade, contract = [], [], []
    for row in list(ws.iter_rows(values_only=True))[1:]:
        if row is None or len(row) < 3 or row[0] is None:
            continue
        algos.append(str(row[0]).strip())
        upgrade.append(float(str(row[1]).strip().replace(",", "")))
        contract.append(float(str(row[2]).strip().replace(",", "")))
    wb.close()
    return algos, upgrade, contract


def main() -> None:
    if not DATA_XLSX.exists():
        raise FileNotFoundError(f"Missing data file: {DATA_XLSX}")

    apply_style()
    algos, upgrade_gas, contract_gas = load_upgrade_gas()
    labels = [ALGO_DISPLAY.get(a, a) for a in algos]

    n = len(labels)
    series = [
        ("EvoCrypt", upgrade_gas, PALETTE["evocrypt"]),
        ("Solidity", contract_gas, PALETTE["solidity"]),
    ]
    n_series = len(series)
    width = 0.78 / n_series
    x = np.arange(n)

    # 画布尺寸与论文中既有版本一致（328.32 x 187.2 pt），保证版式可互换。
    fig, ax = plt.subplots(figsize=(max(3.4, 0.42 * n + 1.2), 2.6))

    # 对数轴上从 0 起画的 bar 会生成延伸到无穷远的矩形路径（log(0) 无定义），
    # 其白色描边在 PDF 几何中穿过坐标轴下方的刻度标签。改为从轴下限起画，
    # 视觉不变但路径有限。上下限按 matplotlib 对数默认边距（log 空间 5%）确定。
    all_values = [v for _, vals, _ in series for v in vals]
    log_lo, log_hi = np.log10(min(all_values)), np.log10(max(all_values))
    margin = 0.05 * (log_hi - log_lo)
    y_lo, y_hi = float(10 ** (log_lo - margin)), float(10 ** (log_hi + margin))

    for idx, (name, values, color) in enumerate(series):
        offset = (idx - (n_series - 1) / 2) * width
        ax.bar(
            x + offset,
            [v - y_lo for v in values],
            width=width,
            bottom=y_lo,
            label=name,
            color=color,
            edgecolor="white",
            linewidth=0.4,
            zorder=3,
        )

    ax.set_xticks(x)
    ax.set_xticklabels(labels, rotation=35, ha="right", rotation_mode="anchor")
    # 分类轴不需要刻度线；保留刻度线会穿过旋转后标签的包围盒。
    ax.tick_params(axis="x", which="both", length=0)
    # 纵轴标签只设置一次：历史版本重复绘制 "Upgrade gas" 导致 collision FAIL。
    ax.set_ylabel("Upgrade gas")
    ax.set_yscale("log")
    ax.set_ylim(y_lo, y_hi)
    # 对数轴使用纯文本刻度标签：mathtext 上下标会以约 0.7 倍字号渲染，
    # 跌破 5 pt 字号下限；数据均为正，无需非正数保护。
    ax.yaxis.set_major_formatter(
        mpl.ticker.FuncFormatter(lambda v, _: f"{v / 1e6:g}M" if v >= 1e6 else f"{v / 1e3:g}k")
    )
    ax.yaxis.set_minor_formatter(mpl.ticker.NullFormatter())
    ax.grid(axis="y", color="#E6E6E6", linewidth=0.6, zorder=0)
    ax.legend(loc="upper left", bbox_to_anchor=(0, 1.02), ncol=n_series, handlelength=1.2)

    fig.tight_layout()

    # 导出前执行多面板对齐门禁：单面板图记录 NOT APPLICABLE。
    scripts_dir = os.environ.get(
        "NATURE_FIGURE_SCRIPTS",
        str(Path.home() / ".agents" / "skills" / "nature-figure" / "scripts"),
    )
    if scripts_dir not in sys.path:
        sys.path.insert(0, scripts_dir)
    from audit_panel_alignment import require_matplotlib_panel_alignment

    require_matplotlib_panel_alignment(
        fig,
        json_out=OUT_DIR / f"{STEM}.alignment.json",
        tolerance_pt=1.5,
        gutter_tolerance_pt=1.5,
        strict=True,
    )

    fig.savefig(OUT_DIR / f"{STEM}.pdf", bbox_inches="tight")
    fig.savefig(OUT_DIR / f"{STEM}.png", dpi=600, bbox_inches="tight")
    fig.savefig(OUT_DIR / f"{STEM}.svg", bbox_inches="tight")
    plt.close(fig)
    print(f"Wrote {STEM}.pdf, {STEM}.png, {STEM}.svg into {OUT_DIR}")


if __name__ == "__main__":
    main()
