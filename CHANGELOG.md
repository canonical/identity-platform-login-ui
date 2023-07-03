# Changelog

## [0.5.0](https://github.com/canonical/identity-platform-login-ui/compare/v0.4.1...v0.5.0) (2023-07-03)


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
