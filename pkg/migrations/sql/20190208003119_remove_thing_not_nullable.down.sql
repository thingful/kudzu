ALTER TABLE things
  ADD COLUMN resource_url TEXT,
  ADD COLUMN data_url VARCHAR;

UPDATE things SET
  resource_url = 'https://api-flower-power-pot.parrot.com/sensor_data/v6/sample/location/' || location_identifier;

ALTER TABLE things
  ALTER COLUMN resource_url SET NOT NULL;

ALTER TABLE things
  ALTER COLUMN owner_id DROP NOT NULL;