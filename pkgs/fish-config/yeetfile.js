const pkgName = rpm.build({
  name: "xe-fish-prompt",
  description: "Xe's fish prompt theme",
  homepage: "https://xeiaso.net",
  license: "CC0",
  goarch: "all",
  depends: ["fish"],

  build: ({ out }) => {
    $`mkdir -p ${out}/etc/fish/completions`;
    $`mkdir -p ${out}/etc/fish/conf.d`;
    $`mkdir -p ${out}/etc/fish/functions`;

    $`cp -vrf ./completions/* ${out}/etc/fish/completions`;
    $`cp -vrf ./conf.d/* ${out}/etc/fish/conf.d`;
    $`cp -vrf ./functions/* ${out}/etc/fish/functions`;
  },
});

gitea.uploadPackage("xe", "x", "current", pkgName);
