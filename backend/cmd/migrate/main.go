package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"scoring-backend/internal/store"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID is required")
	}

	databaseID := os.Getenv("FIRESTORE_DATABASE")

	ctx := context.Background()

	// Open source (file store)
	src, err := store.NewFileStore(dataDir)
	if err != nil {
		log.Fatalf("Failed to open file store: %v", err)
	}

	// Open destination (firestore)
	dst, err := store.NewFirestoreStore(ctx, projectID, databaseID)
	if err != nil {
		log.Fatalf("Failed to open firestore store: %v", err)
	}
	defer dst.Close()

	dbName := databaseID
	if dbName == "" {
		dbName = "(default)"
	}
	fmt.Printf("Migrating from %s -> Firestore (project: %s, database: %s)\n\n", dataDir, projectID, dbName)

	// Migrate tournaments — write directly to preserve original timestamps
	tournaments, err := src.ListTournaments(ctx)
	if err != nil {
		log.Fatalf("Failed to list tournaments: %v", err)
	}
	fmt.Printf("Tournaments: %d\n", len(tournaments))
	for _, t := range tournaments {
		totalMatches := 0
		for _, r := range t.Rounds {
			totalMatches += len(r.Matches)
		}
		fmt.Printf("  %s (%s)\n", t.Name, t.ID)
		fmt.Printf("    Created: %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Teams: %s vs %s\n", t.Teams[0].Name, t.Teams[1].Name)
		fmt.Printf("    Players: %d + %d\n", len(t.Teams[0].Players), len(t.Teams[1].Players))
		fmt.Printf("    Rounds: %d, Matches: %d\n", len(t.Rounds), totalMatches)
		for _, r := range t.Rounds {
			fmt.Printf("      Round %d (%s): %d matches\n", r.Number, r.Name, len(r.Matches))
		}
		if err := dst.ImportTournament(ctx, t); err != nil {
			fmt.Printf("    SKIP: %v\n", err)
			continue
		}
		fmt.Printf("    OK\n")
	}

	// Migrate registered users
	users, err := src.ListRegisteredUsers(ctx)
	if err != nil {
		log.Fatalf("Failed to list registered users: %v", err)
	}
	fmt.Printf("\nRegistered users: %d\n", len(users))
	for _, u := range users {
		fmt.Printf("  %s <%s>\n", u.Name, u.Email)
		if err := dst.RegisterUser(ctx, u); err != nil {
			fmt.Printf("    SKIP: %v\n", err)
			continue
		}
		fmt.Printf("    OK\n")
	}

	// Migrate local users
	localUsers, err := src.ListLocalUsers(ctx)
	if err != nil {
		log.Fatalf("Failed to list local users: %v", err)
	}
	fmt.Printf("\nLocal users: %d\n", len(localUsers))
	for _, u := range localUsers {
		status := "unverified"
		if u.EmailVerified && u.Confirmed {
			status = "active"
		} else if u.EmailVerified {
			status = "pending approval"
		}
		fmt.Printf("  %s <%s> [%s]\n", u.Name, u.Email, status)
		if err := dst.CreateLocalUser(ctx, u); err != nil {
			fmt.Printf("    SKIP: %v\n", err)
			continue
		}
		fmt.Printf("    OK\n")
	}

	fmt.Printf("\nDone. Migrated %d tournament(s), %d registered user(s), %d local user(s).\n",
		len(tournaments), len(users), len(localUsers))
}
