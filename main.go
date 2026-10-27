package main

import (
	"freelance-backend/db"
	"freelance-backend/handlers"
	"freelance-backend/middleware"
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

	auth := middleware.Authenticate([]byte(os.Getenv("JWT_SECRET")))

	frontendOrigin := os.Getenv("FRONTEND_ORIGINS")
	if frontendOrigin == "" {
		log.Fatal("FRONTEND_ORIGINS are required")
	}

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

	http.HandleFunc(
		"GET /jobs",
		handlers.ListJobs(conn),
	)

	http.HandleFunc("GET /jobs/{id}", handlers.GetJob(conn))

	http.HandleFunc(
		"GET /freelancers/{freelancer_id}/reviews",
		handlers.ListFreelancerReviews(conn),
	)

	http.HandleFunc(
		"GET /clients/{client_id}/reviews",
		handlers.ListClientReviews(conn),
	)

	// PROTECTED ROUTES
	http.Handle(
		"GET /clients/me",
		auth(handlers.GetClientMe(conn)),
	)

	http.Handle(
		"GET /freelancer/me",
		auth(handlers.GetFreelancerMe(conn)),
	)

	http.Handle(
		"POST /jobs",
		auth(handlers.CreateJob(conn)),
	)

	http.Handle(
		"POST /jobs/{job_id}/applications",
		auth(handlers.ApplyToJob(conn)),
	)

	http.Handle(
		"GET /jobs/{job_id}/applications",
		auth(handlers.ListJobApplications(conn)),
	)

	http.Handle(
		"GET /freelancers/me/applications",
		auth(handlers.ListMyApplications(conn)),
	)

	http.Handle(
		"GET /clients/me/jobs",
		auth(handlers.ListMyJobs(conn)),
	)

	http.Handle(
		"POST /jobs/{job_id}/assignments",
		auth(handlers.AssignFreelancer(conn)),
	)

	http.Handle(
		"POST /jobs/{job_id}/assignment/accept",
		auth(handlers.AcceptAssignment(conn)),
	)

	http.Handle(
		"POST /jobs/{job_id}/reviews",
		auth(handlers.CreateReview(conn)),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)

	if err := http.ListenAndServe(
		":"+port,
		middleware.CORS(frontendOrigin, http.DefaultServeMux),
	); err !=  nil {
		log.Fatal(err)
	}
}
