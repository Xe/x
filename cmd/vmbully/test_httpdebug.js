$`wget -O /tmp/httpdebug.deb https://github.com/Xe/x/releases/download/v1.31.0/httpdebug_1.31.0_amd64.deb`;
$`apt-get update`;
$`DEBIAN_FRONTEND=noninteractive apt-get install -y /tmp/httpdebug.deb`;
$`systemctl enable --now httpdebug.service`;
console.log($`systemctl --no-pager status httpdebug.service`);
