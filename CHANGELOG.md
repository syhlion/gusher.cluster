## [Unreleased]

## [V1.5.1] 2018-03-29

### [Fix]
- add buffer
- update websocket pkg

## [V1.5.0] 2018-03-15

### [Fix] 
- update garyburd/redigo to gomodule/redigo

### [Add]
- add feature api. push message by pattern


## [V1.4.1] 2018-01-12

### [Fix]
- fix push sid & uid bug
- remove glide & use govendor


## [V1.4.0] 2017-12-5

### [Add]
- add ws ping pong protocal

## [v1.3.3] 2017-11-01

### Fixed
- fix README.md
- fix CHANGELOG.md
- fix logrus

## [v1.3.0] 2017-05-04

### Removed
- 把env.example 移除

### Added
- example env得部分，拆成 [slave.env.example](./slave.env.example), [master.env.example](./master.env.example) 以避免誤解都要合在一起
- 新增4個env變數 [GUSHER_REDIS_DBNO](./master.env.example#L3), [GUSHER_JOB_REDIS_ADDR](./slave.env.example#L8), [GUSHER_JOB_REDIS_DBNO](./slave.env.example#L9), [GUSHER_JOB_REDIS_MAX_IDLE](./slave.env.example#L10), [GUSHER_JOB_REDIS_MAX_CONN](./slave.env.example#L11)
