create database if not exists routing;
use routing;
create table assign
(
    id                 bigint unsigned auto_increment comment 'id'
        primary key,
    vehicle_id         bigint not null comment '车辆id',
    driver_id          bigint not null comment '司机id',
    transport_order_id bigint not null comment '订单id'
);

create table cargo
(
    id   bigint unsigned auto_increment comment 'id'
        primary key,
    name varchar(255) null comment '货物名称',
    type int          null comment '打包类型 1:一箱 2:一捆 3:一袋 4:一桶'
);

create table category
(
    id       bigint unsigned auto_increment comment 'id'
        primary key,
    weight   double      null comment '车辆自重 单位：kg',
    brand    varchar(32) null comment '品牌',
    length   double      null comment '长度 单位：米',
    width    double      null comment '宽度 单位：米',
    height   double      null comment '高度 单位：米',
    capacity double      null comment '最大承载量 单位：kg',
    type     int         null comment '车辆类型 1平板车 2高护栏 3全封闭',
    scope    int         null comment '使用范围 1普通运输车辆 2恒温或保温车辆 3危险品车辆 4特种车 5监管车'
);

create table driver
(
    id     bigint unsigned auto_increment comment 'id'
        primary key,
    name   varchar(32) null comment '司机姓名',
    status int         null comment '状态 1可用 2任务中 3休假中',
    type   int         null comment '驾照类型 1:A1 2:A2 3:A3 4:B1 5:B2 6:C1'
);

create table order_detail
(
    id                 bigint unsigned auto_increment comment 'id'
        primary key,
    transport_order_id bigint not null comment '订单id',
    cargo_id           bigint not null comment '货物id',
    quantity           int    null comment '货物数量'
);

create table order_route
(
    id                 bigint unsigned auto_increment comment 'id'
        primary key,
    transport_order_id bigint not null comment '订单id',
    route_id           bigint not null comment '基础路线id',
    sequence           int    null comment '顺序号，从1开始',
    status             int    null comment '运输状态 1:未开始 2:运输中 3:已完成 4:异常'
);

create table poi
(
    id     bigint unsigned auto_increment comment 'id'
        primary key,
    name   varchar(255) null comment 'poi名称',
    lon    double       null comment '经度',
    lat    double       null comment '纬度',
    type   int          null comment '类型 1:加油站 2:保养点',
    status int          null comment '1:工作中 2:歇业中'
);

create table route
(
    id           bigint unsigned auto_increment comment 'id'
        primary key,
    begin_poi_id bigint not null comment '起点poi的id',
    end_poi_id   bigint not null comment '终点poi的id',
    distance     double null comment '距离 单位:km',
    type         int    null comment '路线类型 1:公路 2:铁路 3:多试'
);

create table transport_order
(
    id             bigint unsigned auto_increment comment 'id'
        primary key,
    order_number   varchar(64) null comment '订单编号',
    begin_poi_id   bigint      not null comment '起点poi的id',
    end_poi_id     bigint      not null comment '终点poi的id',
    est_begin_time date        null comment '预计发车时间',
    est_end_time   date        null comment '预计到达时间',
    act_begin_time date        null comment '实际发车时间',
    act_end_time   date        null comment '实际到达时间',
    create_time    datetime    null comment '订单创建时间',
    status         int         null comment '订单状态 1:待接单 2:已接单 3:运输中 4:已送达 5:已取消 6:异常',
    constraint order_number
        unique (order_number)
);

create table vehicle
(
    id      bigint unsigned auto_increment comment 'id'
        primary key,
    license varchar(32) null comment '车牌号',
    status  int         null comment '车辆状态 1可用 2运输中 3卸货中 4保养中',
    lon     double      null comment '经度',
    lat     double      null comment '纬度',
    speed   double      null comment '速度 单位：km/h',
    constraint license
        unique (license)
);

