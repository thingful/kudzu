CREATE TABLE IF NOT EXISTS location_changes (
  id SERIAL PRIMARY KEY,
  thing_id INTEGER NOT NULL REFERENCES things(id) ON DELETE CASCADE,
  inserted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  previous_long DOUBLE PRECISION,
  previous_lat DOUBLE PRECISION,
  new_long DOUBLE PRECISION NOT NULL,
  new_lat DOUBLE PRECISION NOT NULL
);

CREATE OR REPLACE FUNCTION thing_location_audit() RETURNS TRIGGER AS $body$
BEGIN
  IF (TG_OP = 'UPDATE') THEN
    IF (NEW.long IS DISTINCT FROM OLD.long OR NEW.lat IS DISTINCT FROM OLD.lat) THEN
      INSERT INTO location_changes (thing_id, previous_long, previous_lat, new_long, new_lat)
      VALUES (NEW.id, OLD.long, OLD.lat, NEW.long, NEW.lat);
    END IF;
  ELSE
    INSERT INTO location_changes (thing_id, new_long, new_lat)
    VALUES (NEW.id, NEW.long, NEW.lat);
  END IF;
  RETURN NULL;
END;
$body$ LANGUAGE plpgsql;

CREATE TRIGGER thing_location_audit
AFTER INSERT OR UPDATE ON things
  FOR EACH ROW
  EXECUTE PROCEDURE thing_location_audit();