package main

import (
	"freelance-backend/db"
	"freelance-backend/handlers"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Loads .env only when it exists locally.
	// On Render, environment variables come from the Render dashboard.
	_ = godotenv.Load()

	conn, err := db.Connect()
	if err != nil {
		log.Fatal("Database conncection failed: ", err)
	}
	defer conn.Close()

	log.Println("Connected to PostgreSQL")

	// POST Client registration
	http.HandleFunc(
		"POST /clients",
		handlers.RegisterClient(conn),
	)

	// POST Freelancer registration
	http.HandleFunc(
		"POST /freelancers",
		handlers.RegisterFreelancer(conn),
	)

	http.HandleFunc(
		"POST /login",
		handlers.Login(conn, []byte(os.Getenv("JWT_SECRET"))),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
