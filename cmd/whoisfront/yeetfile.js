yeet.setenv("GOOS", "linux");
yeet.setenv("GOARCH", "amd64");

go.build();
slug.build("whoisfront");
log.info(nix.hashURL(slug.push("whoisfront")));