nix.build(".#docker.within-website");
docker.load("./result");
docker.push(`ghcr.io/xe/x/within-website`);
yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("sh", "-c", "kubectl rollout restart deployments/within-website");

