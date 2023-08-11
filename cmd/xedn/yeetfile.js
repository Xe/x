nix.build(".#docker.xedn")
docker.load("./result")
docker.push("registry.fly.io/xedn:latest")
fly.deploy()