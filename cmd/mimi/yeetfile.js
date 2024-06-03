nix.build(".#docker.mimi");
docker.load("./result");
docker.push(`ghcr.io/xe/x/mimi`);
yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("sh", "-c", "kubectl rollout restart -n mimi deployments/mimi");
