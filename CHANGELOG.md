# Changelog

## [0.26.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.25.0...v0.26.0) (2026-02-16)


### Features

* **frontend:** implement verification flow ([78d9809](https://github.com/canonical/identity-platform-login-ui/commit/78d98092d43f4c4ddc9db6abe773409cc3bbe491))
* implement registration ([2369f15](https://github.com/canonical/identity-platform-login-ui/commit/2369f159de9948f427fa799965bbc43c0ef62c01))
* implement verification flow handlers and service methods ([ea3c01c](https://github.com/canonical/identity-platform-login-ui/commit/ea3c01cfb36e38fff53b39c74c0bcb2c98c799c2))
* initiate login in a new context ([383f085](https://github.com/canonical/identity-platform-login-ui/commit/383f085692b39949be97f5961b036dde26cde5cf))
* temporary UI work as a guide for web team implementation ([f37ef3e](https://github.com/canonical/identity-platform-login-ui/commit/f37ef3e7f0e5e5b9d194f9d25c23c021c678435e))
* update service interface ([19a46dc](https://github.com/canonical/identity-platform-login-ui/commit/19a46dc673684f846ff500cc474dc9df532410a6))


### Bug Fixes

* propagate aal2 to determine if cookies must be deleted ([c01a716](https://github.com/canonical/identity-platform-login-ui/commit/c01a716e59fa87313da7a90d5beef08566ed8609))
* redirect to user details page when session is valid ([be3ca15](https://github.com/canonical/identity-platform-login-ui/commit/be3ca15be817301093ff32ec4f95f8df68f77771))
* redirect to user details page when session is valid ([#769](https://github.com/canonical/identity-platform-login-ui/issues/769)) ([6f39b2b](https://github.com/canonical/identity-platform-login-ui/commit/6f39b2b6b21e60ed8d30d70dfb5238001413cfd5))
* respect max_age ([f8326f1](https://github.com/canonical/identity-platform-login-ui/commit/f8326f1ec420bfd0ea9dd33b24bbc784f52ccda1))
* return error on GetLoginFlow and delete session cookie in case of csrf violation ([b3cd112](https://github.com/canonical/identity-platform-login-ui/commit/b3cd1127a6532d9a025a1db8252a4ba8131a53da))

## [0.25.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.24.2...v0.25.0) (2026-02-02)


### Features

* add error recording and status codes to tracing spans ([0dd5a90](https://github.com/canonical/identity-platform-login-ui/commit/0dd5a903bb5b4f5fe58f117a946fb14f950a80a3))
* add pop value to AMR ([ce0fa73](https://github.com/canonical/identity-platform-login-ui/commit/ce0fa73d840d36c94d03f757a595242a18a80191)), closes [#841](https://github.com/canonical/identity-platform-login-ui/issues/841)
* Add tracing to all methods accepting context.Context ([6b56e71](https://github.com/canonical/identity-platform-login-ui/commit/6b56e71090cddf156bd5951c8a975112930babd5))
* allow hot module reaload for local frontend development ([283c347](https://github.com/canonical/identity-platform-login-ui/commit/283c3475c19718c538c72159c83ee209120ca910))
* update docker kratos to 25.4.0 ([8376b80](https://github.com/canonical/identity-platform-login-ui/commit/8376b80064d03985346672f3d0206aa0005c7497))
* update docker kratos to 25.4.0 ([#832](https://github.com/canonical/identity-platform-login-ui/issues/832)) ([7a61c9d](https://github.com/canonical/identity-platform-login-ui/commit/7a61c9d95e0d338d7812452fdb2e63a25a8bfa41))
* upgrade kratos sdk to v25 ([#827](https://github.com/canonical/identity-platform-login-ui/issues/827)) ([3522639](https://github.com/canonical/identity-platform-login-ui/commit/3522639c18cdb0b40cd3f6565081a180e04c34b8))
* upgrade sdk to v25 ([85bca8d](https://github.com/canonical/identity-platform-login-ui/commit/85bca8d8ad470812bad7962ddceae0f9b424df0f))


### Bug Fixes

* **cmd:** Fix error wrapping to use %w instead of %s ([7f253c6](https://github.com/canonical/identity-platform-login-ui/commit/7f253c6d05ed51f7b1e70ad7fcdc6f7256fbd2a3))
* correct span status for validation failures and error returns ([aac609e](https://github.com/canonical/identity-platform-login-ui/commit/aac609e0cb49ff8f7662700baf7cf3fa6d5bde0e))
* fix vulnerabilities ([d58ca47](https://github.com/canonical/identity-platform-login-ui/commit/d58ca47c05ccc4a049fc83aeb2132328dfbc360e))
* handle 403 on settings update ([d403375](https://github.com/canonical/identity-platform-login-ui/commit/d4033756409b6f99c939945a4b5bcb3728f7580c))
* mark validation failures as errors in tracing for consistency ([69ad15f](https://github.com/canonical/identity-platform-login-ui/commit/69ad15fc383663d8e3d86b2621a9c825d7282fb3))
* preserve original error return behavior in status methods ([9ccb59a](https://github.com/canonical/identity-platform-login-ui/commit/9ccb59a58b5abb6b9729ea34247d2c8b5dfa8160))

## [0.24.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.24.1...v0.24.2) (2026-01-07)


### Bug Fixes

* find the csrf token ([8c8a8ff](https://github.com/canonical/identity-platform-login-ui/commit/8c8a8ff6b76aa8527c66c77288a2f56e4f0f09ee))
* handle password policy violations on backend ([b775d32](https://github.com/canonical/identity-platform-login-ui/commit/b775d32e2e7fd72fc9ab22832f31f0b1b253cc18))
* handle password policy violations on backend ([#820](https://github.com/canonical/identity-platform-login-ui/issues/820)) ([70caffe](https://github.com/canonical/identity-platform-login-ui/commit/70caffe0234a349fcc5ab9f4611eeab643c2fef7))

## [0.24.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.24.0...v0.24.1) (2025-12-04)


### Bug Fixes

* specify device_challenge query param ([4fc1e34](https://github.com/canonical/identity-platform-login-ui/commit/4fc1e343d89f43c3b8ea9a06d43bbdca539879f0))

## [0.24.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.23.1...v0.24.0) (2025-11-14)


### Features

* add kratos features flags to app-config response ([a84a596](https://github.com/canonical/identity-platform-login-ui/commit/a84a596258b65fe32277219ebe5054012501fa8c))
* add real validation of env var with validator library ([e403631](https://github.com/canonical/identity-platform-login-ui/commit/e40363141c19e69accde613e14f004f1781da80b))
* adopt new component wrapper "FeatureEnabled" ([1982c65](https://github.com/canonical/identity-platform-login-ui/commit/1982c655d986a1ee22db926edc44ebd5b062cf9a))
* implement AppConfigProvider and hook ([14e408a](https://github.com/canonical/identity-platform-login-ui/commit/14e408a2f639b1f4c4c986104629c609b2be2c48))


### Bug Fixes

* allow webauthn users to use backup codes ([4464795](https://github.com/canonical/identity-platform-login-ui/commit/4464795202b0b5d6e067546b31d2ccafce16fe20))

## [0.23.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.23.0...v0.23.1) (2025-10-17)


### Bug Fixes

* add security logger ([e3953ec](https://github.com/canonical/identity-platform-login-ui/commit/e3953ec36bf9c62dfdcebae504c1f67cf245496c))
* deal with settings flow while setting up the passkeys, accept form data and redirect like kratos does ([70b4599](https://github.com/canonical/identity-platform-login-ui/commit/70b45997bca992c9aed1183ce6bd372edbb0611c))
* **deps:** update module github.com/openfga/go-sdk to v0.7.2 ([cdc98a4](https://github.com/canonical/identity-platform-login-ui/commit/cdc98a4f70fa1763ef3346b588b6e179b275105f))
* **deps:** update module github.com/openfga/go-sdk to v0.7.2 ([0269b7a](https://github.com/canonical/identity-platform-login-ui/commit/0269b7a046f45fbf9e5b8f06208d9a322d982fc6))
* **deps:** update module github.com/openfga/go-sdk to v0.7.3 ([e829a77](https://github.com/canonical/identity-platform-login-ui/commit/e829a77ed7912b0855febab1cd8cd86a4c50d0b7))
* **deps:** update module github.com/openfga/go-sdk to v0.7.3 ([6afb7d8](https://github.com/canonical/identity-platform-login-ui/commit/6afb7d8d565b6b958c1b05aab0c0a941ca1049d1))
* dont clear cookies if we are not in an aal2 state or if totp or webauthn are disabled ([443e2d2](https://github.com/canonical/identity-platform-login-ui/commit/443e2d2ba32f181ff5a9e16ebbe0a26f9dca788f))
* handle requests that expect html response ([8614cf2](https://github.com/canonical/identity-platform-login-ui/commit/8614cf25242ada0268ec5705e7ae52964113edaa))
* if request expects a non json response, then redirect to ui ([da2bdc6](https://github.com/canonical/identity-platform-login-ui/commit/da2bdc68164d14c702caa95d538d94515b715b8f))
* improve startup/shutdown logic ([bd0aa21](https://github.com/canonical/identity-platform-login-ui/commit/bd0aa21dfe46bc1c35af273b10ed5bd8a11bcd0c))
* switch to basic loading of webauthn script ([a1f6732](https://github.com/canonical/identity-platform-login-ui/commit/a1f67328e2692c6dacd7707466b27a53ec98662a))
* switch to io.ReadAll ([6a65868](https://github.com/canonical/identity-platform-login-ui/commit/6a65868d70394464b1a9b75460839c15b0f6eb6c))
* ui can only talk to login svc backend ([fd301e0](https://github.com/canonical/identity-platform-login-ui/commit/fd301e00e62504e64e79d04ba1e38cba2a2ceb1d))

## [0.23.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.22.0...v0.23.0) (2025-10-02)


### Features

* add social accounts linking and unlinking flows ([aeb30d2](https://github.com/canonical/identity-platform-login-ui/commit/aeb30d2e3027199e099406998fcb120b8c5bacd6))
* add social accounts linking and unlinking flows ([8a75fda](https://github.com/canonical/identity-platform-login-ui/commit/8a75fda18f07756ae31fe4992fdc79708ca96034))
* handle duplicate identifier error in settings flow ([6299c7c](https://github.com/canonical/identity-platform-login-ui/commit/6299c7c23eddcfb1ae2af41292881d42002bd0bf))
* handle duplicate identifier error in settings flow ([2a4dacc](https://github.com/canonical/identity-platform-login-ui/commit/2a4dacc70dd4ea2038c244f973c18562839d8d54))
* show linking success notification after redirection ([bc73db5](https://github.com/canonical/identity-platform-login-ui/commit/bc73db5c3241738078fee426689768d163b489c9))


### Bug Fixes

* **deps:** update dependency @canonical/react-components to v3 ([227fc44](https://github.com/canonical/identity-platform-login-ui/commit/227fc4444d44b33cf216ee8c14a540396877015a))
* **deps:** update dependency @canonical/react-components to v3 ([5969cce](https://github.com/canonical/identity-platform-login-ui/commit/5969cce16935cc5b4dd4a8939e3745ab174cfcd1))
* **deps:** update dependency @canonical/react-components to v3.0.1 ([d7bfeb5](https://github.com/canonical/identity-platform-login-ui/commit/d7bfeb509f8a42853a39aa5943631acf863a1aa5))
* **deps:** update dependency @canonical/react-components to v3.0.1 ([014d309](https://github.com/canonical/identity-platform-login-ui/commit/014d3092f085d2d24047faa61731b14296ef287b))
* **deps:** update dependency @canonical/react-components to v3.1.0 ([60d4c0d](https://github.com/canonical/identity-platform-login-ui/commit/60d4c0de0c676bb9e65b9735c3cf4f59cedc8684))
* **deps:** update dependency @canonical/react-components to v3.1.0 ([8251d86](https://github.com/canonical/identity-platform-login-ui/commit/8251d8631ae52fefcb8fa8c5dd0d2d96e22c57ef))
* **deps:** update dependency @canonical/react-components to v3.1.1 ([47444c8](https://github.com/canonical/identity-platform-login-ui/commit/47444c8cb6d7aacdcda1244a07825a16f2c2a292))
* **deps:** update dependency @canonical/react-components to v3.1.1 ([1095da3](https://github.com/canonical/identity-platform-login-ui/commit/1095da3b546f853e5cdf33145738b45e31afce08))
* **deps:** update dependency @canonical/react-components to v3.2.0 ([5455cc7](https://github.com/canonical/identity-platform-login-ui/commit/5455cc70676b5c0c2d5b0c83bcf6a71b79670577))
* **deps:** update dependency @canonical/react-components to v3.2.0 ([d071f9d](https://github.com/canonical/identity-platform-login-ui/commit/d071f9d19eb7c9c0fd0b03d1f664db867737b3dc))
* **deps:** update dependency @ory/client to v1.21.5 ([d106f2b](https://github.com/canonical/identity-platform-login-ui/commit/d106f2b3b47ec7190bab96a783d8d4e7aad87e5c))
* **deps:** update dependency @ory/client to v1.21.5 ([9f13096](https://github.com/canonical/identity-platform-login-ui/commit/9f130961b1b4702fdedb92091d37d53ea04bf63d))
* **deps:** update dependency @ory/client to v1.22.1 ([4c2b4ef](https://github.com/canonical/identity-platform-login-ui/commit/4c2b4ef9b2d4f23fed9abe144d402157aeea9420))
* **deps:** update dependency @ory/client to v1.22.1 ([a63a727](https://github.com/canonical/identity-platform-login-ui/commit/a63a72765cc1652ed114abdbb9faabfefac7f969))
* **deps:** update dependency sass to v1.92.1 ([f52c777](https://github.com/canonical/identity-platform-login-ui/commit/f52c777fe0ec85b12e2362151b86409ea9c9455f))
* **deps:** update dependency sass to v1.92.1 ([250045e](https://github.com/canonical/identity-platform-login-ui/commit/250045e2547fe20a8d540fac84c9623b8b92b53a))
* **deps:** update dependency sass to v1.93.1 ([d9baef6](https://github.com/canonical/identity-platform-login-ui/commit/d9baef6348fcc5b768b17a8aacbab80e5df058a8))
* **deps:** update dependency sass to v1.93.1 ([7648512](https://github.com/canonical/identity-platform-login-ui/commit/764851256565bc541d5fbe3a4ae0a1043a3ddc0f))
* **deps:** update dependency vanilla-framework to v4.32.1 ([38fa0c2](https://github.com/canonical/identity-platform-login-ui/commit/38fa0c233af08c5c2f626b91c9939ee6b5db742c))
* **deps:** update dependency vanilla-framework to v4.32.1 ([76d52c8](https://github.com/canonical/identity-platform-login-ui/commit/76d52c854ce95ef1b5fc2073934aab395e57cb37))
* **deps:** update dependency vanilla-framework to v4.33.0 ([8a16e0b](https://github.com/canonical/identity-platform-login-ui/commit/8a16e0bf0a46a1eb8722b5f279b0440717da79b8))
* **deps:** update dependency vanilla-framework to v4.33.0 ([d26928b](https://github.com/canonical/identity-platform-login-ui/commit/d26928b9b21ba8c9e691ca0574b1a1b964bbf83d))
* **deps:** update dependency vanilla-framework to v4.34.0 ([f4fda8c](https://github.com/canonical/identity-platform-login-ui/commit/f4fda8c96fd4ee9b358f75fb536a0a41982649ba))
* **deps:** update dependency vanilla-framework to v4.34.0 ([9a82d89](https://github.com/canonical/identity-platform-login-ui/commit/9a82d895c1ae11e206a6850a6fbbe53dbaf512ff))
* **deps:** update dependency vanilla-framework to v4.34.1 ([4a7f7f3](https://github.com/canonical/identity-platform-login-ui/commit/4a7f7f347620843f79f6b5255edfcf7b8769d5d0))
* **deps:** update dependency vanilla-framework to v4.34.1 ([da632bb](https://github.com/canonical/identity-platform-login-ui/commit/da632bbbc25fc00eca6ef9871336f9e576bd7d3a))
* **deps:** update go deps ([0e01ac8](https://github.com/canonical/identity-platform-login-ui/commit/0e01ac8f52a4fc9468d24d9a534b51730361de5e))
* **deps:** update go deps (minor) ([00454f7](https://github.com/canonical/identity-platform-login-ui/commit/00454f765b47817e2d7a4184f27537399993be4e))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.3 ([7d2d58e](https://github.com/canonical/identity-platform-login-ui/commit/7d2d58e94558faabd271416ba8adc819cc18d8a4))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.3 ([ecb8a66](https://github.com/canonical/identity-platform-login-ui/commit/ecb8a66a592e739a1ebac121e850a43788b228bc))
* **deps:** update module github.com/prometheus/client_golang to v1.23.1 ([a871b59](https://github.com/canonical/identity-platform-login-ui/commit/a871b592552d7d40d6fe1a6f4ab03f6acce3375c))
* **deps:** update module github.com/prometheus/client_golang to v1.23.1 ([7706050](https://github.com/canonical/identity-platform-login-ui/commit/77060506ef353e89292f7747c4ee8c718d808b4a))
* **deps:** update module github.com/prometheus/client_golang to v1.23.2 ([a6897a7](https://github.com/canonical/identity-platform-login-ui/commit/a6897a7dc8d97a01ba851d9dea5f01406b27ce4f))
* **deps:** update module github.com/prometheus/client_golang to v1.23.2 ([37ae14d](https://github.com/canonical/identity-platform-login-ui/commit/37ae14d58e9baed9041a922e251d29512b524969))
* **deps:** update module github.com/spf13/cobra to v1.10.1 ([1d1d1f6](https://github.com/canonical/identity-platform-login-ui/commit/1d1d1f631d244f0c7fdff53956782e7386d27610))
* **deps:** update module github.com/spf13/cobra to v1.10.1 ([917631a](https://github.com/canonical/identity-platform-login-ui/commit/917631a2f2596a6884cc900bd3f1edec016355b4))
* **deps:** update module github.com/stretchr/testify to v1.11.0 ([f555465](https://github.com/canonical/identity-platform-login-ui/commit/f555465af6bdb800759afe467b89e3789469f1bb))
* **deps:** update module github.com/stretchr/testify to v1.11.0 ([a391ef5](https://github.com/canonical/identity-platform-login-ui/commit/a391ef5899a61c00cef738a0130463b305ed89c3))
* **deps:** update module github.com/stretchr/testify to v1.11.1 ([e60b36a](https://github.com/canonical/identity-platform-login-ui/commit/e60b36a617fef10af73e947d9076e924ce4ce72a))
* **deps:** update module github.com/stretchr/testify to v1.11.1 ([471c752](https://github.com/canonical/identity-platform-login-ui/commit/471c752930582c29374f2754a2294208adf67a26))
* **deps:** update ui deps ([88b5d1e](https://github.com/canonical/identity-platform-login-ui/commit/88b5d1e8d362b5f63f2adeab3c177d853981d3f1))
* **deps:** update ui deps ([29e8cba](https://github.com/canonical/identity-platform-login-ui/commit/29e8cba3f03b4cf22ddb047133468e5af7fab45d))
* **deps:** update ui deps (minor) ([5eb5bf3](https://github.com/canonical/identity-platform-login-ui/commit/5eb5bf393b9eb2ab95076eeb230ae4818f56e1e4))
* **deps:** update ui deps (minor) ([4443a07](https://github.com/canonical/identity-platform-login-ui/commit/4443a0794002982664383e10a2c071863b66077f))
* **deps:** update ui deps to v15.5.2 ([01026a7](https://github.com/canonical/identity-platform-login-ui/commit/01026a70c02ed36276dc36a3b27b59bb4679d7d5))
* **deps:** update ui deps to v15.5.2 (patch) ([676a1e0](https://github.com/canonical/identity-platform-login-ui/commit/676a1e0385fabd8bd1d7cfdbcf70f155174d11a7))

## [0.22.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.21.3...v0.22.0) (2025-08-22)


### Features

* handle identifier first login pattern ([aa18745](https://github.com/canonical/identity-platform-login-ui/commit/aa18745c60395b4e5fc1876e2c123a6e4d18af1d))
* support oidc settings flow ([cb0407e](https://github.com/canonical/identity-platform-login-ui/commit/cb0407e82b97192f5af4f9fa35035430065d39b0))


### Bug Fixes

* **deps:** update dependency @canonical/react-components to v2.15.0 ([831c63f](https://github.com/canonical/identity-platform-login-ui/commit/831c63f299c03fbd4c8c4b52b2cf723c31cf8995))
* **deps:** update dependency @canonical/react-components to v2.15.0 ([73aa6ff](https://github.com/canonical/identity-platform-login-ui/commit/73aa6ffb45f513c7d0e20d8cbd65d11fe083fb38))
* **deps:** update dependency @canonical/react-components to v2.15.1 ([8b61406](https://github.com/canonical/identity-platform-login-ui/commit/8b614062edbc8ce5045889319081ff6e7cc3056d))
* **deps:** update dependency @canonical/react-components to v2.15.1 ([0df4126](https://github.com/canonical/identity-platform-login-ui/commit/0df412629610c327fb7541544da4fcaf4adaf375))
* **deps:** update dependency @canonical/react-components to v2.16.1 ([383fee1](https://github.com/canonical/identity-platform-login-ui/commit/383fee1c7d3b56c87342cc139614323cfd16ca2b))
* **deps:** update dependency @canonical/react-components to v2.16.1 ([f7b7097](https://github.com/canonical/identity-platform-login-ui/commit/f7b7097c616e2f5556a81af8f4035f4e26dc4fc1))
* **deps:** update dependency @ory/client to v1.21.3 ([106dadd](https://github.com/canonical/identity-platform-login-ui/commit/106daddfe34193c588e7b5ca4c3f4343d467d594))
* **deps:** update dependency @ory/client to v1.21.3 ([e77d7f9](https://github.com/canonical/identity-platform-login-ui/commit/e77d7f92799cb49d4e837d98972aabe01c467bb9))
* **deps:** update dependency @ory/client to v1.21.4 ([3c46c02](https://github.com/canonical/identity-platform-login-ui/commit/3c46c02e0cef8416b4c686abf665de2732c1218a))
* **deps:** update dependency @ory/client to v1.21.4 ([cffc24b](https://github.com/canonical/identity-platform-login-ui/commit/cffc24ba9f6a61e4639e77f046cf4859aff32392))
* **deps:** update dependency vanilla-framework to v4.28.0 ([273e18a](https://github.com/canonical/identity-platform-login-ui/commit/273e18acad8d4871a04cc3da71aa9b71caa6c167))
* **deps:** update dependency vanilla-framework to v4.28.0 ([ed35bbe](https://github.com/canonical/identity-platform-login-ui/commit/ed35bbeda19e42a62083d2c284a3f031a9b50b2d))
* **deps:** update dependency vanilla-framework to v4.30.0 ([317f9c6](https://github.com/canonical/identity-platform-login-ui/commit/317f9c668fed23970421caa145e40f824d29f808))
* **deps:** update dependency vanilla-framework to v4.30.0 ([8b2c3c8](https://github.com/canonical/identity-platform-login-ui/commit/8b2c3c8cacfe1dfb951afc2c75ab3377742d5bd1))
* **deps:** update internal ui dependencies ([4319254](https://github.com/canonical/identity-platform-login-ui/commit/43192545eb5ea3b3405733efee65ea57da8227b6))
* **deps:** update internal ui dependencies ([6060183](https://github.com/canonical/identity-platform-login-ui/commit/6060183d922c34e564aaab540ef79833ff20f24f))
* **deps:** update internal ui dependencies (minor) ([d516b93](https://github.com/canonical/identity-platform-login-ui/commit/d516b9312023c910b460026cd37347410fbf2f3c))
* **deps:** update internal ui dependencies (minor) ([0cc32e6](https://github.com/canonical/identity-platform-login-ui/commit/0cc32e60b604f5624bb633d75644133eb64822ae))
* **deps:** update module github.com/prometheus/client_golang to v1.23.0 ([a5cdfcf](https://github.com/canonical/identity-platform-login-ui/commit/a5cdfcff397359da298b32abca3ad2cd737a7564))
* **deps:** update module github.com/prometheus/client_golang to v1.23.0 ([03f3d2f](https://github.com/canonical/identity-platform-login-ui/commit/03f3d2ff5a3722397799cfa30a55092b90d32c0a))
* **deps:** update module go.uber.org/mock to v0.6.0 ([7606c2b](https://github.com/canonical/identity-platform-login-ui/commit/7606c2b7742274daafe885b6bc3bd24854f60c9c))
* **deps:** update module go.uber.org/mock to v0.6.0 ([93bf2c9](https://github.com/canonical/identity-platform-login-ui/commit/93bf2c9ba38356ee6a39d0526762dc8e23730ed6))
* **deps:** update ui deps ([7ee230a](https://github.com/canonical/identity-platform-login-ui/commit/7ee230a2e492704eeba6758e25740c8424b3ba45))
* **deps:** update ui deps ([1059f45](https://github.com/canonical/identity-platform-login-ui/commit/1059f4544171fa12e6026948f458845e4ab51ac3))
* **deps:** update ui deps ([6f2d8fe](https://github.com/canonical/identity-platform-login-ui/commit/6f2d8fee3e93417ea2b871ca80f42bcdf836e45a))
* **deps:** update ui deps (minor) ([81882d9](https://github.com/canonical/identity-platform-login-ui/commit/81882d9222809ab0e1d5858506bea7167fff849c))
* **deps:** update ui deps (minor) ([c855378](https://github.com/canonical/identity-platform-login-ui/commit/c8553787b580a2b9498d8f45043f0a41e19ebd06))
* **deps:** update ui deps (patch) ([2d53c3a](https://github.com/canonical/identity-platform-login-ui/commit/2d53c3a2c4955a5220d0cbdeb6d6a855cdb6e759))
* **deps:** update ui deps to v15.4.6 ([5a56a05](https://github.com/canonical/identity-platform-login-ui/commit/5a56a05863b8ab715e54b536132a4dabe1778cc3))
* **deps:** update ui deps to v15.4.6 (patch) ([362796b](https://github.com/canonical/identity-platform-login-ui/commit/362796bc15329e430100e83a0599c0dcb187d467))

## [0.21.3](https://github.com/canonical/identity-platform-login-ui/compare/v0.21.2...v0.21.3) (2025-07-29)


### Bug Fixes

* add support email config ([d1d97c3](https://github.com/canonical/identity-platform-login-ui/commit/d1d97c34c0fe52c252b6fd4034ab7a4b4e9ddd2f))
* **deps:** update dependency @canonical/react-components to v2.8.0 ([31acbf3](https://github.com/canonical/identity-platform-login-ui/commit/31acbf3983169e67b24b3f82be64fc058e21e3c5))
* **deps:** update dependency @canonical/react-components to v2.8.0 ([8955471](https://github.com/canonical/identity-platform-login-ui/commit/8955471f8d2b26b99864a981b9edd2b777d416bb))
* **deps:** update dependency @canonical/react-components to v2.9.0 ([0f66678](https://github.com/canonical/identity-platform-login-ui/commit/0f66678f05ce63ba475234abeff272a8152d2974))
* **deps:** update dependency @canonical/react-components to v2.9.0 ([4b53559](https://github.com/canonical/identity-platform-login-ui/commit/4b53559d18b5e807de67d0f97b0f8516bc3f02c2))
* **deps:** update dependency @ory/client to v1.20.23 ([b8dc49c](https://github.com/canonical/identity-platform-login-ui/commit/b8dc49c8c111199aab6ca7252b631577551abb7e))
* **deps:** update dependency @ory/client to v1.20.23 ([10f9aad](https://github.com/canonical/identity-platform-login-ui/commit/10f9aadef7018de92071f2798e92c995468055ea))
* **deps:** update dependency vanilla-framework to v4.26.0 ([ffc100c](https://github.com/canonical/identity-platform-login-ui/commit/ffc100c53464aab3711b974a49b5aa402ee4f23f))
* **deps:** update dependency vanilla-framework to v4.26.0 ([809ca86](https://github.com/canonical/identity-platform-login-ui/commit/809ca867f1a1092a850f07a38212bf60d9fb5075))
* **deps:** update internal ui dependencies ([7fd4d48](https://github.com/canonical/identity-platform-login-ui/commit/7fd4d48f75ec27c15161ca163c490d570a4f3362))
* **deps:** update internal ui dependencies (patch) ([a1c2b2b](https://github.com/canonical/identity-platform-login-ui/commit/a1c2b2bcfbbb336ecb5619cd292cb1213f7961f9))
* **deps:** update ui deps ([b7c1094](https://github.com/canonical/identity-platform-login-ui/commit/b7c1094364caee3f2e04ce89ce746813d5c477ab))
* **deps:** update ui deps (patch) ([65648a3](https://github.com/canonical/identity-platform-login-ui/commit/65648a3be5b4bc22856d4002ef127a8315b10029))
* **deps:** update ui deps to v15.4.1 ([054b99f](https://github.com/canonical/identity-platform-login-ui/commit/054b99f60d91076f7b24cd65843f3c97bf0a21cd))
* **deps:** update ui deps to v15.4.1 (minor) ([778edcc](https://github.com/canonical/identity-platform-login-ui/commit/778edccf955b7796e4e90f3f19868f15d6f4b812))
* **deps:** update ui deps to v15.4.3 ([fcce0f1](https://github.com/canonical/identity-platform-login-ui/commit/fcce0f1c8a10d0d0c964d2f2437c88236c625578))
* **deps:** update ui deps to v15.4.3 (patch) ([88d20e1](https://github.com/canonical/identity-platform-login-ui/commit/88d20e15114e05152fbb3582ba3c69e0dd195331))
* **ui:** make support email configurable ([ecb0686](https://github.com/canonical/identity-platform-login-ui/commit/ecb0686489e71b496e9f7848a3b70d935ab18e0d))

## [0.21.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.21.1...v0.21.2) (2025-07-10)


### Bug Fixes

* clean the request body parsing in login & settings flow ([9d8eb87](https://github.com/canonical/identity-platform-login-ui/commit/9d8eb873fdc7b920d9a0c31aa8ed1b357ddec8ad))
* clean the request body parsing in login & settings flow ([ffe0305](https://github.com/canonical/identity-platform-login-ui/commit/ffe0305200bf46f37449ce1eb9dc30d0ee0563ca))
* **deps:** update dependency @canonical/react-components to v2.6.1 ([459da15](https://github.com/canonical/identity-platform-login-ui/commit/459da15403ae57c1a0abbeea3c10674f1f6baecd))
* **deps:** update dependency @canonical/react-components to v2.6.1 ([b743e02](https://github.com/canonical/identity-platform-login-ui/commit/b743e02dbad7f37f8225f418bd4befcc8191d94c))
* **deps:** update dependency @canonical/react-components to v2.7.0 ([aaa17e5](https://github.com/canonical/identity-platform-login-ui/commit/aaa17e58c13f05eb2a49ff45ef9270ef2a58578c))
* **deps:** update dependency @canonical/react-components to v2.7.0 ([fafbec0](https://github.com/canonical/identity-platform-login-ui/commit/fafbec080f098921b8980ce56c2f4113f092a942))
* **deps:** update dependency @ory/client to v1.20.22 ([4168540](https://github.com/canonical/identity-platform-login-ui/commit/4168540625326f61b83695e9e60d40312152286c))
* **deps:** update dependency @ory/client to v1.20.22 ([068bb26](https://github.com/canonical/identity-platform-login-ui/commit/068bb2647a31317995c2e3a3ae1fd9b889d26083))
* **deps:** update go deps ([29e7661](https://github.com/canonical/identity-platform-login-ui/commit/29e7661863f4a629853c16af44be9dae47b41026))
* **deps:** update go deps (minor) ([5698c1e](https://github.com/canonical/identity-platform-login-ui/commit/5698c1e467ae72f27fecdaf5ba95f7ea8d00bd20))
* **deps:** update go deps to v1.36.0 ([7f804d1](https://github.com/canonical/identity-platform-login-ui/commit/7f804d128f3c2efa7da751698cacccb9f8452601))
* **deps:** update go deps to v1.36.0 (minor) ([957c3d2](https://github.com/canonical/identity-platform-login-ui/commit/957c3d2be459e21825982fe1e4d2406d2b40fbae))
* **deps:** update go deps to v1.37.0 ([7d50251](https://github.com/canonical/identity-platform-login-ui/commit/7d502515a33910d03f4d9e40795a722a67cbf8b7))
* **deps:** update go deps to v1.37.0 (minor) ([bf259c0](https://github.com/canonical/identity-platform-login-ui/commit/bf259c0036ce4159cb67c74558a9be2223d2d3d4))
* **deps:** update internal ui dependencies ([b409d7d](https://github.com/canonical/identity-platform-login-ui/commit/b409d7d4a36653e803e5e6ab9665619012229d3b))
* **deps:** update internal ui dependencies (minor) ([3eb5391](https://github.com/canonical/identity-platform-login-ui/commit/3eb5391def5c1dcf628c764a449716dd2d18e7e4))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.2 [security] ([1ed6503](https://github.com/canonical/identity-platform-login-ui/commit/1ed6503917696e439bc40a8240a660954a5e235a))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.2 [security] ([d8a5ec9](https://github.com/canonical/identity-platform-login-ui/commit/d8a5ec9b0ccf8947638510d4d13f49b2236a933a))
* **deps:** update module github.com/go-chi/cors to v1.2.2 ([94d28a0](https://github.com/canonical/identity-platform-login-ui/commit/94d28a04d2c416b40aba8a4b07253943e764b4c9))
* **deps:** update module github.com/go-chi/cors to v1.2.2 ([69fa4ca](https://github.com/canonical/identity-platform-login-ui/commit/69fa4caae9aabad2a5be83777afdea5d5e0512a8))
* **deps:** update module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp to v0.61.0 ([d6f7ede](https://github.com/canonical/identity-platform-login-ui/commit/d6f7ede055150599430e74fba62d4eb16853b46c))
* **deps:** update module go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp to v0.61.0 ([25ad808](https://github.com/canonical/identity-platform-login-ui/commit/25ad8086526b049c6bbe342bc7773667cbd7867e))
* **deps:** update module go.opentelemetry.io/contrib/propagators/jaeger to v1.36.0 ([b379d9f](https://github.com/canonical/identity-platform-login-ui/commit/b379d9fff893490678f1259e48cfd2b9c54494ff))
* **deps:** update module go.opentelemetry.io/contrib/propagators/jaeger to v1.36.0 ([ed345bb](https://github.com/canonical/identity-platform-login-ui/commit/ed345bb66194350b07e9b840cafdc2230162b313))
* **deps:** update module go.uber.org/mock to v0.5.2 ([882d47e](https://github.com/canonical/identity-platform-login-ui/commit/882d47e68d8cdd3bf33311674cef16af727c8715))
* **deps:** update module go.uber.org/mock to v0.5.2 ([2ae3f5d](https://github.com/canonical/identity-platform-login-ui/commit/2ae3f5db989e78d6b667457eb5d0777aa30a5079))
* **deps:** update ui deps ([8548db3](https://github.com/canonical/identity-platform-login-ui/commit/8548db31c9ad3220d8e347cdd6437ca1834b908d))
* **deps:** update ui deps ([d928fe2](https://github.com/canonical/identity-platform-login-ui/commit/d928fe2b1a5848d06f6d5e75c31b2ccb7a78ec08))
* **deps:** update ui deps ([ac8ac52](https://github.com/canonical/identity-platform-login-ui/commit/ac8ac525ad7fb5c07f72c06937305b4c310528ed))
* **deps:** update ui deps ([2df583a](https://github.com/canonical/identity-platform-login-ui/commit/2df583a74f1f1fea1bba261eeb139a2bb7b174de))
* **deps:** update ui deps ([56487c6](https://github.com/canonical/identity-platform-login-ui/commit/56487c60a92eb28a11890764979104b718ccf2f6))
* **deps:** update ui deps (minor) ([9eddff4](https://github.com/canonical/identity-platform-login-ui/commit/9eddff4eaf21226018f9f4156e0fcdca9b8c1881))
* **deps:** update ui deps (minor) ([c01e06d](https://github.com/canonical/identity-platform-login-ui/commit/c01e06d6a29204ff33d75c1b3847e737c7fa60bb))
* **deps:** update ui deps (patch) ([1017b14](https://github.com/canonical/identity-platform-login-ui/commit/1017b1424c3878d466349e72a82887627ac4fb25))
* **deps:** update ui deps (patch) ([d54d073](https://github.com/canonical/identity-platform-login-ui/commit/d54d07342edd7c955270b95a4baaaddad532356f))
* **deps:** update ui deps (patch) ([c67a9aa](https://github.com/canonical/identity-platform-login-ui/commit/c67a9aabd4c0c907c5cb35a0e38e28e5108ad464))
* remove wildcard domain and unsafe-inline of styles in content security policy ([74ff821](https://github.com/canonical/identity-platform-login-ui/commit/74ff821e79935d340bccbf7b9bb9440258a529f5))

## [0.21.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.21.0...v0.21.1) (2025-04-08)


### Bug Fixes

* add kratos to csp if needed ([1730db1](https://github.com/canonical/identity-platform-login-ui/commit/1730db1ccfd05ce3ddd40c987f83d20e8edb1faf))
* add required value for config ([ad3188a](https://github.com/canonical/identity-platform-login-ui/commit/ad3188a2e5b66806e1782d33a61041f8c48bfa2d))
* allow frontend to set return_to ([81eff57](https://github.com/canonical/identity-platform-login-ui/commit/81eff57ec5cb412bfd8be57d9a943e1678183ed7))
* check required session aal in consent ([0759c40](https://github.com/canonical/identity-platform-login-ui/commit/0759c40da15b1d426ca75c873aedc2856e250df7))
* **deps:** update dependency @canonical/react-components to v1.10.0 ([0d25000](https://github.com/canonical/identity-platform-login-ui/commit/0d25000bbc36072916cf9ad638bbe39c590110b7))
* **deps:** update dependency @canonical/react-components to v1.10.0 ([fa40304](https://github.com/canonical/identity-platform-login-ui/commit/fa403044d434505113ada5fb23995ec7132f3610))
* **deps:** update dependency @canonical/react-components to v1.9.1 ([5d382ea](https://github.com/canonical/identity-platform-login-ui/commit/5d382eae02d4b33afde21e19f44f8a1cdd9fd434))
* **deps:** update dependency @canonical/react-components to v1.9.1 ([ecaa1d9](https://github.com/canonical/identity-platform-login-ui/commit/ecaa1d9d667cdb2a8f746d6aa49ef04e584b18d0))
* **deps:** update dependency @ory/client to v1.16.7 ([3a150ca](https://github.com/canonical/identity-platform-login-ui/commit/3a150ca897b7a20a0e074bdaa4742b5cce57fdf7))
* **deps:** update dependency @ory/client to v1.16.7 ([868acc8](https://github.com/canonical/identity-platform-login-ui/commit/868acc8abefd3c397e4d0303080f854682cdae88))
* **deps:** update dependency @ory/client to v1.17.2 ([ce86ffd](https://github.com/canonical/identity-platform-login-ui/commit/ce86ffd32ef6cd0aefc21ff04430c3b0a4e7881f))
* **deps:** update dependency @ory/client to v1.17.2 ([7cab38e](https://github.com/canonical/identity-platform-login-ui/commit/7cab38ed7aaf885246c2660f2dd47d599b16d5c7))
* **deps:** update dependency sass to v1.85.0 ([8288db0](https://github.com/canonical/identity-platform-login-ui/commit/8288db024798c530ddc8557f4524399b5561e1f2))
* **deps:** update dependency sass to v1.85.0 ([61060b4](https://github.com/canonical/identity-platform-login-ui/commit/61060b492ab9c633cb07d2153d5ce84b515d065a))
* **deps:** update dependency vanilla-framework to v4.21.0 ([9529b7f](https://github.com/canonical/identity-platform-login-ui/commit/9529b7fd7bc53f9d61a33375851f8eb669956585))
* **deps:** update dependency vanilla-framework to v4.21.0 ([cf277a8](https://github.com/canonical/identity-platform-login-ui/commit/cf277a838d7228880f5da93624637352e2b64597))
* **deps:** update go deps ([a3f9f7e](https://github.com/canonical/identity-platform-login-ui/commit/a3f9f7ee1f159e0daf95fb5e1699f0cd6f12a9bd))
* **deps:** update go deps ([d9793b0](https://github.com/canonical/identity-platform-login-ui/commit/d9793b0ac1e46a7c35c119fed729e3d1225c74a7))
* **deps:** update go deps ([1724e00](https://github.com/canonical/identity-platform-login-ui/commit/1724e00d583d7c95478dea1fd60aab2f1ea4942a))
* **deps:** update go deps ([e7db62c](https://github.com/canonical/identity-platform-login-ui/commit/e7db62c446c17aa8ff245ff4051577084a0d5b54))
* **deps:** update go deps (minor) ([73938ab](https://github.com/canonical/identity-platform-login-ui/commit/73938abb8dda589431d8b0eb5f3097d599466286))
* **deps:** update go deps (patch) ([8f8d329](https://github.com/canonical/identity-platform-login-ui/commit/8f8d3292345bb04595b1cb8a8123e590c280f6eb))
* **deps:** update go deps to v1.35.0 ([61e573e](https://github.com/canonical/identity-platform-login-ui/commit/61e573e28dfb066c514a2b64c1fa3655f89e3481))
* **deps:** update go deps to v1.35.0 (minor) ([e38aa9e](https://github.com/canonical/identity-platform-login-ui/commit/e38aa9e83d998955bc7ce708fa0b32d5b8cfe946))
* **deps:** update module github.com/openfga/go-sdk to v0.7.0 ([447a1e4](https://github.com/canonical/identity-platform-login-ui/commit/447a1e41e6746ba0fe1ad3d4016a98a871bcb726))
* **deps:** update module github.com/openfga/go-sdk to v0.7.0 ([707f616](https://github.com/canonical/identity-platform-login-ui/commit/707f6160a288c66ab4031da578b23ecf2274610a))
* **deps:** update module github.com/ory/kratos-client-go to v1.3.5 ([f057485](https://github.com/canonical/identity-platform-login-ui/commit/f057485d2333c396af56d9867638c2fb8c807922))
* **deps:** update module github.com/ory/kratos-client-go to v1.3.5 ([7ea4c06](https://github.com/canonical/identity-platform-login-ui/commit/7ea4c06819747db8356824e8f6ccf42e700e4869))
* **deps:** update module github.com/ory/kratos-client-go to v1.3.6 ([d68830a](https://github.com/canonical/identity-platform-login-ui/commit/d68830af989e7be91003d823c006592932c3285f))
* **deps:** update module github.com/ory/kratos-client-go to v1.3.6 ([6fec4d5](https://github.com/canonical/identity-platform-login-ui/commit/6fec4d56d19401a3948e608c305df8bf2cd9b033))
* **deps:** update module github.com/ory/kratos-client-go to v1.3.8 ([12da2e9](https://github.com/canonical/identity-platform-login-ui/commit/12da2e9d86068a579404fc57bdc04dbb2376ca80))
* **deps:** update module github.com/ory/kratos-client-go to v1.3.8 ([4cd9e47](https://github.com/canonical/identity-platform-login-ui/commit/4cd9e47708f7df62407da182b189fdbc4602c7fc))
* **deps:** update module github.com/prometheus/client_golang to v1.21.0 ([ed04e8f](https://github.com/canonical/identity-platform-login-ui/commit/ed04e8f007e0f21e1a2d404ddf7d398626cde6da))
* **deps:** update module github.com/prometheus/client_golang to v1.21.0 ([c1fc0c9](https://github.com/canonical/identity-platform-login-ui/commit/c1fc0c9108544704ab7fbe67a8ddce900d05c792))
* **deps:** update module github.com/prometheus/client_golang to v1.21.1 ([eaadf04](https://github.com/canonical/identity-platform-login-ui/commit/eaadf04969f27b492824ece8587880f4715dee03))
* **deps:** update module github.com/prometheus/client_golang to v1.21.1 ([ea5fe2e](https://github.com/canonical/identity-platform-login-ui/commit/ea5fe2e6f1aefa7f725b7c1fbe6e681626a1adb6))
* **deps:** update module github.com/prometheus/client_golang to v1.22.0 ([9057433](https://github.com/canonical/identity-platform-login-ui/commit/905743379f37bf0ab3414ec20b5d292800379ed1))
* **deps:** update module github.com/prometheus/client_golang to v1.22.0 ([7a74d33](https://github.com/canonical/identity-platform-login-ui/commit/7a74d339826f647cbf23b5c6f56fa3a9f8dad036))
* **deps:** update module github.com/spf13/cobra to v1.9.0 ([bbae1d2](https://github.com/canonical/identity-platform-login-ui/commit/bbae1d23c07c8d1d6b367f063fbd9dbd6d715aaa))
* **deps:** update module github.com/spf13/cobra to v1.9.0 ([c4297d0](https://github.com/canonical/identity-platform-login-ui/commit/c4297d01537cdddf49f9541d39a67e719dcc305f))
* **deps:** update module github.com/spf13/cobra to v1.9.1 ([5f1df12](https://github.com/canonical/identity-platform-login-ui/commit/5f1df12b0a4001d2caa7696c1d090253de2d3ac3))
* **deps:** update module github.com/spf13/cobra to v1.9.1 ([3f47047](https://github.com/canonical/identity-platform-login-ui/commit/3f470477f786655e8ef55ae31c32b11ea044cf99))
* **deps:** update ui deps ([e8727f5](https://github.com/canonical/identity-platform-login-ui/commit/e8727f5bbbee21bb2e6a710167a3560002caea4b))
* **deps:** update ui deps ([f55713f](https://github.com/canonical/identity-platform-login-ui/commit/f55713fbf291d8e15022866ee5ca57f8b470a7af))
* **deps:** update ui deps ([05ce886](https://github.com/canonical/identity-platform-login-ui/commit/05ce886a2a6aab4a9df9b56c3fc84c86e2b370e6))
* **deps:** update ui deps ([9081ab1](https://github.com/canonical/identity-platform-login-ui/commit/9081ab10d3b54604a3cc5986f6ff3b6caeffc1ce))
* **deps:** update ui deps ([949bb16](https://github.com/canonical/identity-platform-login-ui/commit/949bb16d8872f1dff1f5f2cad7f0b965c893583c))
* **deps:** update ui deps ([e6484f5](https://github.com/canonical/identity-platform-login-ui/commit/e6484f50a396f59be512e0350a224b255aed5d18))
* **deps:** update ui deps ([fff28d3](https://github.com/canonical/identity-platform-login-ui/commit/fff28d3d7b923baf813205839971aee517f05ce5))
* **deps:** update ui deps ([ec0ee96](https://github.com/canonical/identity-platform-login-ui/commit/ec0ee96b97792e0f5df82784d50350795f820ebe))
* **deps:** update ui deps ([129f71d](https://github.com/canonical/identity-platform-login-ui/commit/129f71d47b4b672dd6b616be3b4010c4737f64e7))
* **deps:** update ui deps (minor) ([fa8532e](https://github.com/canonical/identity-platform-login-ui/commit/fa8532eb5a009b257732090359cf309dc586da69))
* **deps:** update ui deps (minor) ([8b34256](https://github.com/canonical/identity-platform-login-ui/commit/8b342568cf7b2a67bdbfd2e2371e303c6ccb73b6))
* **deps:** update ui deps (minor) ([326fcff](https://github.com/canonical/identity-platform-login-ui/commit/326fcffe01cc5b8f677081bf9b83e07b5a4b219b))
* **deps:** update ui deps (minor) ([8935ec5](https://github.com/canonical/identity-platform-login-ui/commit/8935ec5405a2298d5cff1f0203c588f1447845dd))
* **deps:** update ui deps (minor) ([cc27ea9](https://github.com/canonical/identity-platform-login-ui/commit/cc27ea95b11d51d0bdca659f1e39802cbe2c9574))
* **deps:** update ui deps (patch) ([9ea0db0](https://github.com/canonical/identity-platform-login-ui/commit/9ea0db0862d6bc03b67d3112cd02b7874a5a6dc1))
* **deps:** update ui deps (patch) ([955e47f](https://github.com/canonical/identity-platform-login-ui/commit/955e47f3b6269de9170aba91c65c8fdf28c99eb1))
* **deps:** update ui deps (patch) ([e3c9705](https://github.com/canonical/identity-platform-login-ui/commit/e3c9705dd2dbc0e7467bd9f7182cf9a8aa10bfdd))
* **deps:** update ui deps (patch) ([021c1a9](https://github.com/canonical/identity-platform-login-ui/commit/021c1a9f38875a3e46569c10f4f3c8517cd4e113))
* **deps:** update ui deps to v15.1.7 ([3c0a6cc](https://github.com/canonical/identity-platform-login-ui/commit/3c0a6cc6f329ee074c6c8e6a034ce76e8935dd75))
* **deps:** update ui deps to v15.1.7 (patch) ([ef7b924](https://github.com/canonical/identity-platform-login-ui/commit/ef7b924a8af1dc2ab34c2c0bd641ce419ceaab44))
* **deps:** update ui deps to v15.2.0 ([afb26ba](https://github.com/canonical/identity-platform-login-ui/commit/afb26bac3bb4b90242ea8255b9fa06127cb7b33a))
* **deps:** update ui deps to v15.2.0 (minor) ([278b155](https://github.com/canonical/identity-platform-login-ui/commit/278b1556538c358ad2f4ca58285a22f0d0d0a41a))
* handle redirect_to in error ([e315de8](https://github.com/canonical/identity-platform-login-ui/commit/e315de80a5278f239fec66c7432419d69f833aef))
* hydrate flow with hydra req ([da06826](https://github.com/canonical/identity-platform-login-ui/commit/da06826b900636589bc9238d3eeefb850cb967de))
* implement handling webauthn submission requests ([63cea2e](https://github.com/canonical/identity-platform-login-ui/commit/63cea2e29366fa8e073ef0b3623aed00b40fdbf4))
* introduce `DEV` flag ([e69ece1](https://github.com/canonical/identity-platform-login-ui/commit/e69ece153f0f19b31ed0cee6aa7448954b0c2ddd))
* remove redundant 'to' ([0bf51ca](https://github.com/canonical/identity-platform-login-ui/commit/0bf51ca0dc16e3662f38e15a83eb0cde42a2f097))
* remove unused frontend code ([1278b68](https://github.com/canonical/identity-platform-login-ui/commit/1278b68b935bcc7e9b7081b6a43d431dd51dd04f))
* return 200 if browser needs to redirect ([7a3ee3b](https://github.com/canonical/identity-platform-login-ui/commit/7a3ee3b0637decf44a6c790c243a94c1cbdbb471))
* uniformly handle redirect responses ([26b0085](https://github.com/canonical/identity-platform-login-ui/commit/26b0085f2c298943e1b1a2141237f49f41eaed1e))
* update go version in rockcraft ([f49f31d](https://github.com/canonical/identity-platform-login-ui/commit/f49f31d72efbc72ee5c4792103420b1d5826de72))
* update hydra sdk ([cb27e82](https://github.com/canonical/identity-platform-login-ui/commit/cb27e8232af3db06de4aaa03c71033ed62d7d5a4))
* update hydra sdk ([a7462e5](https://github.com/canonical/identity-platform-login-ui/commit/a7462e582c4a62e8083de3a70401b18dce5db915))
* update openfga sdk ([bf34591](https://github.com/canonical/identity-platform-login-ui/commit/bf34591157118b6f16ada9ba71b43a816adbf624))
* update openfga sdk ([bc04bb6](https://github.com/canonical/identity-platform-login-ui/commit/bc04bb61b38f4c1794cf9f609d4737d9c515603d))
* use the backend to accept webauthn authn ([73afb6f](https://github.com/canonical/identity-platform-login-ui/commit/73afb6fad3d0d5bc68f54a78c8b4da2aeee64c90))

## [0.21.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.20.0...v0.21.0) (2025-01-28)


### Features

* add endpoint to get sequencing mode ([dd6c524](https://github.com/canonical/identity-platform-login-ui/commit/dd6c524ae5ec15640fc177709a396f86811a31cc))


### Bug Fixes

* **deps:** update dependency @canonical/react-components to v1.9.0 ([89d2842](https://github.com/canonical/identity-platform-login-ui/commit/89d28425f33b3aea1e7f947559cbb898db955b56))
* **deps:** update dependency @ory/client to v1.16.0 ([7693ca2](https://github.com/canonical/identity-platform-login-ui/commit/7693ca240992ec4b31c01601578ef330332210e7))
* **deps:** update dependency vanilla-framework to v4.19.0 ([fbd505c](https://github.com/canonical/identity-platform-login-ui/commit/fbd505cdcb6935118047a5c78e2ac128dad1f756))
* **deps:** update dependency vanilla-framework to v4.20.0 ([ff2ec04](https://github.com/canonical/identity-platform-login-ui/commit/ff2ec045dc7766053bbcd8b9803057c231b2bd5b))
* **deps:** update dependency vanilla-framework to v4.20.1 ([826300c](https://github.com/canonical/identity-platform-login-ui/commit/826300ca990cf3c7b8922df054395ad9f221aee1))
* **deps:** update dependency vanilla-framework to v4.20.2 ([359c55d](https://github.com/canonical/identity-platform-login-ui/commit/359c55d5373f877b90342199c3c122f034848615))
* **deps:** update dependency vanilla-framework to v4.20.3 ([a30a6cc](https://github.com/canonical/identity-platform-login-ui/commit/a30a6cccf4b4eb2eb52d80a306c9e0f540649268))

## [0.20.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.19.3...v0.20.0) (2025-01-13)


### Features

* support oidc and webauthn sequencing ([0a3b26e](https://github.com/canonical/identity-platform-login-ui/commit/0a3b26ee165e0583a273a80d9d0de01a48126efa))


### Bug Fixes

* **deps:** update dependency @canonical/react-components to v1.8.0 ([040d543](https://github.com/canonical/identity-platform-login-ui/commit/040d543e9fc3355343ed610e1e105d76183560df))

## [0.19.3](https://github.com/canonical/identity-platform-login-ui/compare/v0.19.2...v0.19.3) (2025-01-09)


### Bug Fixes

* **deps:** update dependency @canonical/react-components to v1 ([c2f5ba9](https://github.com/canonical/identity-platform-login-ui/commit/c2f5ba99a7ee68b2894234e8c6f551e8f044ae20))
* **deps:** update internal ui dependencies ([9ceb235](https://github.com/canonical/identity-platform-login-ui/commit/9ceb235d151248cb9b97f49931cb2f36108f5e02))
* **deps:** update ui deps ([fc98396](https://github.com/canonical/identity-platform-login-ui/commit/fc9839630e952133767706b2a8068cb07b115359))

## [0.19.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.19.1...v0.19.2) (2025-01-09)


### Bug Fixes

* upgrade net lib to address CVE-2024-45338 ([ae20da9](https://github.com/canonical/identity-platform-login-ui/commit/ae20da9eade9f3e7b1f12aec50beba5ea648f770)), closes [#397](https://github.com/canonical/identity-platform-login-ui/issues/397)

## [0.19.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.19.0...v0.19.1) (2025-01-08)


### Bug Fixes

* do not enforce re-authentication when user was offered to regenerate backup codes ([718be4d](https://github.com/canonical/identity-platform-login-ui/commit/718be4dc5dd5464ed02d39a63f0b7f93999f37f8))

## [0.19.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.6...v0.19.0) (2024-12-04)


### Features

* drop LOG_FILE env var ([9a97de7](https://github.com/canonical/identity-platform-login-ui/commit/9a97de7c895e3ab936af153b9b7156574de76fd6))


### Bug Fixes

* do not write logs to file ([031bd55](https://github.com/canonical/identity-platform-login-ui/commit/031bd5577d8927470f7ed039865701523e9d2b74))

## [0.18.6](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.5...v0.18.6) (2024-11-28)


### Bug Fixes

* set SettingsFlow.ContinueWith to nil to work around the json marshal error ([dddf488](https://github.com/canonical/identity-platform-login-ui/commit/dddf488861d16c22c4abfe76d481fd478919f075))
* use hydra CLI to perform OIDC flow ([ed22b51](https://github.com/canonical/identity-platform-login-ui/commit/ed22b512564e4e1088f305b0117f78e5cf1d2385))

## [0.18.5](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.4...v0.18.5) (2024-11-13)


### Bug Fixes

* update rock to go 1.23.2 to deal with CVE-2024-34156 ([9d6701a](https://github.com/canonical/identity-platform-login-ui/commit/9d6701a63ce47d1b1a37bee42dd71446a6bb9e33))

## [0.18.4](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.3...v0.18.4) (2024-11-13)


### Bug Fixes

* check hydra request skip value ([ca08006](https://github.com/canonical/identity-platform-login-ui/commit/ca08006a412c057635b4613ca0ceaaffb2b9a71a))
* remember hydra sessions ([84494d4](https://github.com/canonical/identity-platform-login-ui/commit/84494d45eeb530a55b3e41b7f18532902d4b6cb2))
* use cookies to handle first login ([f0ede5f](https://github.com/canonical/identity-platform-login-ui/commit/f0ede5f3623b267e8499cba24abf65d4c5763bd8))

## [0.18.3](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.2...v0.18.3) (2024-10-21)


### Bug Fixes

* update google.golang.org/grpc to address GHSA-m425-mq94-257g ([96e4f44](https://github.com/canonical/identity-platform-login-ui/commit/96e4f44ed014e6b067dd7834b4e62748631764ec))

## [0.18.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.1...v0.18.2) (2024-10-21)


### Bug Fixes

* address CVE-2023-39325 found by https://github.com/canonical/oci-factory/actions/runs/11406456591/job/31741561258 ([a68d463](https://github.com/canonical/identity-platform-login-ui/commit/a68d463eff782902f88fd1b479aeba72f3ec307f))
* **deps:** update ui deps ([6228aa2](https://github.com/canonical/identity-platform-login-ui/commit/6228aa227085d8829a8377d68a8c9f40daf89942))
* return error is no login_challenge or return_to ([1efbff4](https://github.com/canonical/identity-platform-login-ui/commit/1efbff4d3dd59505ead437cb844cb4c77e4e7047))

## [0.18.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.18.0...v0.18.1) (2024-10-16)


### Bug Fixes

* improve ui for reset password flow. fixes [#322](https://github.com/canonical/identity-platform-login-ui/issues/322) ([2dde4e7](https://github.com/canonical/identity-platform-login-ui/commit/2dde4e791e4a416ada18fa4d99434412c2d4fbc3))

## [0.18.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.17.0...v0.18.0) (2024-10-11)


### Features

* add security headers ([37241fd](https://github.com/canonical/identity-platform-login-ui/commit/37241fd6c06de9547bad783e43810d81989a0c26))


### Bug Fixes

* add return_to parameter for regenerate backup codes redirect ([4bb01a9](https://github.com/canonical/identity-platform-login-ui/commit/4bb01a9e7b5cfd6222f8117851b4a8b27f4abd52))
* delete session on recovery ([fa39b00](https://github.com/canonical/identity-platform-login-ui/commit/fa39b00a4f2ce922de44eb8d2b4e23435809eb73))
* **deps:** update go deps ([d7b601b](https://github.com/canonical/identity-platform-login-ui/commit/d7b601baf1c06a86ff13444c3f05253ca741fbf0))
* do not redirect to error, on no error ([4f74f07](https://github.com/canonical/identity-platform-login-ui/commit/4f74f07d365cfb1488b08b8b729033b90152ef58))
* do not send kratos session cookie on new login flow ([89fbe03](https://github.com/canonical/identity-platform-login-ui/commit/89fbe03a3abd384a82280822b94a10702f0d4d16))
* do not send session cookie on 1fa ([a951882](https://github.com/canonical/identity-platform-login-ui/commit/a9518824e6410206bc45ba956c9b00a5d62d2d2b))
* fix backup codes setup ([19254b1](https://github.com/canonical/identity-platform-login-ui/commit/19254b143591ebd74aefe574d8de37e35b14f20f))
* handle login flow without login_challenge ([b902910](https://github.com/canonical/identity-platform-login-ui/commit/b902910ba8905e9752830ca49df0d7162d8e52cf))
* pass the return_to from request ([d4e2399](https://github.com/canonical/identity-platform-login-ui/commit/d4e2399af22a6b2194890cc8b3fac73ae4942bfb))
* update the setup_secure page ([4f92c65](https://github.com/canonical/identity-platform-login-ui/commit/4f92c6576aed90c6c96067a500b2f23fa393edd4))

## [0.17.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.16.0...v0.17.0) (2024-09-04)


### Features

* support lookup_secret method, offer user to regenerate the set if 3 or less codes are left ([45738a8](https://github.com/canonical/identity-platform-login-ui/commit/45738a8ab87e57ee9ded79d070552f00f8ffb50b))

## [0.16.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.15.0...v0.16.0) (2024-08-14)


### Features

* add MFA_ENABLED to enable/disable mfa enforcing + KRATOS_ADMIN_URL ([511e2a2](https://github.com/canonical/identity-platform-login-ui/commit/511e2a2edb9b1c607650486737835412919147fd))
* implement method to check if totp is configured ([22c063e](https://github.com/canonical/identity-platform-login-ui/commit/22c063eb7fd63d056cc2149ac7f4c52c6b0b22fc))
* implement mfa enforcing ([7ff6496](https://github.com/canonical/identity-platform-login-ui/commit/7ff6496be73c5ff5fbc82597c9e700f385cb57ef))
* implement MFA enforcing for getConsent ([a17d0ae](https://github.com/canonical/identity-platform-login-ui/commit/a17d0aeb73352dc578a260d223abfac96b439aab))


### Bug Fixes

* pass request context to service ([0398690](https://github.com/canonical/identity-platform-login-ui/commit/0398690adc7f354100f82ba3fc911795e3806d6e))
* redirection in case session is available ([b71ab84](https://github.com/canonical/identity-platform-login-ui/commit/b71ab841f1ac135d9b533e817cdcc73aaf79a572))

## [0.15.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.14.0...v0.15.0) (2024-07-29)


### Features

* add settings flow ([fcbd7ec](https://github.com/canonical/identity-platform-login-ui/commit/fcbd7eceb246403c2c892f8b7fa8a2609b24f795))
* handle missing webauthn credentials error ([85979cd](https://github.com/canonical/identity-platform-login-ui/commit/85979cd9c02fdfb5b43f71b67590bdae5d3572ee))
* settings flow ([d6770c8](https://github.com/canonical/identity-platform-login-ui/commit/d6770c8e4604f33936daa6f61576e94957dffe48))
* support account recovery ([5fe26e5](https://github.com/canonical/identity-platform-login-ui/commit/5fe26e5e4918033fa1a8088d5442cf136fbc2372))
* support mfa with totp method ([16df279](https://github.com/canonical/identity-platform-login-ui/commit/16df2793f5882cb0a3e4b264352b62235842c3ed))
* support passwordless webauthn method ([ecfa6f4](https://github.com/canonical/identity-platform-login-ui/commit/ecfa6f414b003f70663b6503955cad10edef5eb9))
* ui support for mfa flows ([dc88e6b](https://github.com/canonical/identity-platform-login-ui/commit/dc88e6bf92e276583a8a9abe93bfb8d6e4f18368))


### Bug Fixes

* add client id to at aud ([20fed79](https://github.com/canonical/identity-platform-login-ui/commit/20fed796d51fff3689ea169e8fddb58c0dfcb5f3))
* group -&gt; parent/child relationship + add spaces for readability ([da553c4](https://github.com/canonical/identity-platform-login-ui/commit/da553c46155e078dd241a8366e547f349285447f))
* handle invalid code case ([bbfb9f1](https://github.com/canonical/identity-platform-login-ui/commit/bbfb9f12ab93587b41f29f2507dd488c3c0fd972))
* handle invalid recovery code case (wip) ([c60343a](https://github.com/canonical/identity-platform-login-ui/commit/c60343a14f95df240988fd0df1df7da9e6e895a9))
* pass cookie when creating settings flow to enable password change once logged in ([0fc9bfc](https://github.com/canonical/identity-platform-login-ui/commit/0fc9bfc2d6d5ce9a72d1cd51e93063b2d4c6b9b7))
* remove certificates from image ([bfdd295](https://github.com/canonical/identity-platform-login-ui/commit/bfdd295b22a77608fd4bce5209e2f3f14af672f3))
* remove unnecessary logging ([f446110](https://github.com/canonical/identity-platform-login-ui/commit/f4461102f17f18767a2afce03dd3f6a3e108b2b4))
* return to url with flow id ([39d299c](https://github.com/canonical/identity-platform-login-ui/commit/39d299c72d21a9814e054b6314db69eb43f60644))
* update docker compose ([533a70c](https://github.com/canonical/identity-platform-login-ui/commit/533a70caee68fd0028f29510a510fcce9d306584))

## [0.14.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.13.0...v0.14.0) (2024-04-30)


### Features

* add error handling for login flow WD-10252 ([f96567e](https://github.com/canonical/identity-platform-login-ui/commit/f96567e7901b8b41c9b5cabff09bf49139a8ddc8))
* return common errors to ui ([5828a6a](https://github.com/canonical/identity-platform-login-ui/commit/5828a6a28e6b36dd45ae53a5bdee7ec8d895848f))
* support password method ([f8842f0](https://github.com/canonical/identity-platform-login-ui/commit/f8842f0d7edc386c8ac7a29064c48487facb65e9))


### Bug Fixes

* fix path ([e7ffc03](https://github.com/canonical/identity-platform-login-ui/commit/e7ffc03d718fb7507de3c162009b54ea2ed8b595))
* password login method WD-10252 ([1e98e09](https://github.com/canonical/identity-platform-login-ui/commit/1e98e094dd37e1e2865848ed3794e09f227db3ce))
* use relative path for callback ([4aa6c83](https://github.com/canonical/identity-platform-login-ui/commit/4aa6c83ea98ce0905f23e60626258dafade874cc))

## [0.13.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.12.0...v0.13.0) (2024-04-16)


### Features

* add success screen for device flow WD-10251 ([e9b77db](https://github.com/canonical/identity-platform-login-ui/commit/e9b77dba63dfdb622460bf1c6a6add1aff923aab))


### Bug Fixes

* add device flow logic to hydra client ([ca4a1f9](https://github.com/canonical/identity-platform-login-ui/commit/ca4a1f99c999ef57a42bb4b5c3c95ff11a891487))
* add error handling to the device_code page ([666f95b](https://github.com/canonical/identity-platform-login-ui/commit/666f95b6ad4e38bfc0c07879f0ccc626c6edcbf7))
* **CheckAllowedProviders:** fix OAuthKeeper case ([24212f4](https://github.com/canonical/identity-platform-login-ui/commit/24212f4decc0c5e76e74a21ffc21eab8eae08feb))
* implement device_code backend logic ([06bb0b2](https://github.com/canonical/identity-platform-login-ui/commit/06bb0b29105aab10e87ada456d932899f7c8ef3e))
* pass error response from hydra to UI ([b165f43](https://github.com/canonical/identity-platform-login-ui/commit/b165f43d8e3e83d6bfcf595088c94497203b3bd4))
* typo in Oathkeeper name ([5f7634d](https://github.com/canonical/identity-platform-login-ui/commit/5f7634dca31afb854be4ff0254659c5c51859d2d))
* update device_code page ([4ca0d13](https://github.com/canonical/identity-platform-login-ui/commit/4ca0d138e4eb850a5e10cf5ae1ea384035b105c8))
* update hydra config ([cb1e049](https://github.com/canonical/identity-platform-login-ui/commit/cb1e0493d7c95c0b1b980c5a18bc3bdcede96110))

## [0.12.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.11.3...v0.12.0) (2024-01-31)


### Features

* Add authorization logic for allowed providers ([5da85b7](https://github.com/canonical/identity-platform-login-ui/commit/5da85b785a49e11600ce8f342c80ecd20bbd38bf))
* Add authorizer struct ([c6170a2](https://github.com/canonical/identity-platform-login-ui/commit/c6170a209c49003dbf19ff277a0d2b3e1b268a53))
* add create-fga-model CLI command ([42f47df](https://github.com/canonical/identity-platform-login-ui/commit/42f47df508312f4a28a3a0d4a71859d40c053ed4))
* add fallback logo for identity providers ([ca957ee](https://github.com/canonical/identity-platform-login-ui/commit/ca957ee6db4f20f7b0c89fe172b2b2f9d446cdb8))
* Add openfga client ([8628fa6](https://github.com/canonical/identity-platform-login-ui/commit/8628fa6a41805519cc99b4aaea94d3f857d7079e))
* remove okta logo from login providers list ([88bb621](https://github.com/canonical/identity-platform-login-ui/commit/88bb621c8cd82334d91219c8f5c8d983fe81fc26))
* use cobra for CLI ([50963a3](https://github.com/canonical/identity-platform-login-ui/commit/50963a396ad4c64023c4e5d28b53b4a456f71eb4))
* use new design and add user flows as dummy pages WD-8469 ([2ac0e1a](https://github.com/canonical/identity-platform-login-ui/commit/2ac0e1addcad6ef899fcdb5d10aee653da7cb019))


### Bug Fixes

* add noop clients ([5b6f62a](https://github.com/canonical/identity-platform-login-ui/commit/5b6f62a9ce35cdd5c79463c34fe780a64aa3fbe1))
* bump otel/trace version ([d2f65fc](https://github.com/canonical/identity-platform-login-ui/commit/d2f65fc35bb53b938beeca70132851e3674ec217))
* readme typo ([80de6dd](https://github.com/canonical/identity-platform-login-ui/commit/80de6dd598c6a5b778ea43123bf9bc32b185aec1))
* switch to zap nop logger ([57b37f8](https://github.com/canonical/identity-platform-login-ui/commit/57b37f8b7102b4b62057f27c9648e6546e9a0e67))
* update rockcraft.yaml ([c595939](https://github.com/canonical/identity-platform-login-ui/commit/c595939e6ebffa562fa9597edfc9d677dda9280c))
* update ui ([e862475](https://github.com/canonical/identity-platform-login-ui/commit/e8624753928602ff5664d8bdc601705c2a2cdb76))
* Use github as test provider ([71eb388](https://github.com/canonical/identity-platform-login-ui/commit/71eb388594889c4707a1703fece37c41ea564b09))
* Use label to generate button text ([0494160](https://github.com/canonical/identity-platform-login-ui/commit/0494160446f5320989224e8b40b6aaa749eed7a1))

## [0.11.3](https://github.com/canonical/identity-platform-login-ui/compare/v0.11.2...v0.11.3) (2023-11-01)


### Bug Fixes

* **deps:** update dependency @canonical/react-components to v0.47.1 ([529c5d6](https://github.com/canonical/identity-platform-login-ui/commit/529c5d676039cb47b2e6f45404c59800e8326196))
* **deps:** update dependency vanilla-framework to v4.4.0 ([b9e3b03](https://github.com/canonical/identity-platform-login-ui/commit/b9e3b0395cbec8fde99705bdb4562979c1eff0a1))
* **deps:** update dependency vanilla-framework to v4.5.0 ([bd40882](https://github.com/canonical/identity-platform-login-ui/commit/bd40882423ec6fed74ddd9aa77d6aa236a7315d3))
* downgrade Kratos sdk ([b36de36](https://github.com/canonical/identity-platform-login-ui/commit/b36de3666f7732f26df52f07dafa9bda8090287f))

## [0.11.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.11.1...v0.11.2) (2023-10-04)


### Bug Fixes

* add app version config ([78e93b5](https://github.com/canonical/identity-platform-login-ui/commit/78e93b5fd572c9a988bb1ed0f56989e51dd38045))
* add flag parsing logic ([39123a1](https://github.com/canonical/identity-platform-login-ui/commit/39123a13be19dfef2ddf6c88dc78d625ed925ec0))
* **deps:** update go deps ([71edbe7](https://github.com/canonical/identity-platform-login-ui/commit/71edbe7273671004547899189e6be097853c3939))
* **deps:** update go deps to v1.19.0 ([1a880ce](https://github.com/canonical/identity-platform-login-ui/commit/1a880cefbf2bf8dfe76f8e706281677425e4b097))
* **deps:** update module github.com/ory/kratos-client-go to v1 ([8a4894a](https://github.com/canonical/identity-platform-login-ui/commit/8a4894a5fd8d5838eb4121575e37015077554fba))
* **deps:** update module github.com/prometheus/client_golang to v1.17.0 ([1066d13](https://github.com/canonical/identity-platform-login-ui/commit/1066d13023735222f2e5505c36d4c2869c04dadf))
* **deps:** update module go.uber.org/zap to v1.26.0 ([9041a1a](https://github.com/canonical/identity-platform-login-ui/commit/9041a1aae951dc4c7c35a0fe6b382bab3fbf7c93))
* do not prefix version with v ([cdc9cf1](https://github.com/canonical/identity-platform-login-ui/commit/cdc9cf15a1c05afcd858bebbc2b80c7712f9392a))
* IAM-514 compare json payloads to verify object is the same ([fde33fa](https://github.com/canonical/identity-platform-login-ui/commit/fde33fa1557d325d9e725123652d5df74ca9f936))
* move version in a separate package ([6f3c03b](https://github.com/canonical/identity-platform-login-ui/commit/6f3c03b7d0e971324db040820b74d234f766122f))

## [0.11.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.11.0...v0.11.1) (2023-09-14)


### Bug Fixes

* add debug flag ([e32b5c0](https://github.com/canonical/identity-platform-login-ui/commit/e32b5c059acd5d73a12786ce7f22555b3e940863)), closes [#155](https://github.com/canonical/identity-platform-login-ui/issues/155)
* **deps:** update dependency vanilla-framework to v4.3.0 ([e19304c](https://github.com/canonical/identity-platform-login-ui/commit/e19304c6ff15d4746659a1cc08fb0c34f29e75a6))
* **deps:** update go deps ([20cc702](https://github.com/canonical/identity-platform-login-ui/commit/20cc70269222d39e444465e451e0bbc64729dd17))
* **deps:** update go deps to v1.18.0 ([6bda56c](https://github.com/canonical/identity-platform-login-ui/commit/6bda56c2e7f27f94687fe9efd9d1204028b5ff68))

## [0.11.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.10.0...v0.11.0) (2023-09-12)


### Features

* added ca-certificates package to stage-packages ([e09f07c](https://github.com/canonical/identity-platform-login-ui/commit/e09f07c1d1d932d345cc6fba330b664d57d696f1))
* removed index.tsx and registration.tsx from ui/pages ([0e15096](https://github.com/canonical/identity-platform-login-ui/commit/0e15096c8f942b18223da1cb178d7484de69d344))


### Bug Fixes

* **deps:** update dependency vanilla-framework to v4 ([27079f6](https://github.com/canonical/identity-platform-login-ui/commit/27079f606565911b94df0af6b69fc55bf1c86ad0))
* **deps:** update go deps ([6658d1a](https://github.com/canonical/identity-platform-login-ui/commit/6658d1a03809876becbe16aa37b30c6423d43616))
* **deps:** update internal ui dependencies ([72f9278](https://github.com/canonical/identity-platform-login-ui/commit/72f9278974523beac9e92310bf1ea747df881e2d))
* **deps:** update module github.com/go-chi/chi/v5 to v5.0.10 ([e5042c9](https://github.com/canonical/identity-platform-login-ui/commit/e5042c9cef0e1d56694a2605e5374ed804c5993b))
* remove renovate workflow ([991215f](https://github.com/canonical/identity-platform-login-ui/commit/991215f7c19534a2b20882d47572059acff51e72))
* tranform Kratos 422 HTTP resp to 200 ([3deebc4](https://github.com/canonical/identity-platform-login-ui/commit/3deebc469366bdddf05f58295fddfa39e9ce30fc))
* Use variables instead of ints for HTTP statuses ([9332221](https://github.com/canonical/identity-platform-login-ui/commit/93322212e82caaff61c6239d7ce188d249affa57))

## [0.10.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.9.0...v0.10.0) (2023-08-23)


### Features

* add healthcheck package for background tasks ([4b3fe14](https://github.com/canonical/identity-platform-login-ui/commit/4b3fe148f36cc78e07aebdb8eea3f92e64956c5f))


### Bug Fixes

* fixed bug in consent page ([d1a8e7a](https://github.com/canonical/identity-platform-login-ui/commit/d1a8e7acf407ae806cb0f16eefde6566524447a2))
* move buildInfo into service ([a8a0115](https://github.com/canonical/identity-platform-login-ui/commit/a8a01150ed04908a7aedea2aa7589b27905d4252))
* use check statuses in the service layer ([1d89f0f](https://github.com/canonical/identity-platform-login-ui/commit/1d89f0f134e4cb5e16570f198558bf9b2861e4ca))

## [0.9.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.8.2...v0.9.0) (2023-08-22)


### Features

* add instrumentation to deep health check ([f565c50](https://github.com/canonical/identity-platform-login-ui/commit/f565c50f20e15e580475cce5116a0cddff6f7199))


### Bug Fixes

* add set methods for each metric ([df62e32](https://github.com/canonical/identity-platform-login-ui/commit/df62e3272964cdc952d43b1c5fc02f978eaeb02c))
* adjust wiring of status pkg ([1d69a58](https://github.com/canonical/identity-platform-login-ui/commit/1d69a58f35e2173a35623d34abeb210322624d1c))
* drop MetricInterface and adjust interface methods ([50542f6](https://github.com/canonical/identity-platform-login-ui/commit/50542f6098eb52eb7c6811251523ac7f89ac07b6))
* fixed handlers for kratos api proxying ([2ac0764](https://github.com/canonical/identity-platform-login-ui/commit/2ac0764bcccdc68acb55dd5d88187bec71473a16))

## [0.8.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.8.1...v0.8.2) (2023-08-17)


### Bug Fixes

* use the same import and drop an alias ([5af7944](https://github.com/canonical/identity-platform-login-ui/commit/5af79443803655e50e6bb392d2c8cceaf5688110))

## [0.8.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.8.0...v0.8.1) (2023-08-15)


### Bug Fixes

* fixed timeout in deepcheck handler ([511f86f](https://github.com/canonical/identity-platform-login-ui/commit/511f86feadf3cde8898120ba1c47f98d46e400e1))

## [0.8.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.7.0...v0.8.0) (2023-08-15)


### Features

* added new /api/v0/deepcheck route ([7ccc8eb](https://github.com/canonical/identity-platform-login-ui/commit/7ccc8eb41391fb87d96d40a09159d5986be1365f))
* added service layer to pkg/status ([aee2027](https://github.com/canonical/identity-platform-login-ui/commit/aee202779dc417a405013b2a7f50dd50a24e8a8b))
* adjust the wiring of the status pkg ([2b6b66c](https://github.com/canonical/identity-platform-login-ui/commit/2b6b66c92a83e23311bec21d0bad3808f4ae482a))


### Bug Fixes

* add MetadataAPI methods to clients objects ([c5e76e1](https://github.com/canonical/identity-platform-login-ui/commit/c5e76e1de728c38c2e93c958198fc670e8fc2425))

## [0.7.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.6.3...v0.7.0) (2023-08-15)


### Features

* changed the route to the ui resources to /ui/* ([bff64e5](https://github.com/canonical/identity-platform-login-ui/commit/bff64e541ded6766c9feae668fc72468f326ec97))


### Bug Fixes

* fixed server treating all misc routes as fileserver requests ([b8b4ded](https://github.com/canonical/identity-platform-login-ui/commit/b8b4dedb5d69d50bab645e8aed34ff8b5ee92aa9))

## [0.6.3](https://github.com/canonical/identity-platform-login-ui/compare/v0.6.2...v0.6.3) (2023-07-27)


### Bug Fixes

* use otelhttp transport to propagate traces ([fe14b59](https://github.com/canonical/identity-platform-login-ui/commit/fe14b5970bbef93ec6809f9626912b0ad8f25194))

## [0.6.2](https://github.com/canonical/identity-platform-login-ui/compare/v0.6.1...v0.6.2) (2023-07-26)


### Bug Fixes

* add jaeger propagator as ory components support only these spans for now ([0b5f248](https://github.com/canonical/identity-platform-login-ui/commit/0b5f2483020b83cc69bea1fbdf6788b601c0005f))
* add otel grpc+http endpoint for tracing ([e1b1424](https://github.com/canonical/identity-platform-login-ui/commit/e1b14247c3ab02a44f3ca09b243bfd34c747c2c0))
* pass new context to clients to propagate trace ids ([7dfdf05](https://github.com/canonical/identity-platform-login-ui/commit/7dfdf0503320c9056b84ebe4763682097f213692))
* wire up new config needed for otel grpc+http ([17399ea](https://github.com/canonical/identity-platform-login-ui/commit/17399ea9219a06d522f213658bb8f3c88135ad32))

## [0.6.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.6.0...v0.6.1) (2023-07-11)


### Bug Fixes

* Change response status to 200 ([#94](https://github.com/canonical/identity-platform-login-ui/issues/94)) ([89af73b](https://github.com/canonical/identity-platform-login-ui/commit/89af73baf5cfe384b48ce05f64fb435d36bdd3a0))
* Copy only cookies in proxied requests ([#95](https://github.com/canonical/identity-platform-login-ui/issues/95)) ([2d1fa6a](https://github.com/canonical/identity-platform-login-ui/commit/2d1fa6aad9d47127e6c914cc5389fc94beb35a1c))

## [0.6.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.4.1...v0.6.0) (2023-07-07)


### Bug Fixes

* add cors middleware ([9d5cb04](https://github.com/canonical/identity-platform-login-ui/commit/9d5cb0412a31c77a377ee349219ccc5e5c9c3b91))
* Move logic out of misc package ([8ae6da5](https://github.com/canonical/identity-platform-login-ui/commit/8ae6da52532661af33ea86e06bc6a3a594c8f22a))


## [0.4.1](https://github.com/canonical/identity-platform-login-ui/compare/v0.4.0...v0.4.1) (2023-06-28)


### Bug Fixes

* add tracing enabling variable, defaulting to true ([f0858e3](https://github.com/canonical/identity-platform-login-ui/commit/f0858e3d98083e2e8b92ab4dc1f74e039976bb32))
* IAM-353 - allow tracing to be disabled ([1ff3186](https://github.com/canonical/identity-platform-login-ui/commit/1ff3186a5105a0b3c1fbdf854be53963a88e8a95))

## [0.4.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.3.0...v0.4.0) (2023-06-27)


### Features

* IAM-326 - separate logic and http handling in extra package ([7f9549c](https://github.com/canonical/identity-platform-login-ui/commit/7f9549cab52c87c3f877a3206f6193cacad336ff))
* IAM-330 - introduce otel tracing ([198fb16](https://github.com/canonical/identity-platform-login-ui/commit/198fb16b0ed223433f297dae25e5012c53aece84))
* IAM-330 - use otelhttp middleware ([5efc2a0](https://github.com/canonical/identity-platform-login-ui/commit/5efc2a0dda0b6ffd7f8460826cd4ff8b98469816))
* wire up service in extra package ([9865de8](https://github.com/canonical/identity-platform-login-ui/commit/9865de8ce3e95ae752aaa1ea4b141e1f5d490361))
* wire up tracer inside status endpoint as a dummy example ([b51e91f](https://github.com/canonical/identity-platform-login-ui/commit/b51e91f8b6d1ea7922c91541689af7933da45215))

## [0.3.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.2.0...v0.3.0) (2023-06-27)


### Features

* add log rotator for zap ([dc76368](https://github.com/canonical/identity-platform-login-ui/commit/dc763681a85c492121e163e1526b901a75ca3849))
* IAM-327 - introduce zap ([9e378e8](https://github.com/canonical/identity-platform-login-ui/commit/9e378e8f2cac263a54f67c16cd63eff5ecb2f0e5))
* IAM-327 - wire up zap.SugaredLogger ([6e22236](https://github.com/canonical/identity-platform-login-ui/commit/6e222368d8e58c184d4ca319200aed5ae1c685bb))
* IAM-328 - use chi http logger middleware ([79283d8](https://github.com/canonical/identity-platform-login-ui/commit/79283d8ad51cabcf2e716c8116af2d95670cbfa1))
* IAM-328 - use go-chi for router ([55c873d](https://github.com/canonical/identity-platform-login-ui/commit/55c873d08b633a20cba76d2b063e2f71de8f9125))
* introduce config management via envconfig ([55367ee](https://github.com/canonical/identity-platform-login-ui/commit/55367ee0d2694fe3634f1a0dd79270562c857b66))
* wire up new web pkg for routing ([0d67eda](https://github.com/canonical/identity-platform-login-ui/commit/0d67eda2569ce8637663cb6ceeacd1b16013eaed))


### Bug Fixes

* add * to UI route ([0c7d1d0](https://github.com/canonical/identity-platform-login-ui/commit/0c7d1d05b24c525241f9c196cd181150c5d9ef56))
* create prometheus middleware for chi ([1678f56](https://github.com/canonical/identity-platform-login-ui/commit/1678f56033ba6dc5d28f871d7158bb6abca32072))
* rename prometheus pkg to metrics and change endpoints ([ca99c3c](https://github.com/canonical/identity-platform-login-ui/commit/ca99c3c9c72132c338223df60387c3244c4063c3))

## [0.2.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.1.0...v0.2.0) (2023-06-22)


### Features

* add git commit sha on health endpoint ([a09afce](https://github.com/canonical/identity-platform-login-ui/commit/a09afce63dedf08b6e38a348e9fdca05c0ceaaba))
* create an oidc pkg to be shared ([91e199f](https://github.com/canonical/identity-platform-login-ui/commit/91e199f760d83ba5899108d9d2a8c0473431990a))
* create hydra and kratos pkgs, useful for mocking ([e585da8](https://github.com/canonical/identity-platform-login-ui/commit/e585da8ba89f9f21d020446cbb84453d7e1c4981))
* move conset api in a separate pkgs ([895d392](https://github.com/canonical/identity-platform-login-ui/commit/895d392b6d1ec0b2d5a9d7b714457cce43579cff))
* move kratos api set in a separate pkgs ([0f4e552](https://github.com/canonical/identity-platform-login-ui/commit/0f4e552c05fde1eff5cd752d448a5bb17bf20e22))
* move ui api in a separate pkg ([7142e08](https://github.com/canonical/identity-platform-login-ui/commit/7142e08dfd8e26f86c643175e6083a8b67252c79))
* offload logic to packages ([3226beb](https://github.com/canonical/identity-platform-login-ui/commit/3226bebc9a84ceb51d8d3e5b5a4e08b5b277802c))


### Bug Fixes

* add graceful server shutdown ([d67989f](https://github.com/canonical/identity-platform-login-ui/commit/d67989f277409d9b1e93068be7e5b83c9e5933be))
* move helpers to a shared internal package, plan to remove ([c160874](https://github.com/canonical/identity-platform-login-ui/commit/c1608749c35bd1367cebd59cc5662b7121afb694))
* move logging pkg into internal ([23b32d4](https://github.com/canonical/identity-platform-login-ui/commit/23b32d4bbde66e9d2183ca3d05c95fa7c2e7580d))
* move tests into relative packages, rename health to status ([29d58e1](https://github.com/canonical/identity-platform-login-ui/commit/29d58e10a605ed6a12cefa686281239b001da092))
* use full repo url for go module name ([3c8bc22](https://github.com/canonical/identity-platform-login-ui/commit/3c8bc223db7dde172571d8a0879371394f83f4b8))
* wire up new pkgs handlers ([ed7e9be](https://github.com/canonical/identity-platform-login-ui/commit/ed7e9becc3ac981ded352b531c72ff9d95bcfdca))
