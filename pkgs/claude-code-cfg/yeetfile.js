const fname = confext.build({
  name: "claude-code-cfg",
  description: "Xe's global CLAUDE.md",
  homepage: "https://xeiaso.net",
  license: "CC0",

  build: (out) => {
    $`mkdir -p ${out}/etc/claude-code`;
    $`cp -vrf ./CLAUDE.md ${out}/etc/claude-code`;
  },
});
