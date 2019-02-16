package postgres

import (
	"context"
	"crypto/rand"
	"database/sql/driver"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/logger"
)

// ScopeClaim is a custom type used to represent existing scope levels
type ScopeClaim string

// ScopeClaims is a custom type which implements valuer for persisting to the DB
type ScopeClaims []ScopeClaim

// Value is our implementation of the driver valuer interface which takes a
// scopeclaims instance and converts into some value that can be persisted to
// the DB.
func (s ScopeClaims) Value() (driver.Value, error) {
	var b strings.Builder

	b.WriteString("{")

	for i, c := range s {
		b.WriteString(string(c))
		if i < len(s)-1 {
			b.WriteString(",")
		}
	}

	b.WriteString("}")

	return b.String(), nil
}

// Scan is our implementation of the sql.Scanner interface for reading a column
// from the database and from that creating a new instance of the type.
func (s *ScopeClaims) Scan(src interface{}) error {
	strSlice, ok := src.([]byte)
	if !ok {
		return errors.New("ScopeClaims scan source was not a []string")
	}

	parsed := (string(strSlice[1 : len(strSlice)-1]))
	exploded := strings.Split(parsed, ",")

	out := ScopeClaims{}
	for _, claim := range exploded {
		out = append(out, ScopeClaim(claim))
	}

	*s = out

	return nil
}

const (
	// CreateUserScope is used for clients allowed to create new users
	CreateUserScope = ScopeClaim("create-users")

	// GetMetadataScope is used for clients that can query metadata
	GetMetadataScope = ScopeClaim("metadata")

	// GetTimeSeriesDataScope is used for clients that can query time series data
	GetTimeSeriesDataScope = ScopeClaim("timeseries")

	// encodeCrockford is a list of characters for generating crockford style base 32
	encodeCrockford = "0123456789abcdefghjkmnpqrstvwxyz"

	// keyLength is the length of the hash in bytes we generate
	keyLength = 16
)

var (
	// allowedScopeClaims is a map of permitted scopes
	allowedScopeClaims = map[ScopeClaim]string{
		CreateUserScope:        "Can create new users",
		GetMetadataScope:       "Can query metadata",
		GetTimeSeriesDataScope: "Can query time series data",
	}

	// crockfordEncoding is our base32 encoding that uses our custom string
	crockfordEncoding = base32.NewEncoding(encodeCrockford)
)

// App is our type used for reading auth information back from the DB
type App struct {
	UID   string      `db:"uid"`
	Name  string      `db:"app_name"`
	Hash  string      `db:"key_hash"`
	Roles ScopeClaims `db:"scope"`
	Key   string
}

// CreateApp attempts to create and store an app record into the DB. We generate
// a random UID and a random hash which is hashed and stored to the DB
func (d *DB) CreateApp(ctx context.Context, name string, claims []string) (*App, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "creating app",
			"name", name,
			"claims", claims,
		)
	}

	scope := buildScopeClaims(claims)

	if !areKnownClaims(scope) {
		return nil, errors.New("invalid scope claims")
	}

	b, err := randomBytes(keyLength)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get random bytes when creating app")
	}

	uid, err := randomUID(10)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create random UID when creating app")
	}

	app := &App{
		UID:   uid,
		Name:  name,
		Hash:  fmt.Sprintf("%x", b),
		Roles: scope,
		Key:   fmt.Sprintf("%s-%x", uid, b),
	}

	tx, err := d.DB.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transaction to save app")
	}

	sql := `INSERT INTO applications
		(uid, app_name, key_hash, scope)
	VALUES (:uid, :app_name, crypt(:key_hash, gen_salt('bf', 5)), :scope)`

	sql, args, err := tx.BindNamed(sql, app)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to bind named query")
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(err, "failed to execute query")
	}

	return app, tx.Commit()
}

// LoadApp attempts to load an app on being given the key. We compare against
// the hashed version in the DB.
func (d *DB) LoadApp(ctx context.Context, key string) (*App, error) {
	log := logger.FromContext(ctx)

	if d.verbose {
		log.Log(
			"msg", "loading app",
			"key", key,
		)
	}

	parts := strings.Split(key, "-")
	if len(parts) != 2 {
		return nil, errors.New("invalid key")
	}

	sql := `SELECT uid, app_name, scope FROM applications WHERE uid = $1 AND key_hash = crypt($2, key_hash)`

	var app App

	err := d.DB.Get(&app, sql, parts[0], parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app from DB")
	}

	return &app, nil
}

func buildScopeClaims(claims []string) ScopeClaims {
	scope := ScopeClaims{}

	for _, c := range claims {
		scope = append(scope, ScopeClaim(c))
	}

	return scope
}

// checks the passed in claim set is one of our known values
func areKnownClaims(scope ScopeClaims) bool {
	for _, claim := range scope {
		if _, ok := allowedScopeClaims[claim]; !ok {
			return false
		}
	}

	return true
}

// randomBytes returns n random bytes read from crypto/rand or an error
func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// randomUID returns a crockford base32 encoded random string of length n
func randomUID(n int) (string, error) {
	bytes, err := randomBytes(n)
	if err != nil {
		return "", err
	}

	str := crockfordEncoding.EncodeToString(bytes)

	return str[:n], nil
}
