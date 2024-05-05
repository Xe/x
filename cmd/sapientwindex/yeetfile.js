nix.build(".#docker.sapientwindex");
docker.load("./result");
docker.push("ghcr.io/xe/x/sapientwindex");
