ALTER TABLE things
  ALTER COLUMN serial_num SET NOT NULL;

ALTER TABLE things ADD CONSTRAINT
  things_serial_num_key UNIQUE (serial_num);