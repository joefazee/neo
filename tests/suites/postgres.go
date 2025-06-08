package suites

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm/logger"

	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// Import necessary packages for migrations and database drivers
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type PostgresContainer struct {
	testcontainers.Container
	ConnectionString string
	Host             string
	Port             string
	Database         string
	Username         string
	Password         string
}

func (pc *PostgresContainer) GetConnectionString() string {
	return pc.ConnectionString
}

func NewPostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	const port = "5432/tcp"
	env := map[string]string{
		"POSTGRES_DB":       "testdb",
		"POSTGRES_PASSWORD": "testpass",
		"POSTGRES_USER":     "testuser",
	}

	dbURL := func(host string, port nat.Port) string {
		return fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17.5-alpine3.21",
		ExposedPorts: []string{port},
		Cmd:          []string{"postgres", "-c", "fsync=off"},
		Env:          env,
		WaitingFor: wait.ForSQL(port, "postgres", dbURL).
			WithStartupTimeout(30 * time.Second).
			WithQuery("SELECT 1"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	return &PostgresContainer{
		Container:        container,
		ConnectionString: dbURL(host, mappedPort),
		Host:             host,
		Port:             mappedPort.Port(),
		Database:         "testdb",
		Username:         "testuser",
		Password:         "testpass",
	}, nil
}

type RepositoryTestSuite struct {
	suite.Suite
	Container           *PostgresContainer
	DB                  *gorm.DB
	SQLDB               *sql.DB
	AutoMigrate         bool
	MigrationsPath      string
	SkipDatabaseCleanup bool // Allow tests to skip cleanup
}

func (suite *RepositoryTestSuite) SetupSuite() {
	suite.T().Helper()

	if testing.Short() {
		suite.T().Skip("Skipping database integration tests in short mode")
	}

	// Set defaults
	if suite.MigrationsPath == "" {
		suite.MigrationsPath = suite.findMigrationsPath()
	}
	if !suite.hasMigrationsPath() {
		suite.AutoMigrate = false
	}

	// Create container and connections once
	suite.createContainer()
	suite.createConnections()

	// Run migrations if enabled
	if suite.AutoMigrate {
		if err := suite.RunMigrations(); err != nil {
			suite.T().Fatalf("Failed to run migrations: %v", err)
		}
	}

	// Register cleanup
	suite.T().Cleanup(func() {
		suite.cleanup()
	})
}

func (suite *RepositoryTestSuite) createContainer() {
	ctx := context.Background()
	container, err := NewPostgresContainer(ctx)
	if err != nil {
		suite.T().Fatalf("Failed to create postgres container: %v", err)
	}
	suite.Container = container
}

func (suite *RepositoryTestSuite) createConnections() {
	// Close existing connections if any
	if suite.SQLDB != nil {
		_ = suite.SQLDB.Close()
	}

	// Create SQL connection
	sqlDB, err := sql.Open("postgres", suite.Container.ConnectionString)
	if err != nil {
		suite.T().Fatalf("Failed to open sql connection: %v", err)
	}
	suite.SQLDB = sqlDB

	// Configure connection pool
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		suite.T().Fatalf("Failed to ping database: %v", err)
	}

	// Create GORM connection
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		suite.T().Fatalf("Failed to open gorm connection: %v", err)
	}
	suite.DB = gormDB
}

func (suite *RepositoryTestSuite) findMigrationsPath() string {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return filepath.Join(wd, "migrations")
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return ""
		}
		wd = parent
	}
}

func (suite *RepositoryTestSuite) hasMigrationsPath() bool {
	if suite.MigrationsPath == "" {
		return false
	}
	_, err := os.Stat(suite.MigrationsPath)
	return err == nil
}

func (suite *RepositoryTestSuite) SetupTest() {
	// Override in child suites if needed
}

func (suite *RepositoryTestSuite) TearDownTest() {
	suite.T().Helper()

	// Skip cleanup if disabled
	if suite.SkipDatabaseCleanup {
		return
	}

	if suite.DB == nil {
		return
	}

	// Use a simpler approach - delete all data from tables
	var tables []string
	suite.DB.Raw(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		AND table_name NOT LIKE 'pg_%'
		AND table_name NOT IN ('schema_migrations', 'gorp_migrations')
	`).Scan(&tables)

	// Delete in reverse order to handle foreign keys
	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		suite.DB.Exec(fmt.Sprintf(`DELETE FROM %q`, table))
	}

	// Reset sequences
	for _, table := range tables {
		suite.DB.Exec(fmt.Sprintf(`ALTER SEQUENCE IF EXISTS %s_id_seq RESTART WITH 1`, table))
	}
}

func (suite *RepositoryTestSuite) cleanup() {
	ctx := context.Background()
	if suite.SQLDB != nil {
		_ = suite.SQLDB.Close()
	}
	if suite.Container != nil {
		_ = suite.Container.Terminate(ctx)
	}
}

func (suite *RepositoryTestSuite) RunMigrations() error {
	if suite.MigrationsPath == "" {
		return errors.New("migrations path not set")
	}

	sourceURL := fmt.Sprintf("file://%s", suite.MigrationsPath)
	m, err := migrate.New(sourceURL, suite.GetConnectionString())
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func (suite *RepositoryTestSuite) BeforeTest(_, _ string) {
	if suite.DB != nil && !suite.SkipDatabaseCleanup {
		suite.TearDownTest()
	}
}

func (suite *RepositoryTestSuite) GetDB() *gorm.DB   { return suite.DB }
func (suite *RepositoryTestSuite) GetSQLDB() *sql.DB { return suite.SQLDB }
func (suite *RepositoryTestSuite) GetConnectionString() string {
	return suite.Container.GetConnectionString()
}
func (suite *RepositoryTestSuite) CountRecords(table string) int64 {
	var c int64
	suite.DB.Table(table).Count(&c)
	return c
}
func (suite *RepositoryTestSuite) TableExists(table string) bool {
	return suite.DB.Migrator().HasTable(table)
}
func (suite *RepositoryTestSuite) AssertDBError(err error, args ...interface{}) {
	suite.Assert().Error(err, args...)
}
func (suite *RepositoryTestSuite) AssertNoDBError(err error, args ...interface{}) {
	suite.Assert().NoError(err, args...)
}

func (suite *RepositoryTestSuite) WithTransaction(fn func(tx *gorm.DB) error) error {
	return suite.DB.Transaction(fn)
}

func (suite *RepositoryTestSuite) ExecRaw(sql string, args ...interface{}) error {
	return suite.DB.Exec(sql, args...).Error
}
