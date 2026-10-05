package storage

import (
	"errors"
	"fmt"

	"snorlx/backend/internal/tokencrypt"

	"github.com/rs/zerolog/log"
)

// NewStorage creates the storage backend for the requested mode.
//
// Database mode fails closed: a missing DATABASE_URL or an unreachable database is an error, never a
// silent fallback to memory, because that would present an empty but "healthy" dashboard and lose
// every write made while the database is down.
func NewStorage(mode StorageMode, databaseURL string, cipher *tokencrypt.Cipher) (Storage, error) {
	switch mode {
	case StorageModeDatabase:
		if databaseURL == "" {
			return nil, errors.New("STORAGE_MODE=database requires DATABASE_URL")
		}
		if cipher == nil {
			return nil, errors.New("database storage requires a token cipher")
		}
		log.Info().Msg("Initializing database storage (PostgreSQL + TimescaleDB)")
		store, err := NewDatabaseStorage(databaseURL, cipher)
		if err != nil {
			return nil, fmt.Errorf("initialize database storage: %w", err)
		}
		return store, nil
	case StorageModeMemory:
		return NewMemoryStorage(), nil
	default:
		return nil, fmt.Errorf("unknown storage mode %q", mode)
	}
}
