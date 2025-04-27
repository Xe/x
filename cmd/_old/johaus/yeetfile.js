yeet.run("kubectl", "apply", "-f=manifest.yaml");
yeet.run("sh", "-c", "kubectl rollout restart deployments/johaus");
