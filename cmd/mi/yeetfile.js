nix.build(".#docker.mi");
docker.load("./result");
docker.push(`ghcr.io/xe/x/mi`);
