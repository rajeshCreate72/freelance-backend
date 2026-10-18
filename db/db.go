package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	// AWS SDK v2 packages for loading configuration and generating IAM database tokens
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	rdsauth "github.com/aws/aws-sdk-go-v2/feature/rds/auth"

	// pgx driver packages to connect to PostgreSQL/Aurora using the standard database/sql library
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Connect initializes and returns a pool of database connections (*sql.DB).
// Connecting to render - deployment service
// It dynamically supports both standard password authentication and AWS IAM database token authentication.
// 1. Building postgres connection url.
// 2. Connection to database in aws or local-database according to .env
// 3. verifying connection
func Connect() (*sql.DB, error) {
	// Connecting to render deployment service
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		connConfig, err := pgx.ParseConfig(databaseURL)
		if err != nil {
			return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
		}

		db := stdlib.OpenDB(*connConfig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("ping PostgreSQL: %w", err)
		}

		return db, nil
	}
	// --- 1. FETCH MANDATORY ENVIRONMENT VARIABLES ---
	// Retrieve basic database endpoint details. Error out if any are missing.
	host, err := requiredEnv("DB_HOST")
	if err != nil {
		return nil, err
	}

	port, err := requiredEnv("DB_PORT")
	if err != nil {
		return nil, err
	}

	user, err := requiredEnv("DB_USER")
	if err != nil {
		return nil, err
	}

	dbName, err := requiredEnv("DB_NAME")
	if err != nil {
		return nil, err
	}

	// --- 2. DETERMINE AUTHENTICATION & SECURITY STRATEGY ---
	// Check if the application should use AWS IAM database authentication (defaults to false)
	useIAM, err := strconv.ParseBool(envOrDefault("USE_IAM_DB_AUTH", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid USE_IAM_DB_AUTH: %w", err)
	}

	// Set the SSL mode. AWS IAM authentication strictly requires an encrypted SSL connection.
	sslMode := os.Getenv("DB_SSL_MODE")
	if sslMode == "" {
		if useIAM {
			sslMode = "require" // Enforce SSL if using IAM auth
		} else {
			sslMode = "disable" // Default to no SSL for standard local/development environments
		}
	}

	// --- 3. CONSTRUCT THE DATABASE CONNECTION STRING (DSN) ---
	// Build a valid PostgreSQL connection URL structured as: postgres://user@host:port/dbname
	dsnURL := &url.URL{
		Scheme: "postgres",
		User:   url.User(user),
		Host:   net.JoinHostPort(host, port), // Combines host and port safely (e.g., "localhost:5432")
		Path:   "/" + dbName,
	}

	// Add query parameters for SSL configuration to the connection URL
	query := dsnURL.Query()
	query.Set("sslmode", sslMode)

	// If a custom CA certificate root path is provided (common for RDS/Aurora verification), add it
	if rootCert := os.Getenv("DB_SSL_ROOT_CERT"); rootCert != "" {
		query.Set("sslrootcert", rootCert)
	}

	dsnURL.RawQuery = query.Encode()

	// Parse the constructed URL string into a pgx configuration struct
	connConfig, err := pgx.ParseConfig(dsnURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}

	var db *sql.DB

	// --- 4. CONDITIONAL AUTHENTICATION LOGIC ---
	if useIAM {
		// --- BRANCH A: AWS IAM TOKEN AUTHENTICATION ---
		// IAM authentication requires a target AWS Region (e.g., "us-east-1")
		region, err := requiredEnv("AWS_REGION")
		if err != nil {
			return nil, err
		}

		// Load standard AWS credentials (IAM Roles, ENV variables, or ~/.aws/credentials)
		awsCfg, err := awsConfig.LoadDefaultConfig(
			context.Background(),
			awsConfig.WithRegion(region),
		)
		if err != nil {
			return nil, fmt.Errorf("load AWS configuration %w", err)
		}

		endpoint := net.JoinHostPort(host, port)

		// Open the database using pgx stdlib. 
		// Critically, OptionBeforeConnect runs a hook right before a NEW physical connection is opened.
		// Since IAM tokens expire after 15 minutes, this hook generates a fresh token on-demand.
		db = stdlib.OpenDB(
			*connConfig,
			stdlib.OptionBeforeConnect(func(ctx context.Context, cfg *pgx.ConnConfig) error {
				// Generate a temporary 15-minute authentication token from AWS
				token, err := rdsauth.BuildAuthToken(
					ctx,
					endpoint,
					region,
					user,
					awsCfg.Credentials,
				)
				if err != nil {
					return fmt.Errorf("generate Aurora IAM authentication token: %w", err)
				}

				// Inject the generated token into the password field for this connection attempt
				cfg.Password = token
				return nil
			}),
		)
	} else {
		// --- BRANCH B: TRADITIONAL PASSWORD AUTHENTICATION ---
		// Fetch the standard plaintext password from environment variables
		password, err := requiredEnv("DB_PASSWORD")
		if err != nil {
			return nil, err
		}

		// Inject the password permanently into the configuration and open the database connection pool
		connConfig.Password = password
		db = stdlib.OpenDB(*connConfig)
	}

	// --- 5. VERIFY CONNECTION HEALTH (PING) ---
	// Create a short 10-second context timeout to ensure the application doesn't hang forever if the DB is down
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Physically ping the database to verify credentials, network paths, and SSL handshakes are successful
	if err := db.PingContext(ctx); err != nil {
		db.Close() // Clean up resources if verification fails
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Return the healthy, established connection pool
	return db, nil
}

// requiredEnv retrieves an environment variable. 
// If it is completely missing or just empty whitespace, it returns an error.
func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))

	if value == "" {
		// Note: There is a tiny typo here in your original code ("% must be set"). It should be "%s must be set"
		return "", fmt.Errorf("%s must be set", name)
	}

	return value, nil
}

// envOrDefault attempts to retrieve an environment variable.
// If it doesn't exist, it returns a provided default fallback value instead.
func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	return value
}
