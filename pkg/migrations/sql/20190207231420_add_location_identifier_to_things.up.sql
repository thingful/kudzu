ALTER TABLE things
  ADD COLUMN location_identifier TEXT UNIQUE;

UPDATE things
  SET location_identifier = split_part(resource_url, '/', 8);

ALTER TABLE things
  ALTER COLUMN location_identifier SET NOT NULL;
