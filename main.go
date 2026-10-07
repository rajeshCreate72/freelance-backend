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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}

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

	// Connection to database
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
