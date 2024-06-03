nix.build(".#docker.hlang");
docker.load("./result");
docker.push(`ghcr.io/xe/x/hlang`);
yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("sh", "-c", "kubectl rollout restart deployments/hlang");