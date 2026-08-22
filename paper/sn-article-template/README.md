## tex转换PDF

`.latexmkrc` 已把 `BSTINPUTS` 指到 `./bst`，在本目录直接编译即可：

```shell
latexmk -pdf -interaction=nonstopmode sn-article.tex
```