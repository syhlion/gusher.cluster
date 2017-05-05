## [1.3.0] 2017-05-04
### Removed
- 把env.example 移除
### Add
- example env得部分，拆成 [slave.env.example](./slave.env.example), [master.env.example](./master.env.example) 以避免誤解都要合在一起
- 新增4個env變數 [GUSHER_REDIS_DBNO](./master.env.example#L3), [GUSHER_JOB_REDIS_ADDR](./slave.env.example#L8), [GUSHER_JOB_REDIS_DBNO](./slave.env.example#L9), [GUSHER_JOB_REDIS_MAX_IDLE](./slave.env.example#L10), [GUSHER_JOB_REDIS_MAX_CONN](./slave.env.example#L11)
