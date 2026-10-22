package db

import (
	"context"
	"database/sql"
	"time"
)

type Review struct {
	ID           string    `json:"id"`
	JobID        string    `json:"job_id"`
	ReviewerRole string    `json:"reviewer_role"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

// ReviewListItem is a review shown on a client or freelancer's profile.
type ReviewListItem struct {
	ID               string    `json:"id"`
	JobID            string    `json:"job_id"`
	ReviewerID       string    `json:"reviewer_id"`
	ReviewerRole     string    `json:"reviewer_role"`
	ReviewerUsername string    `json:"reviewer_username"`
	Rating           int       `json:"rating"`
	Comment          string    `json:"comment"`
	CreatedAt        time.Time `json:"created_at"`
}

// CreateReview derives reviewer and reviewee from the authenticated
// account, job and assignment. The caller cannot choose IDs
func CreateReview(
	ctx context.Context,
	conn *sql.DB,
	jobID string,
	reviewerID string,
	reviewerRole string,
	rating int,
	comment string,
) (Review, error) {
	const query = `
		INSERT INTO reviews (job_id, reviewer_role, rating, comment)
		SELECT j.id, $3::user_role, $4, $5
		FROM jobs AS j
		JOIN job_assignments AS ja ON ja.job_id = j.id
		JOIN assignment_statuses AS ast ON ast.id = ja.status_id
		WHERE j.id = $1
		  AND ast.name = 'Accepted'
		  AND (
				($3::user_role = 'client' AND j.client_id = $2)
				OR
				($3::user_role = 'freelancer' AND ja.freelancer_id = $2)
		  )
		RETURNING
			id::text,
			job_id::text,
			reviewer_role::text,
			rating,
			comment,
			created_at
	`

	var review Review
	err := conn.QueryRowContext(
		ctx,
		query,
		jobID,
		reviewerID,
		reviewerRole,
		rating,
		comment,
	).Scan(
		&review.ID,
		&review.JobID,
		&review.ReviewerRole,
		&review.Rating,
		&review.Comment,
		&review.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return Review{}, ErrReviewNotAllowed
	}
	if err != nil {
		if isUniqueViolation(err) {
			return Review{}, ErrReviewAlreadySubmitted
		}
		return Review{}, err
	}
	return review, nil
}

// ListFreelancerReviews returns reviews written by clients
// about the specified freelancer.
func ListFreelancerReviews(
	ctx context.Context,
	conn *sql.DB,
	freelancerID string,
) ([]ReviewListItem, error) {
	const query = `
		SELECT
			r.id::text,
			r.job_id::text,
			j.client_id::text,
			r.reviewer_role::text,
			c.username,
			r.rating,
			r.comment,
			r.created_at
		FROM reviews AS r
		JOIN jobs AS j ON j.id = r.job_id
		JOIN job_assignments AS ja ON ja.job_id = j.id
		JOIN clients AS c ON c.id = j.client_id
		WHERE ja.freelancer_id = $1
		  AND r.reviewer_role = 'client'
		ORDER BY r.created_at DESC
	`
	
	rows, err := conn.QueryContext(ctx, query, freelancerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]ReviewListItem, 0)

	for rows.Next(){
		var review ReviewListItem
		if err := rows.Scan(
			&review.ID,
			&review.JobID,
			&review.ReviewerID,
			&review.ReviewerRole,
			&review.ReviewerUsername,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}

// ListClientReviews return reviews written by freelancers
// about the specified client
func ListClientReviews(
	ctx context.Context,
	conn *sql.DB,
	clientID string,
) ([]ReviewListItem, error) {
	const query = `
		SELECT
			r.id::text,
			r.job_id::text,
			ja.freelancer_id::text,
			r.reviewer_role::text,
			f.username,
			r.rating,
			r.comment,
			r.created_at
		FROM reviews AS r
		JOIN jobs AS j ON j.id = r.job_id
		JOIN job_assignments AS ja ON ja.job_id = j.id
		JOIN freelancers AS f ON f.id = ja.freelancer_id
		WHERE j.client_id = $1
		  AND r.reviewer_role = 'freelancer'
		ORDER BY r.created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]ReviewListItem, 0)

	for rows.Next() {
		var review ReviewListItem
		if err := rows.Scan(
			&review.ID,
			&review.JobID,
			&review.ReviewerID,
			&review.ReviewerRole,
			&review.ReviewerUsername,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}

	return reviews, nil
}
