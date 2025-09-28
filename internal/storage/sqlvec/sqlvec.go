package sqlvec

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/0x5457/ts-index/internal/models"
	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db        *sql.DB
	dimension int
}

func New(path string, dimension int) (*Store, error) {
	// enable sqlite-vec for all future connections
	sqlite_vec.Auto()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db, dimension); err != nil {
		return nil, err
	}
	return &Store{db: db, dimension: dimension}, nil
}

func migrate(db *sql.DB, dim int) error {
	// symbols table (reuse schema from sqlite store if not exists)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS symbols (
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
	CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);`); err != nil {
		return err
	}
	// chunks and vectors
	// chunks table stores metadata for retrieval
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS chunks (
		id TEXT PRIMARY KEY,
		file TEXT NOT NULL,
		language TEXT,
		node_type TEXT,
		start_line INTEGER,
		end_line INTEGER,
		start_byte INTEGER,
		end_byte INTEGER,
		content TEXT,
		docstring TEXT,
		signature TEXT,
		kind TEXT,
		name TEXT
	);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(file);`); err != nil {
		return err
	}
	// vec0 virtual table holds embeddings; dimension is fixed per table.
	// If dim <= 0, defer creation until first Upsert when dimension is known.
	if dim > 0 {
		if _, err := db.Exec(fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_embeddings USING vec0(
            embedding float32[%d]
        );`, dim)); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS vec_map (
            rid INTEGER UNIQUE NOT NULL,
            id TEXT UNIQUE NOT NULL
        );`); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_vec_map_id ON vec_map(id);`); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }

// Ensure Store implements storage.VectorStore-like methods
func (s *Store) Upsert(chunks []models.CodeChunk, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks and embeddings length mismatch")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// Ensure vec table exists with correct dimension
	if err := s.ensureVecTable(tx, embeddings); err != nil {
		_ = tx.Rollback()
		return err
	}

	// upsert chunks metadata
	chunkStmt, err := tx.Prepare(`INSERT INTO chunks(
		id,file,language,node_type,start_line,end_line,start_byte,end_byte,content,docstring,signature,kind,name
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(id) DO UPDATE SET
		file=excluded.file,
		language=excluded.language,
		node_type=excluded.node_type,
		start_line=excluded.start_line,
		end_line=excluded.end_line,
		start_byte=excluded.start_byte,
		end_byte=excluded.end_byte,
		content=excluded.content,
		docstring=excluded.docstring,
		signature=excluded.signature,
		kind=excluded.kind,
		name=excluded.name`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = chunkStmt.Close() }()

	// prepare statements for vector write and mapping
	insertVecStmt, err := tx.Prepare(`INSERT INTO vec_embeddings(embedding) VALUES(?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = insertVecStmt.Close() }()
	replaceVecStmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO vec_embeddings(rowid, embedding) VALUES(?, ?)`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = replaceVecStmt.Close() }()
	upsertMapStmt, err := tx.Prepare(`INSERT OR REPLACE INTO vec_map(rid, id) VALUES(?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = upsertMapStmt.Close() }()
	selectRidStmt, err := tx.Prepare(`SELECT rid FROM vec_map WHERE id = ?`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = selectRidStmt.Close() }()

	for i, ch := range chunks {
		if _, err := chunkStmt.Exec(
			ch.ID, ch.File, ch.Language, ch.NodeType, ch.StartLine, ch.EndLine, ch.StartByte, ch.EndByte,
			ch.Content, ch.Docstring, ch.Signature, fmt.Sprint(rune(ch.Kind)), ch.Name,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
		v, err := sqlite_vec.SerializeFloat32(embeddings[i])
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		// check existing rid
		var rid sql.NullInt64
		if err := selectRidStmt.QueryRow(ch.ID).Scan(&rid); err != nil &&
			!errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return err
		}
		if rid.Valid {
			if _, err := replaceVecStmt.Exec(rid.Int64, v); err != nil {
				_ = tx.Rollback()
				return err
			}
		} else {
			if _, err := insertVecStmt.Exec(v); err != nil {
				_ = tx.Rollback()
				return err
			}
			var newRid int64
			if err := tx.QueryRow(`SELECT last_insert_rowid()`).Scan(&newRid); err != nil {
				_ = tx.Rollback()
				return err
			}
			if _, err := upsertMapStmt.Exec(newRid, ch.ID); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteByFile(file string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT id FROM chunks WHERE file = ?`, file)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			_ = tx.Rollback()
			return err
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if _, err := tx.Exec(`DELETE FROM chunks WHERE file = ?`, file); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, id := range ids {
		// find rid via map
		var rid sql.NullInt64
		if err := tx.QueryRow(`SELECT rid FROM vec_map WHERE id = ?`, id).Scan(&rid); err != nil &&
			!errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return err
		}
		if rid.Valid {
			if _, err := tx.Exec(`DELETE FROM vec_embeddings WHERE rowid = ?`, rid.Int64); err != nil {
				_ = tx.Rollback()
				return err
			}
			if _, err := tx.Exec(`DELETE FROM vec_map WHERE rid = ?`, rid.Int64); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) Query(embedding []float32, topK int) ([]models.SemanticHit, error) {
	if topK <= 0 {
		topK = 5
	}
	v, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return nil, err
	}
	// KNN via MATCH ... ORDER BY distance using sqlite-vec
	rows, err := s.db.Query(`
        WITH knn AS (
            SELECT rowid, distance
            FROM vec_embeddings
            WHERE embedding MATCH ?
            ORDER BY distance
            LIMIT ?
        )
        SELECT c.id, c.file, c.language, c.node_type, c.start_line, c.end_line, c.start_byte, c.end_byte,
               c.content, c.docstring, c.signature, c.kind, c.name,
               k.distance as score
        FROM knn k
        JOIN vec_map m ON m.rid = k.rowid
        JOIN chunks c ON c.id = m.id
        ORDER BY k.distance ASC
    `, v, topK)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var hits []models.SemanticHit
	for rows.Next() {
		var ch models.CodeChunk
		var kind string
		var score float32
		if err := rows.Scan(
			&ch.ID, &ch.File, &ch.Language, &ch.NodeType, &ch.StartLine, &ch.EndLine, &ch.StartByte, &ch.EndByte,
			&ch.Content, &ch.Docstring, &ch.Signature, &kind, &ch.Name, &score,
		); err != nil {
			return nil, err
		}
		ch.Kind = models.StringToSymbolKind(kind)
		hits = append(hits, models.SemanticHit{Chunk: ch, Score: 1 - score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

func (s *Store) ensureVecTable(tx *sql.Tx, embeddings [][]float32) error {
	// Check if vec_embeddings exists
	var name string
	err := tx.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='vec_embeddings'`).
		Scan(&name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if name == "vec_embeddings" {
		return nil
	}
	// Create with inferred dim
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return fmt.Errorf("cannot create vec_embeddings: unknown embedding dimension")
	}
	dim := len(embeddings[0])
	if _, err := tx.Exec(fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_embeddings USING vec0(
        embedding float32[%d]
    );`, dim)); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS vec_map (
        rid INTEGER UNIQUE NOT NULL,
        id TEXT UNIQUE NOT NULL
    );`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_vec_map_id ON vec_map(id);`); err != nil {
		return err
	}
	s.dimension = dim
	return nil
}

// Optional symbol APIs mirroring existing sqlite store so callers can reuse one DB if desired
func (s *Store) UpsertSymbols(symbols []models.Symbol) error {
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

func (s *Store) DeleteSymbolsByFile(file string) error {
	_, err := s.db.Exec(`DELETE FROM symbols WHERE file = ?`, file)
	return err
}

func (s *Store) FindByName(name string) ([]models.Symbol, error) {
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

func (s *Store) GetByID(id string) (*models.Symbol, error) {
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
