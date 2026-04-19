// Package store предоставляет SQLite-хранилище метрик с автоматической очисткой.
package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DataPoint — одна точка измерений.
type DataPoint struct {
	Timestamp int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	NetIn     float64 `json:"netIn"`  // Mbps
	NetOut    float64 `json:"netOut"` // Mbps
	Clients   int     `json:"clients"`
}

// Store хранит метрики в SQLite базе данных.
type Store struct {
	db            *sql.DB
	retentionDays int
}

// New открывает или создаёт SQLite БД в dataDir и запускает фоновую очистку.
func New(dataDir string, retentionDays int) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir %q: %w", dataDir, err)
	}

	dbPath := filepath.Join(dataDir, "metrics.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dbPath, err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS metrics (
		ts      INTEGER PRIMARY KEY,
		cpu     REAL    NOT NULL,
		ram     REAL    NOT NULL,
		net_in  REAL    NOT NULL,
		net_out REAL    NOT NULL,
		clients INTEGER NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("create metrics table: %w", err)
	}

	s := &Store{db: db, retentionDays: retentionDays}
	go s.runCleanup()
	return s, nil
}

// Add записывает точку в БД.
func (s *Store) Add(p DataPoint) {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO metrics (ts, cpu, ram, net_in, net_out, clients) VALUES (?, ?, ?, ?, ?, ?)`,
		p.Timestamp, p.CPU, p.RAM, p.NetIn, p.NetOut, p.Clients,
	)
	if err != nil {
		log.Printf("[store] insert: %v", err)
	}
}

// GetAll возвращает все точки в хронологическом порядке.
func (s *Store) GetAll() []DataPoint {
	rows, err := s.db.Query(`SELECT ts, cpu, ram, net_in, net_out, clients FROM metrics ORDER BY ts`)
	if err != nil {
		log.Printf("[store] query: %v", err)
		return nil
	}
	defer rows.Close()

	var points []DataPoint
	for rows.Next() {
		var p DataPoint
		if err := rows.Scan(&p.Timestamp, &p.CPU, &p.RAM, &p.NetIn, &p.NetOut, &p.Clients); err != nil {
			log.Printf("[store] scan: %v", err)
			continue
		}
		points = append(points, p)
	}
	return points
}

func (s *Store) runCleanup() {
	// Первая очистка сразу при старте, затем раз в час.
	s.cleanup()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *Store) cleanup() {
	cutoff := time.Now().AddDate(0, 0, -s.retentionDays).UnixMilli()
	res, err := s.db.Exec(`DELETE FROM metrics WHERE ts < ?`, cutoff)
	if err != nil {
		log.Printf("[store] cleanup: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("[store] cleaned up %d old data points", n)
	}
}
