package main

import (
    "crypto/sha256"
    "context"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "text/tabwriter"
    "time"

    "github.com/gnemade360/go-dal/migration"
    "github.com/gnemade360/go-dal/migration/providers/sqlite"
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
	rootCmd.PersistentFlags().String("migrations", "./migrations", "path to SQL migrations directory")
	rootCmd.PersistentFlags().Bool("forward-only", false, "disable rollbacks (down/downTo)")
	rootCmd.PersistentFlags().Bool("require-down", true, "error if a down script is missing when rolling back")
    
    // Bind flags to viper
    viper.BindPFlag("database.dsn", rootCmd.PersistentFlags().Lookup("dsn"))
    viper.BindPFlag("database.driver", rootCmd.PersistentFlags().Lookup("driver"))
	viper.BindPFlag("migrations.table", rootCmd.PersistentFlags().Lookup("table"))
	viper.BindPFlag("migrations.path", rootCmd.PersistentFlags().Lookup("migrations"))
	viper.BindPFlag("migrations.forward_only", rootCmd.PersistentFlags().Lookup("forward-only"))
	viper.BindPFlag("migrations.require_down", rootCmd.PersistentFlags().Lookup("require-down"))
	
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
    fmt.Println("Loading migrations from filesystem...")

    // Get configured path
    migrationsPath := viper.GetString("migrations.path")
    if migrationsPath == "" {
        migrationsPath = "./migrations"
    }

    info, err := os.Stat(migrationsPath)
    if err != nil || !info.IsDir() {
        return fmt.Errorf("migrations directory not found: %s", migrationsPath)
    }

    // Match files:
    // - Timestamp style: 20060102150405_name.up.sql / .down.sql
    // - Flyway style: V<ver>__name.sql (up), U<ver>__name.sql (down)
    reTs := regexp.MustCompile(`^(?P<ver>\d{14})_(?P<name>[A-Za-z0-9_]+)\.(?P<dir>up|down)\.sql$`)
    reFlyUp := regexp.MustCompile(`^V(?P<ver>\d+)__(?P<name>[A-Za-z0-9_]+)\.sql$`)
    reFlyDown := regexp.MustCompile(`^U(?P<ver>\d+)__(?P<name>[A-Za-z0-9_]+)\.sql$`)

    type pair struct {
        name    string
        upFile  string
        downFile string
    }
    pairs := map[int64]*pair{}

    entries, err := os.ReadDir(migrationsPath)
    if err != nil {
        return fmt.Errorf("failed to read migrations dir: %w", err)
    }

    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        fname := e.Name()
        var versionStr, migName, dir string
        matched := false

        if m := reTs.FindStringSubmatch(fname); m != nil {
            versionStr = m[reTs.SubexpIndex("ver")]
            migName = m[reTs.SubexpIndex("name")]
            dir = m[reTs.SubexpIndex("dir")]
            matched = true
        } else if m := reFlyUp.FindStringSubmatch(fname); m != nil {
            versionStr = m[reFlyUp.SubexpIndex("ver")]
            migName = m[reFlyUp.SubexpIndex("name")]
            dir = "up"
            matched = true
        } else if m := reFlyDown.FindStringSubmatch(fname); m != nil {
            versionStr = m[reFlyDown.SubexpIndex("ver")]
            migName = m[reFlyDown.SubexpIndex("name")]
            dir = "down"
            matched = true
        }
        if !matched {
            continue
        }

        ver, err := strconv.ParseInt(versionStr, 10, 64)
        if err != nil {
            return fmt.Errorf("invalid migration version in %s: %w", fname, err)
        }

        p, ok := pairs[ver]
        if !ok {
            p = &pair{name: migName}
            pairs[ver] = p
        }
        path := filepath.Join(migrationsPath, fname)
        if dir == "up" {
            if p.upFile != "" {
                return fmt.Errorf("duplicate up migration for version %d (%s)", ver, migName)
            }
            p.upFile = path
        } else {
            if p.downFile != "" {
                return fmt.Errorf("duplicate down migration for version %d (%s)", ver, migName)
            }
            p.downFile = path
        }
    }

    // Sort versions for deterministic registration
    var versions []int64
    for v := range pairs {
        versions = append(versions, v)
    }
    sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

    // Register migrations
    for _, v := range versions {
        p := pairs[v]
        if p.upFile == "" {
            return fmt.Errorf("missing .up.sql for version %d (%s)", v, p.name)
        }

        upPath := p.upFile
        downPath := p.downFile // may be empty

        // Capture for closures
        name := strings.ReplaceAll(p.name, "_", " ")

        // Precompute checksum from up script content
        upContent, err := os.ReadFile(upPath)
        if err != nil {
            return fmt.Errorf("read up file for checksum: %w", err)
        }
        sum := sha256.Sum256(upContent)

        mig := migration.Migration{
            Version:     v,
            Name:        name,
            Description: "",
            Checksum:    fmt.Sprintf("%x", sum[:]),
            Up: func(ctx context.Context, tx migration.Transaction) error {
                sqlText, err := os.ReadFile(upPath)
                if err != nil {
                    return fmt.Errorf("read up file: %w", err)
                }
                return execSQLScript(ctx, tx, string(sqlText))
            },
            Down: func(ctx context.Context, tx migration.Transaction) error {
                if downPath == "" {
                    if viper.GetBool("migrations.require_down") {
                        return fmt.Errorf("missing down script for version %d (%s)", v, p.name)
                    }
                    // No-op if not required
                    return nil
                }
                sqlText, err := os.ReadFile(downPath)
                if err != nil {
                    return fmt.Errorf("read down file: %w", err)
                }
                return execSQLScript(ctx, tx, string(sqlText))
            },
        }

        migrator.AddMigration(mig)
    }

    return nil
}

// execSQLScript executes a SQL script by splitting statements safely on semicolons.
func execSQLScript(ctx context.Context, tx migration.Transaction, script string) error {
    statements := splitSQLStatements(script)
    for _, s := range statements {
        if strings.TrimSpace(s) == "" {
            continue
        }
        if _, err := tx.Execute(ctx, s); err != nil {
            return err
        }
    }
    return nil
}

// splitSQLStatements splits a SQL string into statements, respecting quotes and comments.
func splitSQLStatements(sql string) []string {
    var out []string
    var buf strings.Builder
    inSingle := false
    inDouble := false
    inLineComment := false
    inBlockComment := false

    for i := 0; i < len(sql); i++ {
        c := sql[i]
        next := byte(0)
        if i+1 < len(sql) {
            next = sql[i+1]
        }

        // Handle line comment end
        if inLineComment {
            if c == '\n' {
                inLineComment = false
                buf.WriteByte(c)
            }
            continue
        }
        // Handle block comment end
        if inBlockComment {
            if c == '*' && next == '/' {
                inBlockComment = false
                i++ // consume '/'
            }
            continue
        }

        // Enter comments (when not in quotes)
        if !inSingle && !inDouble {
            if c == '-' && next == '-' {
                inLineComment = true
                i++ // consume second '-'
                continue
            }
            if c == '/' && next == '*' {
                inBlockComment = true
                i++ // consume '*'
                continue
            }
        }

        // Toggle quotes
        if !inDouble && c == '\'' {
            inSingle = !inSingle
            buf.WriteByte(c)
            continue
        }
        if !inSingle && c == '"' {
            inDouble = !inDouble
            buf.WriteByte(c)
            continue
        }

        // Split on semicolon when not inside quotes/comments
        if c == ';' && !inSingle && !inDouble {
            out = append(out, buf.String())
            buf.Reset()
            continue
        }

        buf.WriteByte(c)
    }
    if strings.TrimSpace(buf.String()) != "" {
        out = append(out, buf.String())
    }
    return out
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
    if viper.GetBool("migrations.forward_only") {
        return fmt.Errorf("down is disabled in forward-only mode")
    }
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
