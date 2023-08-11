yeet.setenv("GOOS", "linux");
yeet.setenv("GOARCH", "amd64");

go.build();
slug.build("within.website", {
    "config.ts": "config.ts"
});
log.info(slug.push("within.website"));