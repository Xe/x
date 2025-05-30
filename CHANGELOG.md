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
