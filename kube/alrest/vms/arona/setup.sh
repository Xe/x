#!/usr/bin/env bash

set -exuo pipefail

## weechat

# dependencies
apt update
apt install -y libpod-parser-perl ca-certificates dirmngr gpg-agent apt-transport-https

# GPG key
mkdir /root/.gnupg
chmod 700 /root/.gnupg
mkdir -p /usr/share/keyrings
gpg --no-default-keyring --keyring /usr/share/keyrings/weechat-archive-keyring.gpg --keyserver hkps://keys.openpgp.org --recv-keys 11E9DE8848F2B65222AA75B8D1820DB22A11534E

# APT
echo "deb [signed-by=/usr/share/keyrings/weechat-archive-keyring.gpg] https://weechat.org/ubuntu noble main" | sudo tee /etc/apt/sources.list.d/weechat.list
echo "deb-src [signed-by=/usr/share/keyrings/weechat-archive-keyring.gpg] https://weechat.org/ubuntu noble main" | sudo tee -a /etc/apt/sources.list.d/weechat.list

# install weechat
apt update
apt install -y weechat-curses weechat-plugins weechat-python weechat-perl