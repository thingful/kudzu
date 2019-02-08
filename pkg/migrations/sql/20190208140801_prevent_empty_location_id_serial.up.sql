ALTER TABLE things
  ADD CONSTRAINT location_identifier_check CHECK (location_identifier <> ''),
  ADD CONSTRAINT serial_num_check CHECK (serial_num <> '');