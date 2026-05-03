package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("ALPHA300_DB_USER"),
		os.Getenv("ALPHA300_DB_PASSWORD"),
		os.Getenv("ALPHA300_DB_HOST"),
		os.Getenv("ALPHA300_DB_PORT"),
		os.Getenv("ALPHA300_DB_NAME"),
		os.Getenv("ALPHA300_DB_SSL_MODE"))

	fmt.Printf("Connecting to: %s\n", url)

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		fmt.Printf("Error creating pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test ping
	if err := pool.Ping(context.Background()); err != nil {
		fmt.Printf("Error pinging: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Ping successful")

	// Test query
	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM daily").Scan(&count)
	if err != nil {
		fmt.Printf("Error querying: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Daily table has %d rows\n", count)

	// Test kline query
	rows, err := pool.Query(context.Background(),
		"SELECT trade_date, close FROM daily WHERE ts_code = $1 ORDER BY trade_date DESC LIMIT 5",
		"605117.SH")
	if err != nil {
		fmt.Printf("Error querying kline: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Println("\nLatest 5 klines for 605117.SH:")
	for rows.Next() {
		var date string
		var close float64
		if err := rows.Scan(&date, &close); err != nil {
			fmt.Printf("Error scanning: %v\n", err)
			continue
		}
		fmt.Printf("  %s: %.2f\n", date, close)
	}

	fmt.Println("\n✓ All tests passed!")
}