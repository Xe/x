export interface Repo {
  kind: "gitea" | "github";
  domain: string;
  user: string;
  repo: string;
  description: string;
}

const githubRepo = (name: string, description: string): Repo => {
  return {
    kind: "github",
    domain: "github.com",
    user: "Xe",
    repo: name,
    description,
  };
};

const giteaRepo = (name: string, description: string): Repo => {
  return {
    kind: "gitea",
    domain: "tulpa.dev",
    user: "cadey",
    repo: name,
    description,
  };
};

const repos: Repo[] = [
  githubRepo("derpigo", "A Derpibooru/Furbooru API client in Go. This is used to monitor Derpibooru/Furbooru for images by artists I care about and archive them."),
  githubRepo("eclier", "A command router for Go programs that implements every command in Lua. This was an experiment for making extensible command-line applications with Lua for extending them."),
  giteaRepo("gopher", "A Gopher (RFC 1436) client/server stack for Go applications. This allows users to write custom Gopher clients and servers."),
  githubRepo("ln", "The natural log function for Go: an easy package for structured logging. This is the logging stack that I use for most of my personal projects."),
  githubRepo("x", "Various experimental things. /x/ is my monorepo of side projects, hobby programming, and other explorations of how programming in Go can be."),
];

export default repos;
