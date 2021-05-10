/* -------------------- TABLE SCHEMA --------------------- */
-- MySQL
CREATE TABLE `tasks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `task_key` varchar(64) NOT NULL DEFAULT '',
  `task_status` varchar(64) NOT NULL DEFAULT '',
  `context` mediumtext,
  `argument` mediumtext,
  `extra` mediumtext,
  `created_at` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
  `updated_at` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
  PRIMARY KEY (`id`),
  KEY `idx_task_key` (`task_key`),
  KEY `idx_task_status` (`task_status`),
  KEY `idx_updated_at` (`updated_at`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4;