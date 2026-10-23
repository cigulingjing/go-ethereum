"""Fig. 5：等宽双图与绘图区外的共享右侧图例，直接读取实验工作簿。"""
from pathlib import Path
import sys
import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib.ticker import FuncFormatter
from openpyxl import load_workbook
ROOT=Path(__file__).resolve().parent
sys.path.insert(0,str(ROOT/'efficiency'))
sys.path.insert(0,str(ROOT/'gas'))
from plot_pbkdf2_runtime import load_measurements
from plot_pbkdf2_gas import read_block
sys.path.insert(0,str(Path.home()/'.agents/skills/nature-figure/scripts'))
from audit_panel_alignment import require_matplotlib_panel_alignment


def main():
    mpl.rcParams.update({'font.family':'sans-serif','font.sans-serif':['Arial','Helvetica','DejaVu Sans'],
                        'font.size':6.5,'svg.fonttype':'none','pdf.fonttype':42,
                        'axes.spines.top':False,'axes.spines.right':False,'legend.frameon':False,
                        'axes.linewidth':0.75})
    iterations,times=load_measurements()
    wb=load_workbook(ROOT/'experiments_result.xlsx',data_only=True,read_only=True)
    s=wb['PBKDF2 手续费用实验']
    base=read_block(s,1,2,10,['单笔交易调用次数','Solidity 累计 Gas','EvoCrypt 累计 Gas'])
    blocks=[read_block(s,17,18,26,['调用次数n','g₀=1500','g₀=3000（基线）','g₀=6000']),
            read_block(s,30,31,39,['调用次数n','α=1/4','α=1/2（基线）','α=3/4']),
            read_block(s,42,43,51,['调用次数n','N₀=1','N₀=2（基线）','N₀=4'])]
    for b in blocks:
        assert b[0]==base[0] and b[1][1]==base[1][1]
    wb.close()
    styles=[('Solidity','#566477','s','-'),('Precompiled Contract','#C56528','^','-'),
            ('EvoCrypt','#087F8C','o','-'),('g₀ = 6000','#B68A28','^','--'),
            ('α = 3/4','#8064A2','D','-.'),('N₀ = 4','#60933F','v',':')]
    fig,axes=plt.subplots(1,2,figsize=(4.8,2.15))
    handles={}
    def draw(ax,x,ys,indices):
        for y,i in zip(ys,indices):
            label,color,marker,ls=styles[i]
            line,=ax.plot(x,y,label=label,color=color,marker=marker,linestyle=ls,
                          markersize=3,linewidth=1.1,zorder=3)
            handles[i]=line
        ax.grid(axis='y',color='#E4E8EC',linewidth=.5,zorder=0)
        ax.tick_params(length=2,pad=2)
    draw(axes[0],iterations,times,[0,1,2])
    assert min(iterations)>0
    axes[0].set_xscale('log');axes[0].set_xlim(.85,120);axes[0].set_ylim(0,9.1)
    axes[0].set_xticks(iterations,labels=[str(x) for x in iterations]);axes[0].minorticks_off()
    axes[0].set_xlabel('PBKDF2 iterations');axes[0].set_ylabel('Invocation time (ms)')
    gas=[*base[1],*[b[1][2] for b in blocks]]
    assert all(v>0 for series in gas for v in series)
    draw(axes[1],base[0],gas,[0,2,3,4,5])
    axes[1].set_yscale('log');axes[1].set_ylim(25000,25000000);axes[1].set_xlim(.5,16.6)
    axes[1].set_xticks([1,4,8,12,16]);axes[1].set_yticks([30000,100000,300000,1000000,3000000,10000000])
    axes[1].yaxis.set_major_formatter(FuncFormatter(lambda v,_:f'{v/1e6:g}'))
    axes[1].minorticks_off();axes[1].set_xlabel('Calls per transaction');axes[1].set_ylabel('Gas (million)')
    fig.subplots_adjust(left=.09,right=.68,bottom=.23,top=.95,wspace=.42)
    # 统一图例放在两幅图之外；三个参数标签仅对应 Gas 图中的 EvoCrypt 变体。
    fig.legend([handles[i] for i in range(6)],[x[0] for x in styles],
               loc='center left',bbox_to_anchor=(.72,.58),borderaxespad=0,
               handlelength=1.5,handletextpad=.5,labelspacing=.8)
    fig.canvas.draw()
    stem=ROOT/'pbkdf2_invocation_comparison'
    require_matplotlib_panel_alignment(fig,json_out=f'{stem}.alignment.json',strict=True)
    fig.savefig(f'{stem}.pdf')
    fig.savefig(f'{stem}.svg')
    fig.savefig(f'{stem}.png',dpi=300);fig.savefig(f'{stem}.tiff',dpi=600)
    plt.close(fig)

if __name__=='__main__':main()
