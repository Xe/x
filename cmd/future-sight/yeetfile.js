nix.build(".#docker.future-sight");
docker.load("./result");
docker.push(`ghcr.io/xe/x/future-sight`);
yeet.run("kubectl", "apply", "-k=manifest/prod");
yeet.run("sh", "-c", "kubectl rollout restart -n future-sight deployments/future-sight");
