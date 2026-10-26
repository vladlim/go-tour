package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"dbreplication/internal/shards"
	"dbreplication/internal/store"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx := context.Background()
	m, err := shards.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()
	s := store.New(m)
	switch os.Args[1] {
	case "create-user":
		createUser(ctx, s, os.Args[2:])
	case "get-user":
		getUser(ctx, s, os.Args[2:])
	case "list-users":
		listUsers(ctx, s)
	case "status":
		status(ctx, s)
	default:
		usage()
		os.Exit(2)
	}
}

func createUser(ctx context.Context, s *store.Store, args []string) {
	f := flag.NewFlagSet("create-user", flag.ExitOnError)
	id := f.Int64("id", 0, "shard key")
	name := f.String("name", "", "name")
	email := f.String("email", "", "email")
	_ = f.Parse(args)
	if *id <= 0 || *name == "" || *email == "" {
		log.Fatal("id, name and email are required")
	}
	u, err := s.CreateUser(ctx, *id, *name, *email)
	if err != nil {
		log.Fatal(err)
	}
	printUser(u)
}
func getUser(ctx context.Context, s *store.Store, args []string) {
	f := flag.NewFlagSet("get-user", flag.ExitOnError)
	primary := f.Bool("primary", false, "read from primary")
	_ = f.Parse(args)
	if f.NArg() != 1 {
		log.Fatal("user id is required")
	}
	id, err := strconv.ParseInt(f.Arg(0), 10, 64)
	if err != nil || id <= 0 {
		log.Fatal("user id must be a positive integer")
	}
	u, err := s.GetUser(ctx, id, *primary)
	if err != nil {
		log.Fatal(err)
	}
	printUser(u)
}
func listUsers(ctx context.Context, s *store.Store) {
	users, err := s.AllUsers(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, user := range users {
		printUser(user)
	}
}
func status(ctx context.Context, s *store.Store) {
	statuses, err := s.Status(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, st := range statuses {
		lag := "no transaction replayed yet"
		if st.ReplicationLag != nil {
			lag = st.ReplicationLag.String()
		}
		fmt.Printf("shard=%d primary_in_recovery=%t replica_in_recovery=%t replay_lag=%s\n", st.Shard, st.PrimaryRecovery, st.ReplicaRecovery, lag)
	}
}
func printUser(u store.User) {
	fmt.Printf("id=%d name=%q email=%q shard=%d source=%s created=%s\n", u.ID, u.Name, u.Email, u.Shard, u.Source, u.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00"))
}
func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  go run ./cmd create-user -id 1 -name Alice -email alice@example.com
  go run ./cmd get-user [--primary] 1
  go run ./cmd list-users
  go run ./cmd status`)
}
