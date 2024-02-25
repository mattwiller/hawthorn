package internal

import (
	"fmt"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type DB struct {
	pool *sqlitex.Pool
}

func NewDB(path string) (*DB, error) {
	pool, err := sqlitex.Open(path, sqlite.OpenCreate|sqlite.OpenReadWrite, 10)
	if err != nil {
		return nil, err
	}

	return &DB{
		pool: pool,
	}, nil
}

func (db *DB) Query(query string, args ...any) ([]Row, error) {
	conn := db.pool.Get(nil)
	if conn == nil {
		return nil, fmt.Errorf("failed to get a database connection from the pool")
	}
	defer db.pool.Put(conn)

	var results []Row
	err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			results = append(results, ParseRow(stmt))
			return nil
		},
	})
	return results, err
}

func (db *DB) Close() error {
	return db.pool.Close()
}

func (db *DB) Batch(conn *sqlite.Conn) error {
    if conn == nil {
        return fmt.Errorf("no database connection provided")
    }

	return sqlitex.Execute(conn, "BEGIN", &sqlitex.ExecOptions{
        ResultFunc: func(stmt *sqlite.Stmt) error {
            return nil
        },
    })
}

func (db *DB) Flush(conn *sqlite.Conn) error {
    if conn == nil {
        return fmt.Errorf("no database connection provided")
    }

	return sqlitex.Execute(conn, "COMMIT", &sqlitex.ExecOptions{
        ResultFunc: func(stmt *sqlite.Stmt) error {
            return nil
        },
	})
}
 

func (db *DB) GetConnection() (*sqlite.Conn, error) {
    conn := db.pool.Get(nil)
    if conn == nil {
        return nil, fmt.Errorf("failed to get a database connection from the pool")
    }
    return conn, nil
}

func (db *DB) PutConnection(conn *sqlite.Conn) {
    db.pool.Put(conn)
}

func (db *DB) QueryWithConnection(conn *sqlite.Conn, query string, args ...any) ([]Row, error) {
    if conn == nil {
        return nil, fmt.Errorf("no database connection provided")
    }

    var results []Row
    err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
        Args: args,
        ResultFunc: func(stmt *sqlite.Stmt) error {
            results = append(results, ParseRow(stmt))
            return nil
        },
    })
    return results, err
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
