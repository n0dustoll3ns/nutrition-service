-- Remove unique constraint from refresh_tokens table to allow multiple refresh tokens per device
ALTER TABLE auth.refresh_tokens DROP CONSTRAINT IF EXISTS refresh_tokens_user_id_device_info_key;