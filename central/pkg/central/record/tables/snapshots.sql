CREATE TABLE IF NOT EXISTS `snapshots` (
  `id` int(64) NOT NULL auto_increment,
  `sum` varchar(512) NOT NULL,
  `url` varchar(256) NOT NULL,
  `time` datetime NULL,
  `origin` varchar(32) NOT NULL,
   PRIMARY KEY  (`id`)
);