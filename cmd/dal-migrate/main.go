package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	
	"github.com/airoles/go-dal/migration"
	"github.com/airoles/go-dal/migration/providers/sqlite"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	tableName string
	dsn       string
	driver    string
	
	rootCmd = &cobra.Command{
		Use:   "dal-migrate",
		Short: "Database migration tool",
		Long:  "A flexible database migration tool for managing schema changes",
	}
	
	upCmd = &cobra.Command{
		Use:   "up",
		Short: "Run pending migrations",
		RunE:  runUp,
	}
	
	downCmd = &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE:  runDown,
	}
	
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE:  runStatus,
	}
	
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show current migration version",
		RunE:  runVersion,
	}
	
	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate migrations",
		RunE:  runValidate,
	}
	
	forceCmd = &cobra.Command{
		Use:   "force VERSION",
		Short: "Force set migration version",
		Args:  cobra.ExactArgs(1),
		RunE:  runForce,
	}
	
	createCmd = &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new migration",
		Args:  cobra.ExactArgs(1),
		RunE:  runCreate,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .migrate.yml)")
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", "", "database connection string")
	rootCmd.PersistentFlags().StringVar(&driver, "driver", "sqlite3", "database driver (sqlite3, postgres, mysql)")
	rootCmd.PersistentFlags().StringVar(&tableName, "table", "schema_migrations", "migrations table name")
	
	// Bind flags to viper
	viper.BindPFlag("database.dsn", rootCmd.PersistentFlags().Lookup("dsn"))
	viper.BindPFlag("database.driver", rootCmd.PersistentFlags().Lookup("driver"))
	viper.BindPFlag("migrations.table", rootCmd.PersistentFlags().Lookup("table"))
	
	// Add commands
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(forceCmd)
	rootCmd.AddCommand(createCmd)
	
	// Up command flags
	upCmd.Flags().Int64("to", 0, "migrate up to specific version")
	
	// Down command flags
	downCmd.Flags().Int64("to", 0, "rollback to specific version")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".migrate")
		viper.SetConfigType("yml")
		viper.AddConfigPath(".")
	}
	
	viper.SetEnvPrefix("MIGRATE")
	viper.AutomaticEnv()
	
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getMigrator() (migration.Migrator, error) {
	// Get configuration
	dsn := viper.GetString("database.dsn")
	if dsn == "" {
		return nil, fmt.Errorf("database DSN not configured")
	}
	
	driver := viper.GetString("database.driver")
	tableName := viper.GetString("migrations.table")
	
	// Create provider based on driver
	var provider migration.Provider
	switch driver {
	case "sqlite3", "sqlite":
		provider = sqlite.NewProvider(dsn, tableName)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
	
	// Connect to database
	ctx := context.Background()
	if err := provider.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Create migrator
	config := migration.Config{
		TableName:         tableName,
		ValidateChecksums: viper.GetBool("options.validate_checksums"),
		AutoRollback:      viper.GetBool("options.auto_rollback"),
	}
	
	migrator := migration.NewBaseMigrator(provider, config)
	
	// Load migrations (this would be customized per project)
	if err := loadMigrations(migrator); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}
	
	return migrator, nil
}

func loadMigrations(migrator migration.Migrator) error {
	// This is a placeholder - in real use, this would be implemented
	// by the project using the migration tool
	fmt.Println("Loading migrations from filesystem...")
	
	// Look for migrations in ./migrations directory
	migrationsPath := viper.GetString("migrations.path")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}
	
	// Check if directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found: %s", migrationsPath)
	}
	
	// Note: Actual migration loading would be implemented here
	// This would read .sql files and create Migration structs
	
	return nil
}

func runUp(cmd *cobra.Command, args []string) error {
	migrator, err := getMigrator()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	
	// Check if migrating to specific version
	if to, _ := cmd.Flags().GetInt64("to"); to > 0 {
		return migrator.UpTo(ctx, to)
	}
	
	return migrator.Up(ctx)
}

func runDown(cmd *cobra.Command, args []string) error {
	migrator, err := getMigrator()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	
	// Check if rolling back to specific version
	if to, _ := cmd.Flags().GetInt64("to"); to >= 0 {
		return migrator.DownTo(ctx, to)
	}
	
	return migrator.Down(ctx)
}

func runStatus(cmd *cobra.Command, args []string) error {
	migrator, err := getMigrator()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	statuses, err := migrator.Status(ctx)
	if err != nil {
		return err
	}
	
	// Print status table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "VERSION\tNAME\tAPPLIED\tAPPLIED AT")
	fmt.Fprintln(w, "-------\t----\t-------\t----------")
	
	for _, status := range statuses {
		applied := "No"
		appliedAt := "-"
		
		if status.Applied {
			applied = "Yes"
			if status.AppliedAt != nil {
				appliedAt = status.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}
		
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", status.Version, status.Name, applied, appliedAt)
	}
	
	w.Flush()
	return nil
}

func runVersion(cmd *cobra.Command, args []string) error {
	migrator, err := getMigrator()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	version, err := migrator.Version(ctx)
	if err != nil {
		return err
	}
	
	if version == 0 {
		fmt.Println("No migrations applied")
	} else {
		fmt.Printf("Current version: %d\n", version)
	}
	
	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	migrator, err := getMigrator()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	if err := migrator.Validate(ctx); err != nil {
		if validationErr, ok := err.(*migration.ValidationError); ok {
			fmt.Println("Validation failed:")
			for _, issue := range validationErr.Issues {
				fmt.Printf("  - [%s] Version %d: %s\n", issue.Type, issue.Version, issue.Message)
			}
			return fmt.Errorf("validation failed with %d issues", len(validationErr.Issues))
		}
		return err
	}
	
	fmt.Println("✓ All migrations valid")
	return nil
}

func runForce(cmd *cobra.Command, args []string) error {
	version, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid version: %s", args[0])
	}
	
	migrator, err := getMigrator()
	if err != nil {
		return err
	}
	
	ctx := context.Background()
	if err := migrator.Force(ctx, version); err != nil {
		return err
	}
	
	fmt.Printf("Forced migration version to %d\n", version)
	return nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	// Generate timestamp-based version
	version := time.Now().Format("20060102150405")
	
	// Sanitize name for filename
	safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	safeName = strings.ReplaceAll(safeName, "-", "_")
	
	// Get migrations path
	migrationsPath := viper.GetString("migrations.path")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}
	
	// Create migrations directory if it doesn't exist
	if err := os.MkdirAll(migrationsPath, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}
	
	// Create up and down migration files
	upFile := filepath.Join(migrationsPath, fmt.Sprintf("%s_%s.up.sql", version, safeName))
	downFile := filepath.Join(migrationsPath, fmt.Sprintf("%s_%s.down.sql", version, safeName))
	
	// Create up file
	upContent := fmt.Sprintf(`-- Migration: %s
-- Description: %s
-- Version: %s

-- Add your UP migration SQL here

`, name, name, version)
	
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to create up migration: %w", err)
	}
	
	// Create down file
	downContent := fmt.Sprintf(`-- Migration: %s (rollback)
-- Description: Rollback %s
-- Version: %s

-- Add your DOWN migration SQL here

`, name, name, version)
	
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to create down migration: %w", err)
	}
	
	fmt.Printf("Created migration files:\n")
	fmt.Printf("  - %s\n", upFile)
	fmt.Printf("  - %s\n", downFile)
	
	return nil
}