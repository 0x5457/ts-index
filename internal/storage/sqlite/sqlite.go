package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/0x5457/ts-index/internal/models"
	_ "modernc.org/sqlite"
)

type SymbolStore struct {
	db *sql.DB
}

func New(path string) (*SymbolStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &SymbolStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS symbols (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		kind TEXT NOT NULL,
		file TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		docstring TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
	CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file);
	CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);`)
	return err
}

func (s *SymbolStore) UpsertSymbols(symbols []models.Symbol) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO symbols(id,name,kind,file,start_line,end_line,docstring)
		VALUES(?,?,?,?,?,?,?)
        ON CONFLICT(id) DO UPDATE SET
        name=excluded.name,
        kind=excluded.kind,
        file=excluded.file,
        start_line=excluded.start_line,
        end_line=excluded.end_line,
        docstring=excluded.docstring`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, sym := range symbols {
		if _, err := stmt.Exec(
			sym.ID,
			sym.Name,
			fmt.Sprint(rune(sym.Kind)),
			sym.File,
			sym.StartLine,
			sym.EndLine,
			sym.Docstring,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SymbolStore) DeleteSymbolsByFile(file string) error {
	_, err := s.db.Exec(`DELETE FROM symbols WHERE file = ?`, file)
	return err
}

func (s *SymbolStore) FindByName(name string) ([]models.Symbol, error) {
	rows, err := s.db.Query(
		`SELECT id,name,kind,file,start_line,end_line,docstring FROM symbols WHERE name = ?`,
		name,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []models.Symbol
	for rows.Next() {
		var sym models.Symbol
		var kind string
		if err := rows.Scan(&sym.ID, &sym.Name, &kind, &sym.File, &sym.StartLine, &sym.EndLine, &sym.Docstring); err != nil {
			return nil, err
		}
		sym.Kind = models.StringToSymbolKind(kind)
		out = append(out, sym)
	}
	return out, rows.Err()
}

func (s *SymbolStore) GetByID(id string) (*models.Symbol, error) {
	row := s.db.QueryRow(
		`SELECT id,name,kind,file,start_line,end_line,docstring FROM symbols WHERE id = ?`,
		id,
	)
	var sym models.Symbol
	var kind string
	if err := row.Scan(&sym.ID, &sym.Name, &kind, &sym.File, &sym.StartLine, &sym.EndLine, &sym.Docstring); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	sym.Kind = models.StringToSymbolKind(kind)
	return &sym, nil
}
