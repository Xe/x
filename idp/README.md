# idp

idp: IDentity Provider

This is a poor man's implementation of [indieauth's authorization endpoint](https://indieweb.org/authorization-endpoint). Setup and usage is simple:

- `go build -o idp .`
- Set up a domain in your nginx/caddy/whatever configuration for the `idp(1)` server
- Set up `idp(1)` to listen on a local port: `./idp -port 5484`
- Generate a TOTP key: `./idp -secret-gen 16`
- Make a config file
- Configure idp to start on boot: `idp -config /etc/within.x/idp.conf`
- Add `<link rel="authorization_endpoint" href="https://idp.christine.website/auth">` to your homepage/index.html where `idp.christine.website` is your domain from the second step
- Test [here](https://indielogin.com)

## Config File

```
// Example idp(1) config file

port 9040
otp-secret 4S62BZNFXXSZLCRO
owner https://example.com
domain idp.example.com
```

Put this in `/etc/within.x/idp.conf`
