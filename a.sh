#!/bin/bash
dir=20170320_ohac_aria_pepe/ohac_aria_pepe

l6="6-0e 6-1f 6-2fs 6-3g 6-4gs 6-5a"
l5="5-1as 5-2b 5-3c 5-4cs 5-5d"
l4="4-0d 4-1ds 4-2e 4-3f 4-4fs 4-5g"
l3="3-0g 3-1gs 3-2a 3-3as 3-4b 3-5c"
l2="2-0b 2-1c 2-2cs 2-3d 2-4ds 2-5e"
l1a="1-0e 1-1f 1-2fs 1-3g 1-4gs 1-5a 1-6as 1-7b 1-8c 1-9cs 1-10d 1-11ds"
l1b="1-12e 1-13f 1-14fs 1-15g 1-16gs 1-17a 1-18as"
#l1="$l1a $l1b"
l1="$l1a"

for name in $l6 $l5 $l4 $l3 $l2 $l1; do
  if ! [ -a $name.s16 ]; then
    sox $dir/pepe_$name.flac $name.s16
  fi
  echo $name
  ./wav2midi -f $name.s16
  echo
done
