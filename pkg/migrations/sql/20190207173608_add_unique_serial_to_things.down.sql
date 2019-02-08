ALTER TABLE things DROP CONSTRAINT
  things_serial_num_key;

ALTER TABLE things
  ALTER COLUMN serial_num DROP NOT NULL;