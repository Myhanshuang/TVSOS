-- MySQL dump 10.13  Distrib 8.4.5, for macos15 (arm64)
--
-- Host: localhost    Database: mover
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
-- Current Database: `mover`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `mover` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `mover`;

--
-- Table structure for table `cargo`
--

DROP TABLE IF EXISTS `cargo`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `cargo` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(64) DEFAULT NULL COMMENT '货物名称',
  `tybe` int DEFAULT NULL COMMENT '货物类型-数字代号',
  `pack` int DEFAULT NULL COMMENT '包装类型-数字代号-例如:一箱苹果,一捆甘蔗',
  `weight` int DEFAULT NULL COMMENT '单位包装下的重量(kg)-例如:一箱苹果的重量',
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
-- Table structure for table `order_task`
--

DROP TABLE IF EXISTS `order_task`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `order_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `shipment_id` bigint unsigned DEFAULT NULL COMMENT '订单id',
  `vehicle_id` bigint unsigned DEFAULT NULL COMMENT '运输车辆id',
  `sequential` int DEFAULT NULL COMMENT '任务阶段-数字代号',
  `create_time` datetime DEFAULT NULL COMMENT '任务创建时间',
  `update_time` datetime DEFAULT NULL COMMENT '任务更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `order_task`
--

LOCK TABLES `order_task` WRITE;
/*!40000 ALTER TABLE `order_task` DISABLE KEYS */;
/*!40000 ALTER TABLE `order_task` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `poi`
--

DROP TABLE IF EXISTS `poi`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `poi` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `name` varchar(32) DEFAULT NULL COMMENT '名称',
  `tybe` int DEFAULT NULL COMMENT '类型-数字代号',
  `lon` double DEFAULT NULL COMMENT '经度',
  `lat` double DEFAULT NULL COMMENT '纬度',
  `status` int DEFAULT NULL COMMENT '状态-数字代号',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `poi`
--

LOCK TABLES `poi` WRITE;
/*!40000 ALTER TABLE `poi` DISABLE KEYS */;
INSERT INTO `poi` VALUES (1,'中国石油中和加油站',1,104.09,30.56,1),(2,'长城加油站',1,104.03,30.58,2),(3,'卡乐洗鞋智能中央广场',2,103.96,30.48,1);
/*!40000 ALTER TABLE `poi` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `shipment`
--

DROP TABLE IF EXISTS `shipment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `shipment` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `start_poi_id` bigint unsigned DEFAULT NULL COMMENT '起始poi点的id',
  `end_poi_id` bigint unsigned DEFAULT NULL COMMENT '终点poi点的id',
  `create_time` datetime DEFAULT NULL COMMENT '订单创建时间',
  `update_time` datetime DEFAULT NULL COMMENT '订单更新时间',
  `status` int DEFAULT NULL COMMENT '订单状态-数字代号',
  `cargo_id` bigint unsigned DEFAULT NULL COMMENT '运输货物的id',
  `count` int DEFAULT NULL COMMENT '货物数量-重量计算公式:货物数量 * 单位包装下的重量',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `shipment`
--

LOCK TABLES `shipment` WRITE;
/*!40000 ALTER TABLE `shipment` DISABLE KEYS */;
/*!40000 ALTER TABLE `shipment` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `vehicle`
--

DROP TABLE IF EXISTS `vehicle`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `vehicle` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'id',
  `license` varchar(64) NOT NULL DEFAULT '' COMMENT '车牌号',
  `lon` double DEFAULT NULL COMMENT '经度',
  `lat` double DEFAULT NULL COMMENT '纬度',
  `speed` double DEFAULT NULL COMMENT '实时速度 单位：m/s',
  `update_time` datetime DEFAULT NULL COMMENT '更新时间',
  `status` int DEFAULT NULL COMMENT '状态-数字代号',
  `tybe` int DEFAULT NULL COMMENT '类型-数字代号',
  `size` int DEFAULT NULL COMMENT '载货量',
  `reserved_size` int NOT NULL DEFAULT '0' COMMENT '预占载重(kg)',
  `capacity` int DEFAULT NULL COMMENT '总载货容量',
  `length` double NOT NULL DEFAULT '0' COMMENT '车厢长度(米)',
  `width` double NOT NULL DEFAULT '0' COMMENT '车厢宽度(米)',
  `height` double NOT NULL DEFAULT '0' COMMENT '车厢高度(米)',
  `wait_time` double NOT NULL DEFAULT '0' COMMENT '当前等待时长(分钟)',
  `total_wait_time` double NOT NULL DEFAULT '0' COMMENT '累计等待时长(分钟)',
  `empty_mileage` double NOT NULL DEFAULT '0' COMMENT '空驶里程(千米)',
  `distance` double NOT NULL DEFAULT '0' COMMENT '运输里程(千米)',
  `duration` double NOT NULL DEFAULT '0' COMMENT '运输时长(小时)',
  `angle` double NOT NULL DEFAULT '90' COMMENT '车辆角度(度)',
  PRIMARY KEY (`id`)
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

-- Dump completed on 2026-03-24 16:06:51
