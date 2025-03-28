#!/bin/sh
set -eu

cd /root
mkdir -p /home/steam/.steam/steam/steamapps/compatdata/1829350
cp -r /home/steam/.steam/steam/compatibilitytools.d/${PROTON_VERSION}/files/share/default_pfx /home/steam/.steam/steam/steamapps/compatdata/1829350
export STEAM_COMPAT_DATA_PATH=${STEAM_PATH}/steamapps/compatdata/1829350

export DISPLAY=:1

Xvfb $DISPLAY -screen 0 1024x768x16 &

sleep 1

fluxbox &

sleep 1

x11vnc -display $DISPLAY -bg -forever -nopw -quiet -listen 0.0.0.0 -xkb &

sleep 1

(cd /root/motortown && while true; do xterm -e "$@ | tee server.log"; sleep 2; done) &

wait

exec xvfb-run -a wine "$@"