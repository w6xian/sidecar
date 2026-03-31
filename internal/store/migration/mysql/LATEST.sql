CREATE TABLE `mi_sidecar_tokens` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `app_id` varchar(128) NOT NULL DEFAULT '',
  `app_sec` varchar(128) NOT NULL DEFAULT '',
  `app_sn` varchar(256) NOT NULL DEFAULT '',
  `name` varchar(256) NOT NULL DEFAULT '',
  `expire_time` int(11) NOT NULL DEFAULT '0',
  `lock_status` tinyint(4) NOT NULL DEFAULT '0',
  `lock_time` int(11) NOT NULL DEFAULT '1',
  `status` tinyint(4) NOT NULL DEFAULT '1',
  `intime` int(11) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_admin` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='token管理';
