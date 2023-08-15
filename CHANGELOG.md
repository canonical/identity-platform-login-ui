# Changelog

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
