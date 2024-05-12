["amd64", "arm64"].forEach(goarch => rpm.build({
    name: "ingressd",
    description: "ingress for my homelab",
    homepage: "https://within.website",
    license: "CC0",
    goarch,

    build: (out) => {
        go.build("-o", `${out}/usr/bin/ingressd`);
        yeet.run("mkdir", "-p", `${out}/usr/lib/systemd/system`);
        yeet.run("cp", "ingressd.service", `${out}/usr/lib/systemd/system/ingressd.service`);
    },
}));