nix.build(".#docker.mi");
docker.load("./result");
docker.push(`ghcr.io/xe/x/mi`);
yeet.run("kubectl", "apply", "-k=manifest");
yeet.run("sh", "-c", "kubectl rollout restart -n mi deployments/mi");
