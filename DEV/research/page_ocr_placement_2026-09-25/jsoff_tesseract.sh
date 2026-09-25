#!/bin/sh
# One short line box (20 px tall) holding a sentence far longer than the box can show.
printf 'level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n'
printf '1\t1\t0\t0\t0\t0\t0\t0\t400\t200\t-1\t\n'
printf '4\t1\t1\t1\t1\t0\t20\t20\t200\t20\t-1\t\n'
x=20
for w in This translated sentence is much longer than the tiny source region it replaces and cannot fit; do
  printf '5\t1\t1\t1\t1\t1\t%d\t20\t10\t20\t95\t%s\n' $x "$w"; x=$((x+11))
done
