nix.build(".#docker.sapientwindex");
docker.load("./result");
const image = `ghcr.io/xe/x/sapientwindex:${yeet.datetag}`;
docker.tag("ghcr.io/xe/x/sapientwindex", image);
docker.push(`ghcr.io/xe/x/sapientwindex`);
docker.push(image);
yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("kubectl", "patch", "deployment", "sapientwindex", "--subresource='spec'", "--type='merge'", "-p", JSON.stringify({
    spec: {
        templates: {
            spec: {
                containers: [{
                    name: "bot",
                    image
                }]
            }
        }
    }
}));
