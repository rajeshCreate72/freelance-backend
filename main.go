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
	// Load .env when running locally.
	// Production environment variables are configured on Render.
	_ = godotenv.Load()

	conn, err := db.Connect()
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	defer conn.Close()

	log.Println("Connected to PostgreSQL")

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET is required")
	}

	frontendOrigin := os.Getenv("FRONTEND_ORIGINS")
	if frontendOrigin == "" {
		log.Fatal("FRONTEND_ORIGINS are required")
	}

	// Middleware
	auth := middleware.Authenticate(jwtSecret)

	// Public routes
	// These routes do not require authentication.

	http.HandleFunc(
		"POST /clients",
		handlers.RegisterClient(conn),
	)

	http.HandleFunc(
		"POST /freelancers",
		handlers.RegisterFreelancer(conn),
	)

	http.HandleFunc(
		"POST /login",
		handlers.Login(conn, jwtSecret),
	)

	http.HandleFunc(
		"POST /refresh",
		handlers.Refresh(conn, jwtSecret),
	)

	http.HandleFunc(
		"GET /jobs",
		handlers.ListJobs(conn),
	)

	http.HandleFunc(
		"GET /jobs/{id}",
		handlers.GetJob(conn),
	)

	http.HandleFunc(
		"GET /freelancers/{freelancer_id}/reviews",
		handlers.ListFreelancerReviews(conn),
	)

	http.HandleFunc(
		"GET /clients/{client_id}/reviews",
		handlers.ListClientReviews(conn),
	)

	// Client routes
	// Authentication is required.
	// Role authorization should also require the "client" role.

	http.Handle(
		"GET /clients/me",
		auth(middleware.RequireRole("client")(handlers.GetClientMe(conn))),
	)

	http.Handle(
		"GET /clients/me/jobs",
		auth(middleware.RequireRole("client")(handlers.ListMyJobs(conn))),
	)

	http.Handle(
		"POST /jobs",
		auth(middleware.RequireRole("client")(handlers.CreateJob(conn))),
	)

	http.Handle(
		"POST /jobs/{job_id}/assignments",
		auth(middleware.RequireRole("client")(handlers.AssignFreelancer(conn))),
	)

	http.Handle(
		"GET /jobs/{job_id}/applications",
		auth(middleware.RequireRole("client")(handlers.ListJobApplications(conn))),
	)

	// Belongs to both client and freelancer
	http.Handle(
		"POST /jobs/{job_id}/reviews",
		auth(handlers.CreateReview(conn)),
	)

	// Freelancer routes
	// Authentication is required.
	// Role authorization should also require the "freelancer" role.

	http.Handle(
		"GET /freelancer/me",
		auth(middleware.RequireRole("freelancer")(handlers.GetFreelancerMe(conn))),
	)

	http.Handle(
		"GET /freelancers/me/applications",
		auth(middleware.RequireRole("freelancer")(handlers.ListMyApplications(conn))),
	)

	http.Handle(
		"POST /jobs/{job_id}/applications",
		auth(middleware.RequireRole("freelancer")(handlers.ApplyToJob(conn))),
	)

	http.Handle(
		"POST /jobs/{job_id}/assignment/accept",
		auth(middleware.RequireRole("freelancer")(handlers.AcceptAssignment(conn))),
	)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)

	if err := http.ListenAndServe(
		":"+port,
		middleware.CORS(frontendOrigin, http.DefaultServeMux),
	); err != nil {
		log.Fatal(err)
	}
}