nix.build(".#docker.mi");
docker.load("./result");
docker.push(`ghcr.io/xe/x/mi`);
yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("sh", "-c", "kubectl rollout restart -n mi deployments/mi");
