ALTER TABLE things
  DROP COLUMN resource_url,
  DROP COLUMN data_url;

ALTER TABLE things
  ALTER COLUMN owner_id SET NOT NULL;
