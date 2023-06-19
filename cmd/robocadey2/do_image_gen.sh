#!/usr/bin/env nix-shell
#! nix-shell -p conda -i conda-shell

set -euo pipefail

cd $HOME/code/stable-diffusion
source ~/.bashrc
conda activate ldm

python optimizedSD/robocadey.py --H 512 --W 512 --turbo --seed $SEED --from-file $OUTDIR/prompt.txt --outdir $OUTDIR
