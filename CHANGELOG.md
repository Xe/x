# [1.30.0](https://github.com/Xe/x/compare/v1.29.0...v1.30.0) (2026-02-16)

### Bug Fixes

- **falin:** resolve npm ci dependency conflict ([#835](https://github.com/Xe/x/issues/835)) ([683bff8](https://github.com/Xe/x/commit/683bff8e5a484f7b4f2a4b506380501fda204720))
- **mi:** use blog post summary in Bluesky embed description ([#848](https://github.com/Xe/x/issues/848)) ([7491810](https://github.com/Xe/x/commit/7491810e2481fd6be8f4baec97a08e4fd758b2f8))
- **nguh:** return error for unsupported tokens ([d1a50e7](https://github.com/Xe/x/commit/d1a50e7a1604c1ecb831be9199365f39c22218a2))
- **skills/xe-writing-style:** update details about successive paragraph starting letter rule ([5808b2b](https://github.com/Xe/x/commit/5808b2b2100dbcac2f1ec63800909d9cc6588ef2))
- **skills/xe-writing-style:** wumbofy this with Opus ([cea6609](https://github.com/Xe/x/commit/cea6609b7f020b682136817604b40ebd8bf1ce61))
- **useragent:** use filepath.Base for os.Args[0] in GenUserAgent ([#830](https://github.com/Xe/x/issues/830)) ([3ef21d9](https://github.com/Xe/x/commit/3ef21d9a605ac0f1607a71677d233d11f1b64187))
- **web:** replace deprecated io/ioutil with io ([#829](https://github.com/Xe/x/issues/829)) ([fee5e4f](https://github.com/Xe/x/commit/fee5e4f1ce31e9bd5b6e2331eddebfd7fa015a3e))

### Features

- **cmd/x:** add ai-add-provider and ai-list-models subcommands ([#850](https://github.com/Xe/x/issues/850)) ([bba7f41](https://github.com/Xe/x/commit/bba7f419c26bc3a099f306cd4006f9e5a86dd8e6))
- **python:** accept io/fs.FS as root filesystem parameter ([#813](https://github.com/Xe/x/issues/813)) ([87b97e8](https://github.com/Xe/x/commit/87b97e8fadaecb2efcbd1e87d6990440b61ddd8a))
- **reviewbot:** add Python interpreter with repo filesystem ([#814](https://github.com/Xe/x/issues/814)) ([b40ff1c](https://github.com/Xe/x/commit/b40ff1cadba0c5aeec5936ef5134ec900d80f310))
- **sakurajima:** add HTTP request timeouts to prevent hanging connections ([#837](https://github.com/Xe/x/issues/837)) ([d50a792](https://github.com/Xe/x/commit/d50a7922e614feaa51d8d593f17d7cc03ffbcc1e))
- **sakurajima:** add request size limits to prevent DoS attacks ([#838](https://github.com/Xe/x/issues/838)) ([f207855](https://github.com/Xe/x/commit/f2078551e8f2cf28d7450e396715873f5acb1607))
- **sakurajima:** add request size limits to prevent DoS attacks ([#839](https://github.com/Xe/x/issues/839)) ([80dd84a](https://github.com/Xe/x/commit/80dd84ad36e8acd028812ae9bd032b901b2524a1))
- **sakurajima:** production readiness fixes and enhancements ([#834](https://github.com/Xe/x/issues/834)) ([4368e6f](https://github.com/Xe/x/commit/4368e6fd401f7d883a26e3eadec1eb685e5aa4fd))
- **sapientwindex:** add state to prevent double-posts ([#825](https://github.com/Xe/x/issues/825)) ([6ba9223](https://github.com/Xe/x/commit/6ba922341741c20bac2beb48e4103bc06a3c2036))
- **skills:** add experimental Xe writing style skill ([baed3bd](https://github.com/Xe/x/commit/baed3bd228a40d5be584dbdff4d8b300be85e10e))
- **skills:** add Go table-driven tests skill ([#817](https://github.com/Xe/x/issues/817)) ([a2e35ea](https://github.com/Xe/x/commit/a2e35ead52b7d3b0849129a9b399b0f2f7e4e283))
- **store:** add filesystem backends (DirectFile, JSONMutexDB, CAS) ([#824](https://github.com/Xe/x/issues/824)) ([4f694cf](https://github.com/Xe/x/commit/4f694cfaa9efe8754f68e2871aeb5d27576c5433))
- **totpgen:** add TOTP code generator command ([#833](https://github.com/Xe/x/issues/833)) ([d0a556d](https://github.com/Xe/x/commit/d0a556d6aa204eec74ffe5cb4b8fbe5496452d89))

### BREAKING CHANGES

- **python:** llm/codeinterpreter/python.Run() now takes fs.FS as first parameter

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes

Signed-off-by: Xe Iaso <me@xeiaso.net>

# [1.29.0](https://github.com/Xe/x/compare/v1.28.0...v1.29.0) (2026-01-13)

### Features

- add cmd/readability ([8043d20](https://github.com/Xe/x/commit/8043d209826d0e038ebd991b32806de9424a4cae))

# [1.28.0](https://github.com/Xe/x/compare/v1.27.0...v1.28.0) (2026-01-10)

### Bug Fixes

- **reviewbot:** add desired output format to the system prompt ([7cb3f98](https://github.com/Xe/x/commit/7cb3f98b2e68f975a4902902c0f5c2987bcbecc1))

### Features

- add reviewbot prototype ([1ea29dd](https://github.com/Xe/x/commit/1ea29dde9e7f984e745d07b4343b6b8366c4b9ae))
- **reviewbot:** add auto-trigger via commit footer ([91726b2](https://github.com/Xe/x/commit/91726b2e5827b2dfe8ae11cedee59bf4a4e96d0b))
- **reviewbot:** if no tool call, default to leaving a PR comment ([#809](https://github.com/Xe/x/issues/809)) ([c530dd7](https://github.com/Xe/x/commit/c530dd777dde5fdc4efdd79a4cf0ac509d9128ab))
- **skills:** add templ-htmx skill using local htmx package ([ad16a19](https://github.com/Xe/x/commit/ad16a19c9810cfb0ae52d122158738cea7eec154))

# [1.27.0](https://github.com/Xe/x/compare/v1.26.1...v1.27.0) (2026-01-08)

### Bug Fixes

- **package.json:** use Signed-off-by instead of Signed-Off-By ([c2406e3](https://github.com/Xe/x/commit/c2406e3de87ab486e37b68e33b044288a704e5bd))
- **yeetfile:** build sysexts ([c9b3ab7](https://github.com/Xe/x/commit/c9b3ab7f70954de56174ed1b634c0d224a8bdbee))

### Features

- add 7-day cooldown to dependabot updates ([#791](https://github.com/Xe/x/issues/791)) ([1a5d9d3](https://github.com/Xe/x/commit/1a5d9d3fa49558fd483c1d2b29c40f842f1f3186))
- add attention attenuator package ([6ca9c01](https://github.com/Xe/x/commit/6ca9c014acb21fa808f20222f9383be8a90c44cb))
- add claude-code-cfg package ([73798a3](https://github.com/Xe/x/commit/73798a312ee21cda529f527370d7c1ca60a8bc64))
- **kube/alrest:** add mcp-auth sidecar ([e56238b](https://github.com/Xe/x/commit/e56238bb3bcc0870b84e4bfbc10e04c99715ccdc))
- **mi:** add birthday field to Member model ([#784](https://github.com/Xe/x/issues/784)) ([69bff8e](https://github.com/Xe/x/commit/69bff8e99e082cfc3e859cfc03bfc597cc36c16a))
- require Signed-off-by in commit messages ([#785](https://github.com/Xe/x/issues/785)) ([7e1c910](https://github.com/Xe/x/commit/7e1c910b5d458f5b7a9e21ebe7272ddb1aae68d1))

## [1.26.1](https://github.com/Xe/x/compare/v1.26.0...v1.26.1) (2025-11-11)

### Bug Fixes

- **mcp:** fix list-system-members output schema ([2a24290](https://github.com/Xe/x/commit/2a2429068fcdbd7dd5f22e0a47caa7037fdec4fb))

# [1.26.0](https://github.com/Xe/x/compare/v1.25.0...v1.26.0) (2025-11-09)

### Features

- add list-system-members MCP tool ([#781](https://github.com/Xe/x/issues/781)) ([3d14b73](https://github.com/Xe/x/commit/3d14b73136855e4444549f6cce866547625fd62e))

# [1.25.0](https://github.com/Xe/x/compare/v1.24.0...v1.25.0) (2025-10-30)

### Bug Fixes

- a bunch of things ([113cde3](https://github.com/Xe/x/commit/113cde35bba091dada6c83f7e40c74dcd0d181a0))
- bump versions of things in kube ([94ef124](https://github.com/Xe/x/commit/94ef124e9eae053d1b9874208eed99854a613196))
- **httpdebug:** log request info ([b7c3bcb](https://github.com/Xe/x/commit/b7c3bcb18d682e30935b6c803b480b2b8fb9c7e6))
- oops ([1796967](https://github.com/Xe/x/commit/17969677917010bdfe3d7aa99cfd8ead189a00fc))
- **sakurajima:** fix tests ([5549686](https://github.com/Xe/x/commit/55496863add61e24c7e6edc1d05288babd52cfc9))

### Features

- add httpdebug Docker target and Dockerfile ([b428337](https://github.com/Xe/x/commit/b428337f8e177ba51110c9895d141542c66c62b2))
- add shiroko k8s ([0172f70](https://github.com/Xe/x/commit/0172f7082b11b4b177ba634461d590296d2a5d53))
- add W'zamqo as valid member name and update suggestions ([d0fff60](https://github.com/Xe/x/commit/d0fff604962700ba5ed185c4828f81db3443b07d))
- **cmd/httpdebug:** enhance security and functionality ([0c0f8ea](https://github.com/Xe/x/commit/0c0f8ea72feab23cbe5ed667e0dd283b1fa77914))
- **cmd/mi:** add MCP server for completely bad ideas ([673dbbd](https://github.com/Xe/x/commit/673dbbde5dd0f61ad7d06239186d49aef732227c))
- cmd/sakurajima ([86de439](https://github.com/Xe/x/commit/86de439c6da51f1751a2c72b79a584ceba72d3ef))
- **cmd/sakurajima:** add autocert config settings ([0017dc0](https://github.com/Xe/x/commit/0017dc0eeab6e42dbc1735ea6a18bf509d36a28f))
- **cmd/x:** fix switch command ([616d8b4](https://github.com/Xe/x/commit/616d8b452be147e9994f292f446ddf49d7f6773d))
- **cmd/x:** port list-switches command over ([edd2e48](https://github.com/Xe/x/commit/edd2e48be7437fe43ea053d11372840199b0caab))
- **cmd:** add x-browser-validation test program ([8c0154e](https://github.com/Xe/x/commit/8c0154e9fd798cdeec0e3f6ad66f287bdae9e7b7))
- **sakurajima:** add the rigging for log filtering logic ([0274cf3](https://github.com/Xe/x/commit/0274cf3e4cdfbe908fb8602d980b5b1636f5e6fb))
- **sakurajima:** implement access logs / logrotate logic ([8b68476](https://github.com/Xe/x/commit/8b68476147a011a8deebf6aa0b8f82770397ae67))
- **sakurajima:** log filters ([b4a5f77](https://github.com/Xe/x/commit/b4a5f77dc13da12437865bf8b5dbed60e0611685))
- **stickers:** use branded presigned URLs ([5223d77](https://github.com/Xe/x/commit/5223d77df6b10ff75d6a0dce4b674a989562b833))

# [1.24.0](https://github.com/Xe/x/compare/v1.23.0...v1.24.0) (2025-07-15)

### Features

- **cmd/httpdebug:** add HTTP method and URI to output ([158fb5b](https://github.com/Xe/x/commit/158fb5b9c4883aed25bf38d1893cf1a84d0db76c))
- **cmd/relayd:** ja4t / ja4h fingerprinting ([e810bdb](https://github.com/Xe/x/commit/e810bdb0c432df273e7cc6f0885273daae5faea6))
- **fish-config:** build a deb too I guess ([063d5c0](https://github.com/Xe/x/commit/063d5c0dd6cefbf24aab2bcebd02003f65994c5d))
- **fish-config:** dotenv.fish ([7c8b929](https://github.com/Xe/x/commit/7c8b92988925a9db748a653ef62fde5fb16f36d3))
- **pb/relayd:** add fingerprints object ([2fc470c](https://github.com/Xe/x/commit/2fc470cb7ed3c887f5feaf3d10a88265dda1d685))

# [1.23.0](https://github.com/Xe/x/compare/v1.22.0...v1.23.0) (2025-06-28)

### Bug Fixes

- **fish-config:** make fish prompt compatible with vs code ([484b614](https://github.com/Xe/x/commit/484b614b607f0a5b610dd6186a07aaa547873ea1))
- **fish-prompt:** clean up theme ([d00dabd](https://github.com/Xe/x/commit/d00dabd9e9962a0950837a9a0eb2d419471ad00c))
- **mi:** enable reflection ([b264671](https://github.com/Xe/x/commit/b264671471cc62a4696442de204085808e628130))

### Features

- **pkgs/fish-config:** add autols, autopair, fisher, and nvm ([2a8ccb5](https://github.com/Xe/x/commit/2a8ccb5e9235262aa0db262caa46061b174472c9))

# [1.22.0](https://github.com/Xe/x/compare/v1.21.0...v1.22.0) (2025-06-11)

### Bug Fixes

- **cmd/relayd:** set the right Host header ([6db9999](https://github.com/Xe/x/commit/6db99999b5de18ae9da2cb995473141b96a46fae))
- **mi:** make old routes work to avoid breaking tools ([b51ae17](https://github.com/Xe/x/commit/b51ae174ba2eb47a4d2d06db706dbee6969a5444))
- **yeetfile:** only build uploud where purego is supported ([8c12cfa](https://github.com/Xe/x/commit/8c12cfa67be87d5f1cfd8a5315160a5197f22ac1))

### Features

- **ingressd:** grpc health checking ([0870e5d](https://github.com/Xe/x/commit/0870e5df3513789b16fc33a588f01370dad55a76))

# [1.21.0](https://github.com/Xe/x/compare/v1.20.0...v1.21.0) (2025-05-31)

### Bug Fixes

- **pvfm/bot:** cache seen messages for up to 10 minutes ([76268a9](https://github.com/Xe/x/commit/76268a97a921358579b8c96feefd4a36f01f6478))
- **recording:** don't delete temporary files ([472754d](https://github.com/Xe/x/commit/472754d2f38c51e6dd77b29c4faf645fb2311caf))

### Features

- **aura:** build with docker buildx bake ([126a3cf](https://github.com/Xe/x/commit/126a3cf9934d4e07647ec08cd6d96ab27f1f9eca))
- **x:** grpchc can do TLS now ([cc4be82](https://github.com/Xe/x/commit/cc4be8203f35b67d58b9d1edf0c2c4ab363e0a96))

# [1.20.0](https://github.com/Xe/x/compare/v1.19.0...v1.20.0) (2025-05-30)

### Features

- **yeetfile:** build riscv64 binaries ([aa0c069](https://github.com/Xe/x/commit/aa0c069cd638b4d3670385f3b9b0934eb186b720))

# [1.19.0](https://github.com/Xe/x/compare/v1.18.0...v1.19.0) (2025-05-29)

### Bug Fixes

- **cmd/relayd:** use X-Http-Protocol for HTTP protocol version ([e73bf5f](https://github.com/Xe/x/commit/e73bf5ffe9c36d7ffb2d8021795503edf0a3a156))
- **yeetfile:** new docker image ([5172ad9](https://github.com/Xe/x/commit/5172ad944aebe6ff44962848333d4912c744d1aa))

### Features

- **cmd/license:** move licenses to go templates ([c01ca48](https://github.com/Xe/x/commit/c01ca4894a0b04ac1cc12d24eda85670f4eaa625))
- **fish-config:** kubernetes info in prompt ([66924c2](https://github.com/Xe/x/commit/66924c2e03d4c0f8860c73875caa24561097aefe))
- **pkgs:** add fish config as a package ([93fd623](https://github.com/Xe/x/commit/93fd6232eb3b5ea22ed4666ec4e2c18b4765464a))
- **x:** grpc health checking ([a78e6cf](https://github.com/Xe/x/commit/a78e6cf59287d7bdf16d21272031d58bfdd38583))
- **yeetfile:** ppc64le binaries ([2fbd92f](https://github.com/Xe/x/commit/2fbd92f092075a26bfd5496742a51a421d72cf6e))

# [1.18.0](https://github.com/Xe/x/compare/v1.17.0...v1.18.0) (2025-05-05)

### Bug Fixes

- **gitea:** fix deployment ([b418d50](https://github.com/Xe/x/commit/b418d50b5d444916e71b06f295c79a10731e6211))
- **relayd:** increase correlation potential ([e047ce3](https://github.com/Xe/x/commit/e047ce314d7ea8b6ec398c1cb633e60ea61f06dc))
- **relayd:** make max bundle size 512 ([915e301](https://github.com/Xe/x/commit/915e3019e03ecb58acc833a503a3708c10b4456d))
- **relayd:** move request ID creation to a variable ([d030f4f](https://github.com/Xe/x/commit/d030f4fcb286791bb1b8be0bdf0d9f6193311b56))

### Features

- **relayd:** change log persistence method ([b3bb404](https://github.com/Xe/x/commit/b3bb404331ddf11b94b1d46a8567308903f94e4c))
- **web:** add iptoasn client ([c53fd2a](https://github.com/Xe/x/commit/c53fd2aad2e65fa362cab2a784488e83d6a9bfb3))

# [1.17.0](https://github.com/Xe/x/compare/v1.16.0...v1.17.0) (2025-04-27)

### Features

- **httpdebug:** quiet mode and function as a systemd service ([1d9fa34](https://github.com/Xe/x/commit/1d9fa34fa84cc125c68ab486d8bbc2dbe7a51f0e))
- **relayd:** autocert support for automatic TLS cert minting ([c9136cc](https://github.com/Xe/x/commit/c9136cc167ca0bbabce1196f88cfc1b302350f0a))
- **xess:** add fancy 404 page ([10e176a](https://github.com/Xe/x/commit/10e176a023ee1a4955160c86f0dc71a435bdf866))

# [1.16.0](https://github.com/Xe/x/compare/v1.15.0...v1.16.0) (2025-04-27)

### Bug Fixes

- **gitea:** use >- instead of > ([972cc99](https://github.com/Xe/x/commit/972cc990716c8593fc1f1d7061e6b707c6bccc51))

### Features

- **relayd:** store and query TLS fingerprints ([ef94cbc](https://github.com/Xe/x/commit/ef94cbcc7f9f90ef5c238413ee3305c305743a42))

# [1.15.0](https://github.com/Xe/x/compare/v1.14.1...v1.15.0) (2025-04-27)

### Features

- **ci:** allow automatically cutting a new release via messages ([b12801a](https://github.com/Xe/x/commit/b12801a2445bbaa8840acd00d76653100a4f6bbe))

## [1.14.1](https://github.com/Xe/x/compare/v1.14.0...v1.14.1) (2025-04-27)

# [1.14.0](https://github.com/Xe/x/compare/v1.13.6...v1.14.0) (2025-04-27)

### Bug Fixes

- **relayd:** disable TCP fingerprinting on Linux for now ([6aa26b7](https://github.com/Xe/x/commit/6aa26b7defa02515fcc8473b8c8603e5fbe45f3f))
- **relayd:** rename HTTP headers for fingerprints ([b64f843](https://github.com/Xe/x/commit/b64f8430190d0a49f8ec6a105e2978714342dd3e))

### Features

- **anubis:** replace with tombstone ([929e2de](https://github.com/Xe/x/commit/929e2debb8b9a63c44e3bb02387a6774821ccb99))
- cmd/aws-secgen for generating fake AWS secrets ([7b8662a](https://github.com/Xe/x/commit/7b8662a0a877fd708afc679b4898e0a54343fe7a))
- **relayd:** add standard reverse proxy headers ([33ebd25](https://github.com/Xe/x/commit/33ebd254071288ae5925b39cc59c3aba67cce499))
- **relayd:** ja4t fingerprinting ([8ecbe6f](https://github.com/Xe/x/commit/8ecbe6f42e0eed79e899178570690aab1ce67c3f))
