nix.build(".#docker.robocadey2");
docker.load();
docker.push("registry.fly.io/xe-robocadey2:latest");
fly.deploy();
