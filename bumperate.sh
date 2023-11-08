#!/bin/sh

OUT=$(mktemp -d -t nar-hash-XXXXXX)
rm -rf $OUT

go mod vendor -o $OUT
go run tailscale.com/cmd/nardump --sri $OUT >go.mod.sri
rm -rf $OUT

perl -pi -e "s,# nix-direnv cache busting line:.*,# nix-direnv cache busting line: $(cat go.mod.sri)," flake.nix
