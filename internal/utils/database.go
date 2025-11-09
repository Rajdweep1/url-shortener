package utils

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rajweepmondal/url-shortener/internal/config"
)

// DatabaseConnection holds database connections
type DatabaseConnection struct {
	PostgreSQL *sql.DB
	Redis      *redis.Client
}

// NewDatabaseConnection creates new database connections
func NewDatabaseConnection(cfg *config.Config) (*DatabaseConnection, error) {
	// Connect to PostgreSQL
	postgres, err := connectPostgreSQL(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Connect to Redis
	redisClient, err := connectRedis(cfg.Redis)
	if err != nil {
		postgres.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &DatabaseConnection{
		PostgreSQL: postgres,
		Redis:      redisClient,
	}, nil
}

// connectPostgreSQL establishes a connection to PostgreSQL
func connectPostgreSQL(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return db, nil
}

// connectRedis establishes a connection to Redis
func connectRedis(cfg config.RedisConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure connection pool
	opt.PoolSize = cfg.PoolSize
	opt.MinIdleConns = cfg.MinIdleConn
	opt.DialTimeout = cfg.DialTimeout
	opt.ReadTimeout = cfg.ReadTimeout
	opt.WriteTimeout = cfg.WriteTimeout

	client := redis.NewClient(opt)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

// Close closes all database connections
func (dc *DatabaseConnection) Close() error {
	var errors []error

	if dc.PostgreSQL != nil {
		if err := dc.PostgreSQL.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close PostgreSQL: %w", err))
		}
	}

	if dc.Redis != nil {
		if err := dc.Redis.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// HealthCheck checks the health of all database connections
func (dc *DatabaseConnection) HealthCheck(ctx context.Context) map[string]string {
	status := make(map[string]string)

	// Check PostgreSQL
	if dc.PostgreSQL != nil {
		if err := dc.PostgreSQL.PingContext(ctx); err != nil {
			status["postgresql"] = fmt.Sprintf("unhealthy: %v", err)
		} else {
			status["postgresql"] = "healthy"
		}
	} else {
		status["postgresql"] = "not connected"
	}

	// Check Redis
	if dc.Redis != nil {
		if err := dc.Redis.Ping(ctx).Err(); err != nil {
			status["redis"] = fmt.Sprintf("unhealthy: %v", err)
		} else {
			status["redis"] = "healthy"
		}
	} else {
		status["redis"] = "not connected"
	}

	return status
}

// GetPostgreSQLStats returns PostgreSQL connection statistics
func (dc *DatabaseConnection) GetPostgreSQLStats() sql.DBStats {
	if dc.PostgreSQL == nil {
		return sql.DBStats{}
	}
	return dc.PostgreSQL.Stats()
}

// GetRedisPoolStats returns Redis connection pool statistics
func (dc *DatabaseConnection) GetRedisPoolStats() *redis.PoolStats {
	if dc.Redis == nil {
		return nil
	}
	return dc.Redis.PoolStats()
}

// MigrateDatabase runs database migrations
func MigrateDatabase(databaseURL string, migrationsPath string) error {
	// This would typically use a migration library like golang-migrate
	// For now, we'll provide a placeholder implementation
	
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database for migration: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database for migration: %w", err)
	}

	// In a real implementation, you would:
	// 1. Read migration files from migrationsPath
	// 2. Track applied migrations in a schema_migrations table
	// 3. Apply pending migrations in order
	// 4. Handle rollbacks if needed

	fmt.Printf("Migration placeholder - would run migrations from %s\n", migrationsPath)
	return nil
}

// IsPostgreSQLError checks if an error is a PostgreSQL error and returns the error code
func IsPostgreSQLError(err error, code string) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return string(pqErr.Code) == code
	}
	return false
}

// IsUniqueViolation checks if an error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	return IsPostgreSQLError(err, "23505")
}

// IsForeignKeyViolation checks if an error is a foreign key constraint violation
func IsForeignKeyViolation(err error) bool {
	return IsPostgreSQLError(err, "23503")
}

// IsCheckViolation checks if an error is a check constraint violation
func IsCheckViolation(err error) bool {
	return IsPostgreSQLError(err, "23514")
}

// IsNotNullViolation checks if an error is a not null constraint violation
func IsNotNullViolation(err error) bool {
	return IsPostgreSQLError(err, "23502")
}

// TransactionWrapper provides a helper for database transactions
type TransactionWrapper struct {
	db *sql.DB
}

// NewTransactionWrapper creates a new transaction wrapper
func NewTransactionWrapper(db *sql.DB) *TransactionWrapper {
	return &TransactionWrapper{db: db}
}

// WithTransaction executes a function within a database transaction
func (tw *TransactionWrapper) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := tw.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
