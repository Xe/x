# dnsd

A custom DNS server for my network. DNS zone files are dynamically downloaded on
startup and are continuously monitored for changes. When the DNS zone is changed,
the service reloads it.

I primarily use this to give myself a limited form of piHole DNS adblocking, as
well as serving my [home network services](https://home.cetacean.club).

This is related to my [WireGuard Site to Site VPN](https://christine.website/blog/site-to-site-wireguard-part-1-2019-04-02)
project.

## How to Configure `dnsd`

`dnsd` relies on [RFC 1035](https://tools.ietf.org/html/rfc1035) zone files. This
is a file that looks roughly like this:

```rfc1035
$TTL 60
$ORIGIN pele.
@       IN     SOA     oho.pele. some@email.address. (
                       2019040601  ; serial number YYYYMMDDNN
                       28800       ; Refresh
                       7200        ; Retry
                       864000      ; Expire
                       60          ; Minimum DNS TTL
                       )
        IN     NS      oho.pele.
        
oho IN A 10.55.0.1
1.0.55.10.in-addr.arpa. IN PTR oho.pele.

;; apps
prometheus IN CNAME oho.pele.
grafana IN CNAME oho.pele.
```

Put this file in a publicly available place and then set its URL as a
`-zone-file` in the command line configuration. This file will be monitored
every minute for changes (via the proxy of the ETag of the HTTP responses).

If you need to change the DNS forwarding server, set the value of the environment
variable `FORWARD_SERVER` or the command line flag `-forward-server`.

## Installation

### Docker

```console
$ docker run --name dnsd -p 53:53/udp -dit --restart always xena/dnsd:1.0.2-5-g64aea8a \
  dnsd -zone-url https://domain.hostname.tld/path/to/your.zone \
       -zone-url https://domain.hostname.tld/path/to/adblock.zone \
       -forward-server 1.1.1.1:53
```

### From Git with systemd

```console
$ go get -u -v github.com/Xe/x/cmd/dnsd@latest
$ GOBIN=$(pwd) go install github.com/Xe/x/cmd/dnsd
$ sudo cp dnsd /usr/local/bin/dnsd
<edit dnsd.service as needed>
$ sudo cp dnsd.service /etc/systemd/system/dnsd.service
$ sudo systemctl daemon-reload
$ sudo systemctl start dnsd
$ sudo systemctl status dnsd
$ sudo systemctl enable dnsd
```

## Testing

```console
$ dig @127.0.0.1 google.com
$ dig @127.0.0.1 custom.domain
```

## Support

If you need help with this, please [contact](https://christine.website/contact) me.
This is fairly simplistic software. If you need anything more, I'd suggest using
[CoreDNS](https://coredns.io) or similar.

If you like this software, please consider donating on [Patreon](https://www.patreon.com/cadey)
or [Ko-Fi](https://www.ko-fi.com/christinedodrill). I use this software daily on my personal
network to service most of my devices.

Thanks and be well.
