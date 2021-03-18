CREATE TABLE IF NOT EXISTS `mirrors` (
  `id` int(64) NOT NULL auto_increment,
  `peerID` int(64),
  `snapshotID` int(64),
   PRIMARY KEY  (`id`),
   FOREIGN KEY (`peerID`) REFERENCES `peers`(`id`),
   FOREIGN KEY (`snapshotID`) REFERENCES `snapshots`(`id`)
);