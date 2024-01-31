nix.build(".#docker.mimi")
docker.load("./result")
docker.push("registry.fly.io/mimi:latest")
fly.deploy()