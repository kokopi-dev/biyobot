-- Create "discord_messages" table
CREATE TABLE `discord_messages` (
  `id` varchar NULL,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime NULL,
  `action` varchar NULL,
  `channel_id` varchar NULL,
  `user_id` varchar NULL,
  `message_id` varchar NULL,
  `content` text NOT NULL,
  `execute_action_on` datetime NULL,
  PRIMARY KEY (`id`)
);
-- Create index "idx_discord_messages_deleted_at" to table: "discord_messages"
CREATE INDEX `idx_discord_messages_deleted_at` ON `discord_messages` (`deleted_at`);
