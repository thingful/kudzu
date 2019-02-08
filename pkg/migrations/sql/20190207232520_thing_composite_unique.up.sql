ALTER TABLE things
  ADD CONSTRAINT location_identifier_serial_num_key
  UNIQUE (location_identifier, serial_num);