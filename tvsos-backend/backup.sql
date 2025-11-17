-- MySQL dump 10.13  Distrib 8.4.5, for macos15 (arm64)
--
-- Host: localhost    Database: routing
-- ------------------------------------------------------
-- Server version	8.4.5

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `assign`
--

DROP TABLE IF EXISTS `assign`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `assign` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `vehicle_id` bigint NOT NULL COMMENT '车辆id',
  `driver_id` bigint NOT NULL COMMENT '司机id',
  `transport_order_id` bigint NOT NULL COMMENT '订单id',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `assign`
--

LOCK TABLES `assign` WRITE;
/*!40000 ALTER TABLE `assign` DISABLE KEYS */;
/*!40000 ALTER TABLE `assign` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cargo`
--

DROP TABLE IF EXISTS `cargo`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cargo` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(255) DEFAULT NULL COMMENT '货物名称',
  `type` int DEFAULT NULL COMMENT '打包类型 1:一箱 2:一捆 3:一袋 4:一桶',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cargo`
--

LOCK TABLES `cargo` WRITE;
/*!40000 ALTER TABLE `cargo` DISABLE KEYS */;
/*!40000 ALTER TABLE `cargo` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `category`
--

DROP TABLE IF EXISTS `category`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `category` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `weight` double DEFAULT NULL COMMENT '车辆自重 单位：kg',
  `brand` varchar(32) DEFAULT NULL COMMENT '品牌',
  `length` double DEFAULT NULL COMMENT '长度 单位：米',
  `width` double DEFAULT NULL COMMENT '宽度 单位：米',
  `height` double DEFAULT NULL COMMENT '高度 单位：米',
  `capacity` double DEFAULT NULL COMMENT '最大承载量 单位：kg',
  `type` int DEFAULT NULL COMMENT '车辆类型 1平板车 2高护栏 3全封闭',
  `scope` int DEFAULT NULL COMMENT '使用范围 1普通运输车辆 2恒温或保温车辆 3危险品车辆 4特种车 5监管车',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `category`
--

LOCK TABLES `category` WRITE;
/*!40000 ALTER TABLE `category` DISABLE KEYS */;
/*!40000 ALTER TABLE `category` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `driver`
--

DROP TABLE IF EXISTS `driver`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `driver` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(32) DEFAULT NULL COMMENT '司机姓名',
  `status` int DEFAULT NULL COMMENT '状态 1可用 2任务中 3休假中',
  `type` int DEFAULT NULL COMMENT '驾照类型 1:A1 2:A2 3:A3 4:B1 5:B2 6:C1',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `driver`
--

LOCK TABLES `driver` WRITE;
/*!40000 ALTER TABLE `driver` DISABLE KEYS */;
/*!40000 ALTER TABLE `driver` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `order_detail`
--

DROP TABLE IF EXISTS `order_detail`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `order_detail` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `transport_order_id` bigint NOT NULL COMMENT '订单id',
  `cargo_id` bigint NOT NULL COMMENT '货物id',
  `quantity` int DEFAULT NULL COMMENT '货物数量',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `order_detail`
--

LOCK TABLES `order_detail` WRITE;
/*!40000 ALTER TABLE `order_detail` DISABLE KEYS */;
/*!40000 ALTER TABLE `order_detail` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `order_route`
--

DROP TABLE IF EXISTS `order_route`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `order_route` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `transport_order_id` bigint NOT NULL COMMENT '订单id',
  `route_id` bigint NOT NULL COMMENT '基础路线id',
  `sequence` int DEFAULT NULL COMMENT '顺序号，从1开始',
  `status` int DEFAULT NULL COMMENT '运输状态 1:未开始 2:运输中 3:已完成 4:异常',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `order_route`
--

LOCK TABLES `order_route` WRITE;
/*!40000 ALTER TABLE `order_route` DISABLE KEYS */;
/*!40000 ALTER TABLE `order_route` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `poi`
--

DROP TABLE IF EXISTS `poi`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `poi` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(255) DEFAULT NULL COMMENT 'poi名称',
  `lon` double DEFAULT NULL COMMENT '经度',
  `lat` double DEFAULT NULL COMMENT '纬度',
  `type` int DEFAULT NULL COMMENT '类型 1:加油站 2:保养点 3:其他',
  `status` int DEFAULT NULL COMMENT '1:工作中 2:歇业中',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `poi`
--

LOCK TABLES `poi` WRITE;
/*!40000 ALTER TABLE `poi` DISABLE KEYS */;
INSERT INTO `poi` VALUES (1,'中国石油中和加油站',104.091008,30.561087,1,1),(2,'延长壳牌盛锦三街加油站',104.0354,30.563117,1,1),(3,'成都通能盛锦CNG加气站',104.035513,30.56348,1,1),(4,'中国石油麻柳湾加油站',104.087907,30.596824,1,1),(5,'中国石油元华加油站',104.050117,30.601513,1,1),(6,'延长壳牌加油站(成都高新益新大道站)',104.033537,30.587778,1,1),(7,'延长壳牌拓新西二街加油站',104.053281,30.542733,1,1),(8,'中国石油油料中和加油站(红星路南延线)',104.088515,30.547206,1,1),(9,'中国石化加油站示例',104.06,30.55,1,1);
/*!40000 ALTER TABLE `poi` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `route`
--

DROP TABLE IF EXISTS `route`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `route` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `begin_poi_id` bigint NOT NULL COMMENT '起点poi的id',
  `end_poi_id` bigint NOT NULL COMMENT '终点poi的id',
  `distance` double DEFAULT NULL COMMENT '距离 单位:km',
  `type` int DEFAULT NULL COMMENT '路线类型 1:公路 2:铁路 3:多试',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `route`
--

LOCK TABLES `route` WRITE;
/*!40000 ALTER TABLE `route` DISABLE KEYS */;
/*!40000 ALTER TABLE `route` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `transport_order`
--

DROP TABLE IF EXISTS `transport_order`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `transport_order` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `order_number` varchar(64) DEFAULT NULL COMMENT '订单编号',
  `begin_poi_id` bigint NOT NULL COMMENT '起点poi的id',
  `end_poi_id` bigint NOT NULL COMMENT '终点poi的id',
  `est_begin_time` date DEFAULT NULL COMMENT '预计发车时间',
  `est_end_time` date DEFAULT NULL COMMENT '预计到达时间',
  `act_begin_time` date DEFAULT NULL COMMENT '实际发车时间',
  `act_end_time` date DEFAULT NULL COMMENT '实际到达时间',
  `create_time` datetime DEFAULT NULL COMMENT '订单创建时间',
  `status` int DEFAULT NULL COMMENT '订单状态 1:待接单 2:已接单 3:运输中 4:已送达 5:已取消 6:异常',
  PRIMARY KEY (`id`),
  UNIQUE KEY `order_number` (`order_number`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `transport_order`
--

LOCK TABLES `transport_order` WRITE;
/*!40000 ALTER TABLE `transport_order` DISABLE KEYS */;
/*!40000 ALTER TABLE `transport_order` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `vehicle`
--

DROP TABLE IF EXISTS `vehicle`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vehicle` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `license` varchar(32) DEFAULT NULL COMMENT '车牌号',
  `status` int DEFAULT NULL COMMENT '车辆状态 1可用 2运输中 3卸货中 4保养中',
  `lon` double DEFAULT NULL COMMENT '经度',
  `lat` double DEFAULT NULL COMMENT '纬度',
  `speed` double DEFAULT NULL COMMENT '速度 单位：km/h',
  `create_time` datetime DEFAULT NULL COMMENT '新建时间',
  `update_time` datetime DEFAULT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `license` (`license`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `vehicle`
--

LOCK TABLES `vehicle` WRITE;
/*!40000 ALTER TABLE `vehicle` DISABLE KEYS */;
/*!40000 ALTER TABLE `vehicle` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-09-30  9:28:19
