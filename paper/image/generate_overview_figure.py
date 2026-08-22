import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib.patches import Circle, FancyArrowPatch, FancyBboxPatch


mpl.rcParams.update(
    {
        "font.family": "sans-serif",
        "font.sans-serif": ["DejaVu Sans Condensed", "Arial", "Helvetica", "DejaVu Sans", "sans-serif"],
        "font.size": 5.4,
        "svg.fonttype": "none",
        "pdf.fonttype": 42,
        "axes.linewidth": 0.8,
    }
)


RED = "#D62728"
RED_EDGE = "#E05A59"
RED_FILL = "#FFF2F2"
BLUE = "#1D5FE7"
BLUE_EDGE = "#3478F6"
BLUE_FILL = "#EFF7FF"
GREEN = "#1E7A4D"
GREEN_EDGE = "#2E9B67"
GREEN_FILL = "#F1FBF4"
PURPLE_EDGE = "#6358C7"
PURPLE_FILL = "#F3F0FF"
AMBER_EDGE = "#B7791F"
AMBER_FILL = "#FFF7DF"
BROWN = "#7A4A24"
BROWN_FILL = "#FFF9EE"
STATE_EDGE = "#2E8B62"
STATE_FILL = "#ECFFF5"
TEXT = "#1F2937"
BLACK = "#111827"


def rounded_box(
    ax,
    x,
    y,
    w,
    h,
    text=None,
    *,
    fc="white",
    ec="#777777",
    lw=0.9,
    radius=0.008,
    fontsize=5.4,
    weight="normal",
    color=TEXT,
    ha="center",
    va="center",
    linespacing=1.15,
):
    patch = FancyBboxPatch(
        (x, y),
        w,
        h,
        boxstyle=f"round,pad=0.004,rounding_size={radius}",
        linewidth=lw,
        edgecolor=ec,
        facecolor=fc,
    )
    ax.add_patch(patch)
    if text:
        ax.text(
            x + w / 2,
            y + h / 2,
            text,
            ha=ha,
            va=va,
            fontsize=fontsize,
            weight=weight,
            color=color,
            linespacing=linespacing,
        )
    return patch


def text(ax, x, y, s, *, size=5.4, weight="normal", color=TEXT, ha="center", va="center"):
    ax.text(x, y, s, fontsize=size, weight=weight, color=color, ha=ha, va=va)


def arrow(ax, start, end, *, color=BLACK, lw=1.35, style="-", scale=10, rad=0.0, zorder=6):
    arr = FancyArrowPatch(
        start,
        end,
        arrowstyle="-|>",
        mutation_scale=scale,
        linewidth=lw,
        linestyle=style,
        color=color,
        connectionstyle=f"arc3,rad={rad}",
        shrinkA=0,
        shrinkB=0,
        zorder=zorder,
    )
    ax.add_patch(arr)
    return arr


def elbow_arrow(ax, points, *, color=BLACK, lw=1.2, style="-", scale=9, zorder=6):
    if len(points) < 2:
        return None
    if len(points) > 2:
        xs = [p[0] for p in points[:-1]]
        ys = [p[1] for p in points[:-1]]
        ax.plot(xs, ys, color=color, lw=lw, linestyle=style, zorder=zorder)
    return arrow(
        ax,
        points[-2],
        points[-1],
        color=color,
        lw=lw,
        style=style,
        scale=scale,
        zorder=zorder,
    )


def vertical_flow(ax, x, ys, *, color=BLACK, lw=1.0):
    for y0, y1 in zip(ys[:-1], ys[1:]):
        arrow(ax, (x, y0), (x, y1), color=color, lw=lw, scale=7)


def draw_developer(ax):
    rounded_box(ax, 0.035, 0.700, 0.085, 0.125, fc=BLUE_FILL, ec="#254C7C", lw=0.8, radius=0.007)
    ax.add_patch(Circle((0.078, 0.785), 0.010, facecolor="#8EB5D9", edgecolor="#12355B", lw=0.7))
    rounded_box(ax, 0.060, 0.735, 0.036, 0.022, fc="#8EB5D9", ec="#12355B", lw=0.7, radius=0.010)
    text(ax, 0.078, 0.717, "Developer", size=5.2, weight="bold")


def draw_management_contract(ax):
    rounded_box(ax, 0.235, 0.620, 0.245, 0.285, fc=BROWN_FILL, ec=BROWN, lw=0.85, radius=0.008)
    text(ax, 0.3575, 0.884, "Management Contract", size=5.8, weight="bold")
    rounded_box(ax, 0.246, 0.829, 0.223, 0.042, "Upgrade Interface", fc="#FFFCF6", ec=BROWN, lw=0.7, radius=0.006, fontsize=5.3, weight="bold")
    rounded_box(ax, 0.246, 0.772, 0.223, 0.042, "Invocation Interface", fc="#FFFCF6", ec=BROWN, lw=0.7, radius=0.006, fontsize=5.3, weight="bold")
    rounded_box(ax, 0.246, 0.638, 0.223, 0.113, fc="#FFFCF6", ec=BROWN, lw=0.7, radius=0.006)
    text(ax, 0.3575, 0.733, "Version Registry", size=5.3, weight="bold")
    ax.text(
        0.263,
        0.708,
        "\u2022  algorithm\n\u2022  input type\n\u2022  output type\n\u2022  activation block",
        ha="left",
        va="top",
        fontsize=5.0,
        color=TEXT,
        linespacing=1.0,
    )


def draw_top_layer(ax):
    rounded_box(ax, 0.210, 0.555, 0.620, 0.405, fc=RED_FILL, ec=RED_EDGE, lw=0.8, radius=0.010)
    text(ax, 0.520, 0.932, "On-chain Layer", size=7.0, weight="bold", color="#A01818")
    draw_management_contract(ax)
    rounded_box(ax, 0.585, 0.710, 0.090, 0.130, "Event Log", fc=PURPLE_FILL, ec=PURPLE_EDGE, lw=0.8, radius=0.007, fontsize=5.4, weight="bold")
    rounded_box(ax, 0.720, 0.710, 0.095, 0.130, "On-chain\nState", fc=STATE_FILL, ec=STATE_EDGE, lw=0.8, radius=0.007, fontsize=5.4, weight="bold")
    rounded_box(ax, 0.485, 0.535, 0.125, 0.040, "Activation Block", fc=AMBER_FILL, ec=AMBER_EDGE, lw=0.8, radius=0.006, fontsize=5.1, weight="bold")


def draw_execution_engine(ax):
    rounded_box(ax, 0.088, 0.165, 0.185, 0.270, fc="#F8FCFF", ec=BLUE_EDGE, lw=0.8, radius=0.007)
    text(ax, 0.1805, 0.411, "Execution Engine", size=5.5, weight="bold")
    rounded_box(ax, 0.102, 0.349, 0.157, 0.045, "EVM", fc="#F8FBFF", ec="#24448A", lw=0.7, radius=0.006, fontsize=5.3, weight="bold")
    rounded_box(ax, 0.102, 0.273, 0.157, 0.045, "Contract Execution", fc="#F8FBFF", ec="#24448A", lw=0.7, radius=0.006, fontsize=5.1, weight="bold")
    rounded_box(ax, 0.102, 0.197, 0.157, 0.045, "Precompile Interface", fc="#F8FBFF", ec="#24448A", lw=0.7, radius=0.006, fontsize=5.1, weight="bold")


def draw_invocation_path(ax):
    rounded_box(ax, 0.340, 0.130, 0.250, 0.300, fc="#F7FFF8", ec=GREEN_EDGE, lw=0.75, radius=0.007)
    text(ax, 0.465, 0.407, "Invocation Path", size=5.2, weight="bold", color=GREEN)
    rounded_box(ax, 0.355, 0.357, 0.220, 0.037, "Invocation Dispatcher", fc="#FCFFFC", ec=GREEN_EDGE, lw=0.7, radius=0.005, fontsize=5.0, weight="bold")
    rounded_box(ax, 0.355, 0.285, 0.220, 0.037, "Version Manager", fc="#FCFFFC", ec=GREEN_EDGE, lw=0.7, radius=0.005, fontsize=5.0, weight="bold")
    rounded_box(ax, 0.355, 0.213, 0.220, 0.037, "Module Manager", fc="#FCFFFC", ec=GREEN_EDGE, lw=0.7, radius=0.005, fontsize=5.0, weight="bold")
    rounded_box(ax, 0.355, 0.136, 0.220, 0.055, fc="#FCFFFC", ec=GREEN_EDGE, lw=0.7, radius=0.005)
    text(ax, 0.465, 0.178, "Native Modules", size=5.0, weight="bold")
    chip_x = [0.363, 0.411, 0.459, 0.507, 0.555]
    chip_text = ["VDF", "BLS", "KZG", "Curve", "..."]
    for x, label in zip(chip_x, chip_text):
        rounded_box(ax, x, 0.144, 0.039, 0.028, label, fc=GREEN_FILL, ec=GREEN_EDGE, lw=0.65, radius=0.004, fontsize=5.0)

    arrow(ax, (0.465, 0.357), (0.465, 0.322), color=BLUE, lw=1.2, scale=8)
    arrow(ax, (0.465, 0.285), (0.465, 0.250), color=BLUE, lw=1.2, scale=8)
    arrow(ax, (0.465, 0.213), (0.465, 0.191), color=BLUE, lw=1.2, scale=8)


def draw_upgrade_handler(ax):
    rounded_box(ax, 0.625, 0.130, 0.155, 0.300, fc="#F7FFF8", ec=GREEN_EDGE, lw=0.75, radius=0.007)
    text(ax, 0.7025, 0.407, "Upgrade Handler", size=5.2, weight="bold", color=GREEN)
    labels = ["Event Listener", "Decompress", "Restore Source", "Compile", "Verify", "Load Module"]
    ys = [0.357, 0.313, 0.269, 0.225, 0.181, 0.137]
    for y, label in zip(ys, labels):
        rounded_box(ax, 0.638, y, 0.129, 0.031, label, fc="#FCFFFC", ec=GREEN_EDGE, lw=0.65, radius=0.004, fontsize=5.0)
    for y0, y1 in [(0.357, 0.344), (0.313, 0.300), (0.269, 0.256), (0.225, 0.212), (0.181, 0.168)]:
        arrow(ax, (0.7025, y0), (0.7025, y1), color=BLACK, lw=0.75, scale=7)


def draw_bottom_layer(ax):
    rounded_box(ax, 0.070, 0.070, 0.820, 0.430, fc=BLUE_FILL, ec=BLUE_EDGE, lw=0.8, radius=0.010)
    text(ax, 0.095, 0.475, "Blockchain Client", size=6.0, weight="bold", color="#1457C8", ha="left")
    draw_execution_engine(ax)
    rounded_box(ax, 0.318, 0.095, 0.480, 0.370, fc=GREEN_FILL, ec=GREEN_EDGE, lw=0.8, radius=0.008)
    text(ax, 0.558, 0.440, "Cryptographic Coprocessor", size=6.1, weight="bold", color=GREEN)
    draw_invocation_path(ax)
    draw_upgrade_handler(ax)


def draw_cross_layer_flows(ax):
    # Developer-driven contract operations.
    arrow(ax, (0.120, 0.795), (0.246, 0.850), color=RED, lw=1.45, scale=10)
    text(ax, 0.165, 0.835, "upgrade", size=5.0, color=BLACK)
    arrow(ax, (0.120, 0.740), (0.246, 0.793), color=BLUE, lw=1.45, scale=10)
    text(ax, 0.166, 0.776, "callFunc", size=5.0, color=BLACK)

    # On-chain publication and event delivery.
    arrow(ax, (0.469, 0.850), (0.585, 0.775), color=RED, lw=1.45, scale=10)
    text(ax, 0.525, 0.816, "emit", size=5.0, color=BLACK)
    elbow_arrow(
        ax,
        [(0.630, 0.710), (0.630, 0.470), (0.785, 0.470), (0.785, 0.3725), (0.767, 0.3725)],
        color=BLACK,
        lw=1.0,
        style="--",
        scale=8,
    )
    text(ax, 0.665, 0.556, "event", size=5.0, color=BLACK, ha="left")

    # Activation metadata binds chain height to local version selection.
    elbow_arrow(ax, [(0.3575, 0.638), (0.3575, 0.555), (0.485, 0.555)], color=BLACK, lw=1.0, style="--", scale=8)
    elbow_arrow(ax, [(0.630, 0.710), (0.630, 0.555), (0.610, 0.555)], color=BLACK, lw=1.0, style="--", scale=8)
    text(ax, 0.515, 0.510, "activate at target block", size=5.0, color=BLUE, ha="left")
    elbow_arrow(
        ax,
        [(0.548, 0.535), (0.548, 0.492), (0.598, 0.492), (0.598, 0.304), (0.575, 0.304)],
        color=BLUE,
        lw=1.0,
        style="--",
        scale=8,
    )
    elbow_arrow(ax, [(0.598, 0.292), (0.598, 0.232), (0.575, 0.232)], color=BLUE, lw=1.0, style="--", scale=8)

    # Invocation and module-loading routes inside the execution client.
    elbow_arrow(ax, [(0.259, 0.2195), (0.305, 0.2195), (0.305, 0.3755), (0.355, 0.3755)], color=BLUE, lw=1.25, scale=9)
    elbow_arrow(ax, [(0.638, 0.1525), (0.603, 0.1525), (0.575, 0.164)], color=BLACK, lw=1.0, scale=8)


def draw_legend(ax):
    rounded_box(ax, 0.895, 0.645, 0.130, 0.160, fc="white", ec="#9CA3AF", lw=0.65, radius=0.006)
    arrow(ax, (0.910, 0.768), (0.945, 0.768), color=RED, lw=1.4, scale=8)
    text(ax, 0.958, 0.768, "Upgrade flow", size=5.0, ha="left")
    arrow(ax, (0.910, 0.723), (0.945, 0.723), color=BLUE, lw=1.4, scale=8)
    text(ax, 0.958, 0.723, "Invocation flow", size=5.0, ha="left")
    arrow(ax, (0.910, 0.678), (0.945, 0.678), color=BLACK, lw=1.2, style="--", scale=8)
    text(ax, 0.958, 0.678, "Event flow", size=5.0, ha="left")


def main():
    fig, ax = plt.subplots(figsize=(7.2, 4.8), dpi=300)
    ax.set_xlim(0, 1.04)
    ax.set_ylim(0, 1)
    ax.axis("off")
    fig.patch.set_facecolor("white")
    ax.set_facecolor("white")

    draw_developer(ax)
    draw_top_layer(ax)
    draw_bottom_layer(ax)
    draw_cross_layer_flows(ax)
    draw_legend(ax)

    fig.savefig("image/overview.png", dpi=300, bbox_inches="tight", facecolor="white")
    fig.savefig("image/overview.svg", bbox_inches="tight", facecolor="white")
    fig.savefig("image/overview.pdf", bbox_inches="tight", facecolor="white")
    fig.savefig(
        "image/overview.tiff",
        dpi=600,
        bbox_inches="tight",
        facecolor="white",
        pil_kwargs={"compression": "tiff_lzw"},
    )


if __name__ == "__main__":
    main()
