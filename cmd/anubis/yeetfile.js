go.install();

["amd64", "arm64"].forEach(goarch => rpm.build({
    name: "anubis",
    description: "Anubis weighs the souls of incoming HTTP requests and uses a sha256 proof-of-work challenge in order to protect upstream resources from scraper bots.",
    homepage: "https://xeiaso.net/blog/2025/anubis",
    license: "CC0",
    goarch,

    build: (out) => {
        // install Anubis binary
        go.build("-o", `${out}/usr/bin/anubis`);

        // install systemd unit
        yeet.run("mkdir", "-p", `${out}/usr/lib/systemd/system`);
        yeet.run("cp", "anubis@.service", `${out}/usr/lib/systemd/system/anubis@.service`);

        // install default config
        yeet.run("mkdir", "-p", `${out}/etc/anubis`);
        yeet.run("cp", "anubis.env.default", `${out}/etc/anubis/anubis-default.env`);
    },
}));