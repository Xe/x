log.println(nix.expr`let hi = ${"there"}; in hi`);

const url = "https://xena.greedo.xeserv.us/files/slugs/johaus-061520192052.tar.gz";
const hash = "0cfx2skh7bz9w4p6xbcns14wgf2szkqlrga6dvnxrhlh3i0if519";

log.println(nix.expr`src = builtins.fetchurl {
  url = ${url};
  sha256 = ${hash};
}`)

const greeting = "Hello"
const data = nix.eval`{ greeting = ${greeting}; }`;
log.info(`greeting = ${data.greeting}`);
