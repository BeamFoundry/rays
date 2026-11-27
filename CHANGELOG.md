# Changelog

## [0.6.0-rc3](https://github.com/BeamFoundry/rays/compare/v0.5.0-rc3...v0.6.0-rc3) (2026-11-27)


### Features

* **scheduler,container:** add fixed-pool task scheduler, Scheduled a… ([#34](https://github.com/BeamFoundry/rays/issues/34)) ([8af8845](https://github.com/BeamFoundry/rays/commit/8af8845dc2bb898ea37a283113f866e52acbdda7))

## [0.5.0-rc3](https://github.com/BeamFoundry/rays/compare/v0.5.0-rc2...v0.5.0-rc3) (2026-11-24)


### Features

* **core,logger:** add WrapLoggingHandler and BeamPreDestroy shutdown… ([#31](https://github.com/BeamFoundry/rays/issues/31)) ([cbca01b](https://github.com/BeamFoundry/rays/commit/cbca01bc69863617dfb77bb77a29f6ff981a6c94))


### Miscellaneous Chores

* **release:** Prepare version 0.5.0-rc3 ([2cfc3e4](https://github.com/BeamFoundry/rays/commit/2cfc3e4039a94e6f90d8751403f267f81a9596ca))

## [0.5.0-rc2](https://github.com/BeamFoundry/rays/compare/v0.5.0-rc1...v0.5.0-rc2) (2026-11-20)


### Bug Fixes

* **container:** restore non-blocking error handling for slice, map, a… ([#29](https://github.com/BeamFoundry/rays/issues/29)) ([291b929](https://github.com/BeamFoundry/rays/commit/291b929b17232a45bbd7893f4b1369250622ae50))
* **container:** restore non-blocking error handling for slice, map, and optional injections ([291b929](https://github.com/BeamFoundry/rays/commit/291b929b17232a45bbd7893f4b1369250622ae50))


### Miscellaneous Chores

* **release:** prepare version v0.5.0-rc2 ([dc42dd4](https://github.com/BeamFoundry/rays/commit/dc42dd4b9afa508ecfeee591f7964f8d246ec43e))

## [0.5.0-rc1](https://github.com/BeamFoundry/rays/compare/v0.4.2...v0.5.0-rc1) (2026-11-20)


### ⚠ BREAKING CHANGES

* **tx:** Transactional[T] now embeds TransactionProvider and only exposes Get() *T; transaction lifecycle and goroutine-local context.Context are managed via TransactionManager.

### Features

* **container:** rename BeamPostConstructWithContext to BeamPrototypeWithContext and enforce Prototype hook order ([8879414](https://github.com/BeamFoundry/rays/commit/887941437130f88fc8c6b0dd80e2e12364c6b95d))
* **tx:** add coordinated TransactionManager and propagate nested Pr… ([#27](https://github.com/BeamFoundry/rays/issues/27)) ([c441738](https://github.com/BeamFoundry/rays/commit/c4417385e3be504fffede0e6a1cd008c85e70ba5))


### Bug Fixes

* **container:** guard InnerBeamPostConstruct reflection against unexported embedded structs ([b624efe](https://github.com/BeamFoundry/rays/commit/b624efeecd84b533e768cfddbb251b115cbcd9e8))
* **container:** reject Prototype provider parents, run hooks on interface-backed prototypes, and harden doc examples ([fe14485](https://github.com/BeamFoundry/rays/commit/fe14485ca359026f3350d4d6f605b89a1e2fb893))
* **release:** Restore .release-please-manifest.json file ([6d56788](https://github.com/BeamFoundry/rays/commit/6d5678897c4f748c63705abcadf12c1a44067198))


### Miscellaneous Chores

* prepare v0.5.0-rc1 ([d0c5df9](https://github.com/BeamFoundry/rays/commit/d0c5df9c9b400ea0bf8768ec926d1c492352b44e))
