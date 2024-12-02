package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	// libray has to be imported to register the driver.

	_ "github.com/mattn/go-sqlite3"
)

const (
	db_folder    = "db"
	tb_playlists = "playlists"
	tb_wol       = "wol"
	tb_cec       = "cec"
)

var (
	ErrParamEmpty   = fmt.Errorf("parameter is empty")
	ErrInvalidID    = fmt.Errorf("invalid ID")
	ErrRecordExists = fmt.Errorf("record exists")
)

// SqliteDB is a wrapper around the sqlite3 database. It also holds the db filename and context.
type SqliteDB struct {
	Name string // DB file name.
	*sql.DB

	ctx    context.Context
	cancel func()
}

func createDBFolder() error {
	_, err := os.Stat(db_folder)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("createDBFolder: %w", err)
	}

	return os.Mkdir(db_folder, 0o755)
}

// NewSqliteDB creates a new Sqlite3 DB instance.
func NewSqliteDB(filename string) (*SqliteDB, error) {
	if filename == "" {
		return nil, fmt.Errorf("NewSqliteDB: filename is empty")
	}

	if err := createDBFolder(); err != nil {
		return nil, fmt.Errorf("NewSqliteDB: %w", err)
	}

	db := &SqliteDB{Name: filename}
	db.ctx, db.cancel = context.WithCancel(context.Background())
	return db, nil
}

// Open opens the database with WAL and foreign_keys enabled. Open then pings the database. If the
// database does not exist, it will be created.
func (db *SqliteDB) Open() error {
	if db.Name == "" {
		return fmt.Errorf("SqliteDB.Open: no database name provided")
	}

	var err error
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=ON", db_folder+"/"+db.Name)
	db.DB, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("SqliteDB.Open: %w", err)
	}

	err = db.DB.Ping()
	if err != nil {
		return fmt.Errorf("SqliteDB.Open: failed to ping db: %w", err)
	}

	// TODO: Implement zstd compression. https://phiresky.github.io/blog/2022/sqlite-zstd/

	return nil
}

// CuttleMigrate runs the migrations for each table in the main cuttle database.
func (db *SqliteDB) Migrate() error {
	if err := db.PlaylistsMigrate(); err != nil {
		return fmt.Errorf("SqliteDB.Migrate: failed to migrate %s: %w", tb_playlists, err)
	}

	if err := db.CECMigrate(); err != nil {
		return fmt.Errorf("SqliteDB.Migrate: failed to migrate %s: %w", tb_cec, err)
	}

	if err := db.WOLMigrate(); err != nil {
		return fmt.Errorf("SqliteDB.Migrate: failed to migrate %s: %w", tb_wol, err)
	}

	return nil
}

// IsUnique returns nil if no records exist in the table that match the where clause. If a record
// exists, it returns an ErrRecordExists error. The 'where' clause should not include the "WHERE"
// keyword but may include multiple column queries which are comma separated: 'col1 = ?, col2 = ?'.
// The 'args' are the values to be used in the where clause.
//
// Example: db.IsUnique("users", "username = ?", "myusername")
func (db *SqliteDB) IsUnique(table string, where string, args ...any) error {
	if table == "" {
		return fmt.Errorf("SqliteDB.IsUnique: table is empty")
	}

	if where == "" {
		return fmt.Errorf("SqliteDB.IsUnique: where is empty")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s;", table, where)
	row, err := db.QueryRow(query, args...)
	if err != nil {
		return fmt.Errorf("SqliteDB.IsUnique: %s", err)
	}

	var count int
	err = row.Scan(&count)
	if err != nil {
		return fmt.Errorf("SqliteDB.IsUnique: %s", err)
	}

	if count > 0 {
		return fmt.Errorf("SqliteDB.IsUnique: %w", ErrRecordExists)
	}

	return nil
}

// IsErrNotUnique checks if the error is due to a unique constraint violation.
func IsErrNotUnique(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed:")
}

// Close cancels the context and closes the database connection.
func (db *SqliteDB) Close() error {
	if db.DB == nil {
		return nil
	}

	db.cancel()
	return db.DB.Close()
}

func (db *SqliteDB) QueryRow(query string, args ...any) (*sql.Row, error) {
	if db.DB == nil {
		return &sql.Row{}, fmt.Errorf("SqliteDB.QueryRow: Sqlite.DB.DB is nil")
	}

	return db.DB.QueryRowContext(db.ctx, query, args...), nil
}

func (db *SqliteDB) Query(query string, args ...any) (*sql.Rows, error) {
	if db.DB == nil {
		return &sql.Rows{}, fmt.Errorf("SqliteDB.Query: Sqlite.DB.DB is nil")
	}

	return db.DB.QueryContext(db.ctx, query, args...)
}

func (db *SqliteDB) Exec(query string, args ...any) (sql.Result, error) {
	if db.DB == nil {
		return nil, fmt.Errorf("SqliteDB.Exec: Sqlite.DB.DB is nil")
	}

	return db.DB.ExecContext(db.ctx, query, args...)
}

// ############################################################################################## //
// ####################################       Playlists      #################################### //
// ############################################################################################## //

// PlaylistsMigrate creates the 'playlists' table if it does not exist.
func (db *SqliteDB) PlaylistsMigrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS ` + tb_playlists + ` (
		id VARCHAR(11) NOT NULL PRIMARY KEY,
		name VARCHAR(32) NOT NULL,
		playlist TEXT NOT NULL DEFAULT '[]'
	);
	CREATE INDEX IF NOT EXISTS idx_playlists_id ON ` + tb_playlists + ` (id);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("SqliteDB.PlaylistsMigrate: %w", err)
	}

	return nil
}

// PlaylistIsUnique checks if the playlist id is unique in the database. If the id is not unique, it
// returns an ErrRecordExists error.
func (db *SqliteDB) PlaylistIsUnique(id string) error {
	if id == "" {
		return fmt.Errorf("SqliteDB.PlaylistIsUnique: id - %w", ErrParamEmpty)
	}

	err := db.IsUnique(tb_playlists, "id = ?", id)
	if errors.Is(err, ErrRecordExists) {
		return fmt.Errorf("SqliteDB.PlaylistIsUnique: %w", ErrRecordExists)
	}

	return err
}

// PlaylistCreate creates a new user in the database and returns the new user data. A password should
// never be provided in plain text. PlaylistCreate will check for hash formatting.
func (db *SqliteDB) PlaylistCreate(pbc PlaybackClient, playlist Playlist) error {
	if pbc.ID == "" {
		return fmt.Errorf("SqliteDB.PlaylistCreate: PlaybackClient.ID - %w", ErrParamEmpty)
	}

	if pbc.Name == "" {
		return fmt.Errorf("SqliteDB.PlaylistCreate: PlaybackClient.Name - %w", ErrParamEmpty)
	}

	plData, err := json.Marshal(playlist)
	if err != nil {
		return fmt.Errorf("SqliteDB.PlaylistCreate: %w", err)
	}

	query := `INSERT INTO ` + tb_playlists + ` (id, name, playlist) VALUES (?, ?, ?)`
	r, err := db.Exec(query, pbc.ID, pbc.Name, plData)
	if err != nil {
		if IsErrNotUnique(err) {
			return fmt.Errorf("SqliteDB.PlaylistCreate: %w", ErrRecordExists)
		}

		return fmt.Errorf("SqliteDB.PlaylistCreate: %w", err)
	}

	_, err = r.LastInsertId()
	if err != nil {
		return fmt.Errorf("SqliteDB.PlaylistCreate: %w", err)
	}

	return nil
}

// PlaylistGet retrieves a playlist from the database by ID.
func (db *SqliteDB) PlaylistGet(id string) (PlaybackClient, Playlist, error) {
	query := `SELECT * FROM ` + tb_playlists + ` WHERE id = ?`
	row, err := db.QueryRow(query, id)
	if err != nil {
		return PlaybackClient{}, Playlist{}, fmt.Errorf("SqliteDB.PlaylistGet: %w", err)
	}

	pbc := PlaybackClient{}
	var plData string

	err = row.Scan(
		&pbc.ID,
		&pbc.Name,
		&plData,
	)
	if err != nil {
		return pbc, Playlist{}, fmt.Errorf("SqliteDB.PlaylistGet: %w", err)
	}

	pl, err := UnmarshalPlaylists([]byte(plData))
	if err != nil {
		return pbc, pl, fmt.Errorf("SqliteDB.PlaylistGet: %w", err)
	}

	return pbc, pl, nil
}

// PlaylistGetByName retrieves a playlist from the database by Name.
func (db *SqliteDB) PlaylistGetByName(name string) (PlaybackClient, Playlist, error) {
	query := `SELECT * FROM ` + tb_playlists + ` WHERE name = ?`
	row, err := db.QueryRow(query, name)
	if err != nil {
		return PlaybackClient{}, Playlist{}, fmt.Errorf("SqliteDB.PlaylistGet: %w", err)
	}

	pbc := PlaybackClient{}
	var plData string

	err = row.Scan(
		&pbc.ID,
		&pbc.Name,
		&plData,
	)
	if err != nil {
		return pbc, Playlist{}, fmt.Errorf("SqliteDB.PlaylistGet: %w", err)
	}

	pl, err := UnmarshalPlaylists([]byte(plData))
	if err != nil {
		return pbc, pl, fmt.Errorf("SqliteDB.PlaylistGet: %w", err)
	}

	return pbc, pl, nil
}

func (db *SqliteDB) PlaylistGetAll() (map[PlaybackClient]Playlist, error) {
	query := `SELECT * FROM ` + tb_playlists
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("SqliteDB.PlaylistGetAll: %w", err)
	}
	defer rows.Close()

	pls := make(map[PlaybackClient]Playlist, 0)
	for rows.Next() {
		var pbc PlaybackClient
		var plData string

		err = rows.Scan(
			&pbc.ID,
			&pbc.Name,
			&plData,
		)
		if err != nil {
			return nil, fmt.Errorf("SqliteDB.PlaylistGetAll: %w", err)
		}

		pl, err := UnmarshalPlaylists([]byte(plData))
		if err != nil {
			return nil, fmt.Errorf("SqliteDB.PlaylistGetAll: %w", err)
		}

		pls[pbc] = pl
	}

	return pls, nil
}

// PlaylistUpdate updates a user in the database and returns the updated user data.
func (db *SqliteDB) PlaylistUpdate(pbc PlaybackClient, pl Playlist) error {
	if pbc.ID == "" {
		return fmt.Errorf("SqliteDB.PlaylistUpdate: %w", ErrInvalidID)
	}

	if pbc.Name == "" {
		return fmt.Errorf("SqliteDB.PlaylistUpdate: %w", ErrInvalidID)
	}

	plData, err := json.Marshal(pl)
	if err != nil {
		return fmt.Errorf("SqliteDB.PlaylistUpdate: %w", err)
	}

	query := `UPDATE ` + tb_playlists + ` SET name = ?, playlist = ? WHERE id = ?`
	if _, err := db.Exec(query, pbc.Name, plData, pbc.ID); err != nil {
		return fmt.Errorf("SqliteDB.PlaylistUpdate: %w", err)
	}

	return nil
}

// PlaylistDelete deletes a user from the database by ID.
func (db *SqliteDB) PlaylistDelete(id string) error {
	if id == "" {
		return fmt.Errorf("SqliteDB.PlaylistUpdate: %w", ErrInvalidID)
	}

	if err := db.PlaylistIsUnique(id); err != nil {
		return fmt.Errorf("SqliteDB.PlaylistDelete: %w", err)
	}

	query := `DELETE FROM ` + tb_playlists + ` WHERE id = ?`
	if _, err := db.Exec(query, id); err != nil {
		return fmt.Errorf("SqliteDB.PlaylistDelete: %w", err)
	}

	return nil
}

// PlaybackClientList retrieves a list of all playback clients from the database.
func (db *SqliteDB) PlaybackClientList() ([]PlaybackClient, error) {
	query := `SELECT id, name FROM ` + tb_playlists
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("SqliteDB.PlaybackClientList: %w", err)
	}
	defer rows.Close()

	var pbcs []PlaybackClient
	for rows.Next() {
		var pbc PlaybackClient
		err = rows.Scan(&pbc.ID, &pbc.Name)
		if err != nil {
			return nil, fmt.Errorf("SqliteDB.PlaybackClientList: %w", err)
		}

		pbcs = append(pbcs, pbc)
	}

	return pbcs, nil
}

// ############################################################################################## //
// ####################################         WOL          #################################### //
// ############################################################################################## //

// WOLMigrate creates the 'wol' table if it does not exist.
func (db *SqliteDB) WOLMigrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS ` + tb_wol + ` (
		pbc_id VARCHAR(11) NOT NULL PRIMARY KEY,
		alias VARCHAR(255) NOT NULL,
		mac VARCHAR(17) NOT NULL,
		interface VARCHAR(15) NOT NULL,
		port INTEGER NOT NULL DEFAULT 9,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		FOREIGN KEY (pbc_id) REFERENCES ` + tb_playlists + `(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_wol_pbc_id ON ` + tb_wol + ` (pbc_id);
	CREATE INDEX IF NOT EXISTS idx_wol_mac ON ` + tb_wol + ` (mac);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("SqliteDB.WOLMigrate: %w", err)
	}

	return nil
}

// WOLIsUnique checks if the WOL mac address is unique in the database.
func (db *SqliteDB) WOLIsUnique(mac string) error {
	if mac == "" {
		return fmt.Errorf("SqliteDB.WOLIsUnique: mac - %w", ErrParamEmpty)
	}

	err := db.IsUnique(tb_wol, "mac = ?", mac)
	if errors.Is(err, ErrRecordExists) {
		return fmt.Errorf("SqliteDB.WOLIsUnique: %w", ErrRecordExists)
	}

	return err
}

// WOLCreate creates a new WOL record in the database.
func (db *SqliteDB) WOLCreate(wol WOL) error {
	if err := wol.Validate(); err != nil {
		return fmt.Errorf("SqliteDB.WOLCreate: %w", err)
	}

	query := `INSERT INTO ` + tb_wol + ` (pbc_id, alias, mac, interface, port, enabled) VALUES (?, ?, ?, ?, ?, ?)`
	r, err := db.Exec(query, wol.PBCID, wol.Alias, wol.MAC, wol.Interface, wol.Port, true)
	if err != nil {
		if IsErrNotUnique(err) {
			return fmt.Errorf("SqliteDB.WOLCreate: %w", ErrRecordExists)
		}

		return fmt.Errorf("SqliteDB.WOLCreate: %w", err)
	}

	_, err = r.LastInsertId()
	if err != nil {
		return fmt.Errorf("SqliteDB.WOLCreate: %w", err)
	}

	return nil
}

// WOLGet retrieves a WOL record from the database by PBC ID.
func (db *SqliteDB) WOLGet(pbcid string) (WOL, error) {
	query := `SELECT * FROM ` + tb_wol + ` WHERE pbc_id = ?`
	row, err := db.QueryRow(query, pbcid)
	if err != nil {
		return WOL{}, fmt.Errorf("SqliteDB.WOLGet: %w", err)
	}

	wol := WOL{}
	err = row.Scan(
		&wol.PBCID,
		&wol.Alias,
		&wol.MAC,
		&wol.Interface,
		&wol.Port,
		&wol.Enabled,
	)
	if err != nil {
		return wol, fmt.Errorf("SqliteDB.WOLGet: %w", err)
	}

	return wol, nil
}

// WOLUpdate updates a WOL record in the database.
func (db *SqliteDB) WOLUpdate(wol WOL) error {
	if err := wol.Validate(); err != nil {
		return fmt.Errorf("SqliteDB.WOLUpdate: %w", err)
	}

	query := `UPDATE ` + tb_wol + ` SET alias = ?, mac = ?, interface = ?, port = ?, enabled = ? WHERE pbc_id = ?`
	if _, err := db.Exec(query, wol.Alias, wol.MAC, wol.Interface, wol.Port, wol.Enabled, wol.PBCID); err != nil {
		return fmt.Errorf("SqliteDB.WOLUpdate: %w", err)
	}

	return nil
}

// WOLDelete deletes a WOL record from the database by PBC ID.
func (db *SqliteDB) WOLDelete(pbcid string) error {
	if pbcid == "" {
		return fmt.Errorf("SqliteDB.WOLDelete: %w", ErrInvalidID)
	}

	if err := db.WOLIsUnique(pbcid); err == nil {
		return fmt.Errorf("SqliteDB.WOLDelete: %w", sql.ErrNoRows)
	}

	query := `DELETE FROM ` + tb_wol + ` WHERE pbc_id = ?`
	if _, err := db.Exec(query, pbcid); err != nil {
		return fmt.Errorf("SqliteDB.WOLDelete: %w", err)
	}

	return nil
}

// ############################################################################################## //
// ####################################         CEC          #################################### //
// ############################################################################################## //

// WOLMigrate creates the 'wol' table if it does not exist.
func (db *SqliteDB) CECMigrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS ` + tb_cec + ` (
		pbc_id VARCHAR(11) NOT NULL PRIMARY KEY,
		alias VARCHAR(14) NOT NULL,
		device VARCHAR(14) NOT NULL,
		logical_addr VARCHAR(7) NOT NULL,
		physical_addr VARCHAR(7) NOT NULL,
		FOREIGN KEY (pbc_id) REFERENCES ` + tb_playlists + `(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_cec_pbc_id ON ` + tb_cec + ` (pbc_id);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("SqliteDB.CECMigrate: %w", err)
	}

	return nil
}

// CECIsUnique checks if the CEC device is unique in the database.
func (db *SqliteDB) CECIsUnique(pbcID string) error {
	if pbcID == "" {
		return fmt.Errorf("SqliteDB.CECIsUnique: device - %w", ErrParamEmpty)
	}

	err := db.IsUnique(tb_cec, "pbc_id = ?", pbcID)
	if errors.Is(err, ErrRecordExists) {
		return fmt.Errorf("SqliteDB.CECIsUnique: %w", ErrRecordExists)
	}

	return err
}

// CECCreate creates a new CEC record in the database.
func (db *SqliteDB) CECCreate(cec CEC) error {
	if err := cec.Validate(); err != nil {
		return fmt.Errorf("SqliteDB.CECCreate: %w", err)
	}

	query := `INSERT INTO ` + tb_cec + ` (pbc_id, alias, device, logical_addr, physical_addr) VALUES (?, ?, ?, ?, ?)`
	r, err := db.Exec(query, cec.PBCID, cec.Alias, cec.Device, cec.LogicalAddr, cec.PhysicalAddr)
	if err != nil {
		return fmt.Errorf("SqliteDB.CECCreate: %w", err)
	}

	_, err = r.LastInsertId()
	if err != nil {
		return fmt.Errorf("SqliteDB.CECCreate: %w", err)
	}

	return nil
}

func (db *SqliteDB) CECGet(pbcid string) (CEC, error) {
	query := `SELECT * FROM ` + tb_cec + ` WHERE pbc_id = ?`
	row, err := db.QueryRow(query, pbcid)
	if err != nil {
		return CEC{}, fmt.Errorf("SqliteDB.CECGet: %w", err)
	}

	cec := CEC{}
	err = row.Scan(
		&cec.PBCID,
		&cec.Alias,
		&cec.Device,
		&cec.LogicalAddr,
		&cec.PhysicalAddr,
	)
	if err != nil {
		return cec, fmt.Errorf("SqliteDB.CECGet: %w", err)
	}

	return cec, nil
}

func (db *SqliteDB) CECUpdate(cec CEC) error {
	if err := cec.Validate(); err != nil {
		return fmt.Errorf("SqliteDB.CECUpdate: %w", err)
	}

	query := `UPDATE ` + tb_cec + ` SET alias = ?, device = ?, logical_addr = ?, physical_addr = ? WHERE pbc_id = ?`
	if _, err := db.Exec(query, cec.Alias, cec.Device, cec.LogicalAddr, cec.PhysicalAddr, cec.PBCID); err != nil {
		return fmt.Errorf("SqliteDB.CECUpdate: %w", err)
	}

	return nil
}

func (db *SqliteDB) CECDelete(pbcid string) error {
	if pbcid == "" {
		return fmt.Errorf("SqliteDB.CECDelete: %w", ErrInvalidID)
	}

	if err := db.CECIsUnique(pbcid); err == nil {
		return fmt.Errorf("SqliteDB.CECDelete: %w", sql.ErrNoRows)
	}

	query := `DELETE FROM ` + tb_cec + ` WHERE pbc_id = ?`
	if _, err := db.Exec(query, pbcid); err != nil {
		return fmt.Errorf("SqliteDB.CECDelete: %w", err)
	}

	return nil
}
