nix.build(".#docker.tourian");
docker.load("./result");
docker.push("registry.fly.io/tourian:latest");
fly.deploy();
