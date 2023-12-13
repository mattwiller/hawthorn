package internal

import (
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type DB struct {
	conn *sqlite.Conn
}

func NewDB(path string) (*DB, error) {
	conn, err := sqlite.OpenConn(path, sqlite.OpenCreate, sqlite.OpenReadWrite)
	if err != nil {
		return nil, err
	}

	return &DB{
		conn: conn,
	}, nil
}

func (db *DB) Query(query string, args ...any) ([]Row, error) {
	var results []Row
	err := sqlitex.Execute(db.conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			results = append(results, ParseRow(stmt))
			return nil
		},
	})
	return results, err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Batch() error {
	return sqlitex.Execute(db.conn, "BEGIN", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			return nil
		},
	})
}

func (db *DB) Flush() error {
	return sqlitex.Execute(db.conn, "COMMIT", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			return nil
		},
	})
}

type Row map[string]any

func ParseRow(stmt *sqlite.Stmt) Row {
	row := make(map[string]any, stmt.ColumnCount())
	for i := 0; i < stmt.ColumnCount(); i++ {
		name := stmt.ColumnName(i)
		switch stmt.ColumnType(i) {
		case sqlite.TypeInteger:
			row[name] = stmt.ColumnInt64(i)
		case sqlite.TypeNull:
			row[name] = nil
		case sqlite.TypeText:
			row[name] = stmt.ColumnText(i)
		}
	}
	return row
}
