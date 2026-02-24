-- Restore unique constraint to refresh_tokens table
ALTER TABLE auth.refresh_tokens ADD CONSTRAINT refresh_tokens_user_id_device_info_key UNIQUE (user_id, device_info);