// Package store предоставляет потокобезопасное кольцевое хранилище метрик.
package store

import "sync"

// DataPoint — одна точка измерений.
type DataPoint struct {
	Timestamp int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	NetIn     float64 `json:"netIn"`  // Mbps
	NetOut    float64 `json:"netOut"` // Mbps
	Clients   int     `json:"clients"`
}

// Store хранит последние maxSize точек в памяти.
type Store struct {
	mu      sync.RWMutex
	points  []DataPoint
	maxSize int
}

// New создаёт Store с указанной максимальной ёмкостью.
func New(maxSize int) *Store {
	return &Store{
		points:  make([]DataPoint, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add добавляет точку, вытесняя самую старую при переполнении.
func (s *Store) Add(p DataPoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.points = append(s.points, p)
	if len(s.points) > s.maxSize {
		// Срезаем одну точку с начала вместо полного перевыделения.
		s.points = s.points[1:]
	}
}

// GetAll возвращает копию всех точек в хронологическом порядке.
func (s *Store) GetAll() []DataPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]DataPoint, len(s.points))
	copy(out, s.points)
	return out
}
