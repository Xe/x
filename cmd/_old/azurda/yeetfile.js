nix.build(".#docker.azurda");
docker.load("./result");
docker.push("registry.fly.io/azurda:latest");
fly.deploy();
