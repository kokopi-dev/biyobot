-- Create "notifications" table
CREATE TABLE `notifications` (
  `id` varchar NULL,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `service` varchar NULL,
  `metadata` text NOT NULL,
  `notify_at` datetime NULL,
  `title` varchar NOT NULL,
  `message` varchar NOT NULL,
  PRIMARY KEY (`id`)
);
-- Create index "idx_notifications_deleted_at" to table: "notifications"
CREATE INDEX `idx_notifications_deleted_at` ON `notifications` (`deleted_at`);
