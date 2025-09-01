package migration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

// BaseMigrator provides common functionality for all database migrators
type BaseMigrator struct {
	migrations []Migration
	provider   Provider
	config     Config
}

// Config holds migration configuration
type Config struct {
	TableName          string        // Name of migrations table (default: schema_migrations)
	LockTimeout        time.Duration // Timeout for acquiring migration lock
	ValidateChecksums  bool          // Whether to validate checksums
	AutoRollback       bool          // Automatically rollback on failure
	RequireConfirmation bool         // Require confirmation for destructive operations
}

// Provider defines the database-specific operations
type Provider interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	BeginTransaction(ctx context.Context) (Transaction, error)
	
	// Migration table operations
	CreateMigrationsTable(ctx context.Context) error
	GetAppliedMigrations(ctx context.Context) ([]MigrationStatus, error)
	RecordMigration(ctx context.Context, status MigrationStatus) error
	RemoveMigration(ctx context.Context, version int64) error
	
	// Lock operations
	AcquireLock(ctx context.Context, timeout time.Duration) error
	ReleaseLock(ctx context.Context) error
	
	// Database-specific operations
	SupportsTransactionalDDL() bool
	GetDatabaseName() string
}

// NewBaseMigrator creates a new base migrator
func NewBaseMigrator(provider Provider, config Config) *BaseMigrator {
	if config.TableName == "" {
		config.TableName = "schema_migrations"
	}
	if config.LockTimeout == 0 {
		config.LockTimeout = 30 * time.Second
	}
	
	return &BaseMigrator{
		migrations: make([]Migration, 0),
		provider:   provider,
		config:     config,
	}
}

// AddMigration adds a migration to the migrator
func (m *BaseMigrator) AddMigration(migration Migration) {
	// Calculate checksum if not provided
	if migration.Checksum == "" {
		migration.Checksum = m.calculateChecksum(migration)
	}
	
	m.migrations = append(m.migrations, migration)
	m.sortMigrations()
}

// GetMigrations returns all registered migrations
func (m *BaseMigrator) GetMigrations() []Migration {
	return m.migrations
}

// Up runs all pending migrations
func (m *BaseMigrator) Up(ctx context.Context) error {
	// Acquire lock
	if err := m.provider.AcquireLock(ctx, m.config.LockTimeout); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.provider.ReleaseLock(ctx)
	
	// Ensure migrations table exists
	if err := m.provider.CreateMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	// Get applied migrations
	applied, err := m.provider.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	appliedMap := make(map[int64]bool)
	for _, status := range applied {
		appliedMap[status.Version] = true
	}
	
	// Run pending migrations
	pendingCount := 0
	for _, migration := range m.migrations {
		if appliedMap[migration.Version] {
			continue
		}
		
		if err := m.runMigration(ctx, migration, "up"); err != nil {
			return err
		}
		pendingCount++
	}
	
	if pendingCount == 0 {
		fmt.Println("No pending migrations")
	} else {
		fmt.Printf("Successfully applied %d migrations\n", pendingCount)
	}
	
	return nil
}

// Down rolls back the last migration
func (m *BaseMigrator) Down(ctx context.Context) error {
	// Acquire lock
	if err := m.provider.AcquireLock(ctx, m.config.LockTimeout); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.provider.ReleaseLock(ctx)
	
	// Get current version
	version, err := m.Version(ctx)
	if err != nil {
		return err
	}
	
	if version == 0 {
		return fmt.Errorf("no migrations to rollback")
	}
	
	// Find migration to rollback
	var migration *Migration
	for _, mig := range m.migrations {
		if mig.Version == version {
			migration = &mig
			break
		}
	}
	
	if migration == nil {
		return fmt.Errorf("migration %d not found", version)
	}
	
	return m.runMigration(ctx, *migration, "down")
}

// UpTo migrates up to a specific version
func (m *BaseMigrator) UpTo(ctx context.Context, targetVersion int64) error {
	// Acquire lock
	if err := m.provider.AcquireLock(ctx, m.config.LockTimeout); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.provider.ReleaseLock(ctx)
	
	// Get current version
	currentVersion, err := m.Version(ctx)
	if err != nil {
		return err
	}
	
	if currentVersion >= targetVersion {
		return fmt.Errorf("current version %d is already at or past target version %d", currentVersion, targetVersion)
	}
	
	// Run migrations up to target
	for _, migration := range m.migrations {
		if migration.Version <= currentVersion || migration.Version > targetVersion {
			continue
		}
		
		if err := m.runMigration(ctx, migration, "up"); err != nil {
			return err
		}
	}
	
	return nil
}

// DownTo rolls back to a specific version
func (m *BaseMigrator) DownTo(ctx context.Context, targetVersion int64) error {
	// Acquire lock
	if err := m.provider.AcquireLock(ctx, m.config.LockTimeout); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.provider.ReleaseLock(ctx)
	
	// Get current version
	currentVersion, err := m.Version(ctx)
	if err != nil {
		return err
	}
	
	if currentVersion <= targetVersion {
		return fmt.Errorf("current version %d is already at or below target version %d", currentVersion, targetVersion)
	}
	
	// Get migrations to rollback in reverse order
	var toRollback []Migration
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if m.migrations[i].Version > targetVersion && m.migrations[i].Version <= currentVersion {
			toRollback = append(toRollback, m.migrations[i])
		}
	}
	
	// Rollback migrations
	for _, migration := range toRollback {
		if err := m.runMigration(ctx, migration, "down"); err != nil {
			return err
		}
	}
	
	return nil
}

// Status returns the status of all migrations
func (m *BaseMigrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	// Get applied migrations
	applied, err := m.provider.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	appliedMap := make(map[int64]MigrationStatus)
	for _, status := range applied {
		appliedMap[status.Version] = status
	}
	
	// Build status list
	var statuses []MigrationStatus
	for _, migration := range m.migrations {
		status := MigrationStatus{
			Version:     migration.Version,
			Name:        migration.Name,
			Description: migration.Description,
			Checksum:    migration.Checksum,
			Applied:     false,
		}
		
		if applied, ok := appliedMap[migration.Version]; ok {
			status.Applied = true
			status.AppliedAt = applied.AppliedAt
			status.Duration = applied.Duration
		}
		
		statuses = append(statuses, status)
	}
	
	return statuses, nil
}

// Version returns the current migration version
func (m *BaseMigrator) Version(ctx context.Context) (int64, error) {
	applied, err := m.provider.GetAppliedMigrations(ctx)
	if err != nil {
		return 0, err
	}
	
	var maxVersion int64
	for _, status := range applied {
		if status.Version > maxVersion {
			maxVersion = status.Version
		}
	}
	
	return maxVersion, nil
}

// Validate checks for migration issues
func (m *BaseMigrator) Validate(ctx context.Context) error {
	var issues []ValidationIssue
	
	// Check for duplicate versions
	versionMap := make(map[int64]bool)
	for _, migration := range m.migrations {
		if versionMap[migration.Version] {
			issues = append(issues, ValidationIssue{
				Type:    "duplicate",
				Version: migration.Version,
				Message: fmt.Sprintf("duplicate migration version: %d", migration.Version),
			})
		}
		versionMap[migration.Version] = true
	}
	
	// Check for gaps in sequence (optional, might be too strict)
	for i := 1; i < len(m.migrations); i++ {
		// Just check that versions are in ascending order
		if m.migrations[i].Version <= m.migrations[i-1].Version {
			issues = append(issues, ValidationIssue{
				Type:    "sequence",
				Version: m.migrations[i].Version,
				Message: fmt.Sprintf("migration %d is out of sequence", m.migrations[i].Version),
			})
		}
	}
	
	// Validate checksums if enabled
	if m.config.ValidateChecksums {
		applied, err := m.provider.GetAppliedMigrations(ctx)
		if err != nil {
			return err
		}
		
		appliedMap := make(map[int64]string)
		for _, status := range applied {
			appliedMap[status.Version] = status.Checksum
		}
		
		for _, migration := range m.migrations {
			if checksum, ok := appliedMap[migration.Version]; ok {
				if checksum != migration.Checksum {
					issues = append(issues, ValidationIssue{
						Type:    "checksum",
						Version: migration.Version,
						Message: fmt.Sprintf("checksum mismatch for migration %d", migration.Version),
					})
				}
			}
		}
	}
	
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	
	return nil
}

// DryRun simulates migration execution without making changes
func (m *BaseMigrator) DryRun(ctx context.Context) ([]MigrationPlan, error) {
	// This would need to be implemented by specific providers
	// as SQL generation is database-specific
	return nil, fmt.Errorf("dry run not implemented")
}

// Force sets the migration version without running migrations
func (m *BaseMigrator) Force(ctx context.Context, version int64) error {
	// Acquire lock
	if err := m.provider.AcquireLock(ctx, m.config.LockTimeout); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.provider.ReleaseLock(ctx)
	
	// Find migration
	var migration *Migration
	for _, mig := range m.migrations {
		if mig.Version == version {
			migration = &mig
			break
		}
	}
	
	if migration == nil {
		return fmt.Errorf("migration %d not found", version)
	}
	
	// Record migration
	status := MigrationStatus{
		Version:     migration.Version,
		Name:        migration.Name,
		Description: migration.Description,
		Applied:     true,
		AppliedAt:   &[]time.Time{time.Now()}[0],
		Checksum:    migration.Checksum,
	}
	
	return m.provider.RecordMigration(ctx, status)
}

// Reset removes all migration records (dangerous!)
func (m *BaseMigrator) Reset(ctx context.Context) error {
	if m.config.RequireConfirmation {
		// In a real implementation, this would prompt for confirmation
		return fmt.Errorf("reset requires confirmation")
	}
	
	// Acquire lock
	if err := m.provider.AcquireLock(ctx, m.config.LockTimeout); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer m.provider.ReleaseLock(ctx)
	
	// Remove all migration records
	applied, err := m.provider.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}
	
	for _, status := range applied {
		if err := m.provider.RemoveMigration(ctx, status.Version); err != nil {
			return fmt.Errorf("failed to remove migration %d: %w", status.Version, err)
		}
	}
	
	return nil
}

// runMigration executes a single migration
func (m *BaseMigrator) runMigration(ctx context.Context, migration Migration, direction string) error {
	start := time.Now()
	
	fmt.Printf("Running migration %d: %s (%s)\n", migration.Version, migration.Name, direction)
	
	// Begin transaction
	tx, err := m.provider.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Execute migration
	var migrationErr error
	if direction == "up" {
		migrationErr = migration.Up(ctx, tx)
	} else {
		migrationErr = migration.Down(ctx, tx)
	}
	
	if migrationErr != nil {
		tx.Rollback()
		return &MigrationError{
			Version:   migration.Version,
			Direction: direction,
			Message:   fmt.Sprintf("migration %d failed: %v", migration.Version, migrationErr),
			Err:       migrationErr,
		}
	}
	
	// Record or remove migration record
	if direction == "up" {
		status := MigrationStatus{
			Version:     migration.Version,
			Name:        migration.Name,
			Description: migration.Description,
			Applied:     true,
			AppliedAt:   &[]time.Time{time.Now()}[0],
			Duration:    time.Since(start),
			Checksum:    migration.Checksum,
		}
		
		// Use the transaction to record migration
		if err := m.recordMigrationInTx(ctx, tx, status); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Remove migration record
		if err := m.removeMigrationInTx(ctx, tx, migration.Version); err != nil {
			tx.Rollback()
			return err
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	fmt.Printf("Migration %d completed successfully (%.3fs)\n", migration.Version, time.Since(start).Seconds())
	
	return nil
}

// recordMigrationInTx records a migration within a transaction
func (m *BaseMigrator) recordMigrationInTx(ctx context.Context, tx Transaction, status MigrationStatus) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (version, name, description, checksum, applied_at) VALUES (?, ?, ?, ?, ?)",
		m.config.TableName,
	)
	
	_, err := tx.Execute(ctx, query, status.Version, status.Name, status.Description, status.Checksum, status.AppliedAt)
	return err
}

// removeMigrationInTx removes a migration record within a transaction
func (m *BaseMigrator) removeMigrationInTx(ctx context.Context, tx Transaction, version int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE version = ?", m.config.TableName)
	_, err := tx.Execute(ctx, query, version)
	return err
}

// sortMigrations sorts migrations by version
func (m *BaseMigrator) sortMigrations() {
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

// calculateChecksum calculates a checksum for a migration
func (m *BaseMigrator) calculateChecksum(migration Migration) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d:%s:%s", migration.Version, migration.Name, migration.Description)))
	return fmt.Sprintf("%x", h.Sum(nil))
}