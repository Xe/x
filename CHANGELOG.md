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
