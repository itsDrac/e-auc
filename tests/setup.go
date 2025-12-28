package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/itsDrac/e-auc/internal/dependency"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Global test environment shared across all tests
var GlobalTestEnv *TestEnv

// TestEnv holds all test dependencies and containers
type TestEnv struct {
	Dependencies       *dependency.Dependencies
	PostgresContainer  *postgres.PostgresContainer
	MinioContainer     testcontainers.Container
	RedisContainer     testcontainers.Container
	DBConnectionString string
	MinioEndpoint      string
	RedisEndpoint      string
	Context            context.Context
}

// SetupTestEnvironment creates all required test containers and dependencies
func SetupTestEnvironment(ctx context.Context) (*TestEnv, error) {
	env := &TestEnv{
		Context: ctx,
	}

	// Start PostgreSQL container
	pgContainer, err := setupPostgresContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup postgres container: %w", err)
	}
	env.PostgresContainer = pgContainer

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}
	env.DBConnectionString = connStr

	// Run migrations
	if err := runMigrations(connStr); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Start MinIO container
	minioContainer, minioEndpoint, err := setupMinioContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup minio container: %w", err)
	}
	env.MinioContainer = minioContainer
	env.MinioEndpoint = minioEndpoint

	// Start Redis container
	redisContainer, redisEndpoint, err := setupRedisContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup redis container: %w", err)
	}
	env.RedisContainer = redisContainer
	env.RedisEndpoint = redisEndpoint

	// Set environment variables to point to test containers
	setTestEnvVars(connStr, minioEndpoint, redisEndpoint)

	// Initialize dependencies
	deps, err := dependency.NewDependencies(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	env.Dependencies = deps

	return env, nil
}

// setupPostgresContainer starts a PostgreSQL testcontainer
func setupPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, error) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	return pgContainer, nil
}

// setupMinioContainer starts a MinIO testcontainer
func setupMinioContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd: []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to start minio container: %w", err)
	}

	host, err := minioContainer.Host(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get minio host: %w", err)
	}

	mappedPort, err := minioContainer.MappedPort(ctx, "9000")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get minio port: %w", err)
	}

	endpoint := fmt.Sprintf("%s:%s", host, mappedPort.Port())
	return minioContainer, endpoint, nil
}

// setupRedisContainer starts a Redis testcontainer
func setupRedisContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		Env: map[string]string{
			"REDIS_PASSWORD": "testredispass",
		},
		Cmd: []string{"redis-server", "--requirepass", "testredispass"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to start redis container: %w", err)
	}

	host, err := redisContainer.Host(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get redis host: %w", err)
	}

	mappedPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get redis port: %w", err)
	}

	endpoint := fmt.Sprintf("%s:%s", host, mappedPort.Port())
	return redisContainer, endpoint, nil
}

// runMigrations runs database migrations
func runMigrations(dbURL string) error {
	// Get the project root directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	migrationsPath := filepath.Join(projectRoot, "migrations")
	migrationsURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.New(migrationsURL, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// getProjectRoot finds the project root directory
func getProjectRoot() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up until we find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// setTestEnvVars sets environment variables for test containers
func setTestEnvVars(dbURL, minioEndpoint, redisEndpoint string) {
	// Database
	os.Setenv("DB_DSN", dbURL)

	// MinIO
	os.Setenv("MINIO_ENDPOINT", minioEndpoint)
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadmin")
	os.Setenv("MINIO_USE_SSL", "false")

	// Redis
	os.Setenv("REDIS_ADDR", redisEndpoint)
	os.Setenv("REDIS_PASSWORD", "testredispass")
	os.Setenv("REDIS_DB", "0")
	os.Setenv("REDIS_PORT", "6379")

	// JWT secrets (for testing)
	os.Setenv("ACCESS_TOKEN_SECRET", "test-access-secret-key-for-testing")
	os.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-secret-key-for-testing")

	// Server
	os.Setenv("SERVER_HOST", "localhost")
	os.Setenv("SERVER_PORT", "8080")
}

// Cleanup terminates all test containers and closes connections
func (env *TestEnv) Cleanup() error {
	var errs []error

	// Close dependencies
	if env.Dependencies != nil {
		if env.Dependencies.Conn != nil {
			if err := env.Dependencies.Conn.Close(env.Context); err != nil {
				errs = append(errs, fmt.Errorf("failed to close db connection: %w", err))
			}
		}
		if env.Dependencies.Cache != nil {
			if err := env.Dependencies.Cache.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close cache: %w", err))
			}
		}
	}

	// Terminate containers
	if env.PostgresContainer != nil {
		if err := env.PostgresContainer.Terminate(env.Context); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate postgres container: %w", err))
		}
	}

	if env.MinioContainer != nil {
		if err := env.MinioContainer.Terminate(env.Context); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate minio container: %w", err))
		}
	}

	if env.RedisContainer != nil {
		if err := env.RedisContainer.Terminate(env.Context); err != nil {
			errs = append(errs, fmt.Errorf("failed to terminate redis container: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

// Setup is the main test setup function that can be called from TestMain
// It initializes the global test environment once and cleans up after all tests
func Setup(m *testing.M) int {
	ctx := context.Background()

	// Setup test environment once for all tests
	env, err := SetupTestEnvironment(ctx)
	if err != nil {
		log.Fatalf("Failed to setup test environment: %v", err)
		return 1
	}

	// Set global variable so all tests can access it
	GlobalTestEnv = env

	// Ensure cleanup happens after all tests complete
	defer func() {
		if err := env.Cleanup(); err != nil {
			log.Printf("Failed to cleanup test environment: %v", err)
		}
	}()

	// Run all tests
	return m.Run()
}

// GetTestEnv returns the global test environment
// Use this in your tests to access the shared test environment
func GetTestEnv() *TestEnv {
	if GlobalTestEnv == nil {
		log.Fatal("Test environment not initialized. Make sure TestMain calls Setup(m)")
	}
	return GlobalTestEnv
}
