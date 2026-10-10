$`wget -O /home/ci/httpdebug.deb https://github.com/Xe/x/releases/download/v1.31.0/httpdebug_1.31.0_amd64.deb`;
$`sudo apt-get -f install /home/ci/httpdebug.deb`;
$`sudo systemctl enable --now httpdebug.service`;
console.log($`sudo systemctl status httpdebug.service`);
