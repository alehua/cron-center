create database if not exists `cron`;

use cron;

create table if not EXISTS `task_info`
(
    id                int auto_increment
    primary key,
    name              varchar(128)  not null comment '任务名称',
    scheduler_status  int           not null comment '任务调度状态',
    cron              varchar(32)   not null comment '定时触发cron配置',
    type              varchar(32)   not null comment '任务类型',
    config            text          not null comment '执行配置',
    version           int default 0 not null comment '任务调度版本',
    instance_id       varchar(32)   not null comment '实例ID',
    max_exec_time     int           not null comment '任务最大运行时间/秒',
    next_time         bigint        not null comment '下次任务执行时间',
    create_time       bigint        not null,
    update_time       bigint        not null
    )
    comment '任务详情';

create index task_info_update_time_index on task_info (update_time);

create table if not EXISTS `task_execution_record`
(
    id             int auto_increment
    primary key,
    task_id        int         not null comment '任务id',
    execute_status varchar(32) not null comment '任务类型',
    create_time    bigint      not null,
    update_time    bigint      not null
    )
    comment '任务执行记录';

create index index_task_id on task_execution_record (id, task_id);

-- 插入数据
INSERT INTO `task_info` (
    `name`,
    `scheduler_status`,
    `cron`,
    `type`,
    `config`,
    `version`,
    `instance_id`,
    `max_exec_time`,
    `next_time`,
    `create_time`,
    `update_time`
) VALUES (
             'demo',
             'RUNNING',
             '0 0 * * * ?',
             'DAILY',
             '{"param1": "value1", "param2": "value2"}',
             1,
             'instance_12345',
             10,
             1626796800000,
             UNIX_TIMESTAMP(),
             UNIX_TIMESTAMP()
         );
