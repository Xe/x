nix.build(".#docker.johaus");
docker.load("./result");
docker.push(`ghcr.io/xe/x/johaus`);
yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("sh", "-c", "kubectl rollout restart deployments/johaus");

