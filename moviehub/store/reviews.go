package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrDuplicate = errors.New("duplicate")
	ErrNoMovie   = errors.New("movie not found")
)

type ReviewAuthor struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Review struct {
	ID        int64        `json:"id"`
	MemberID  int64        `json:"memberId"`
	MovieID   int64        `json:"movieId"`
	Text      string       `json:"text"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Author    ReviewAuthor `json:"author"`
}

func (s *Store) AddReview(ctx context.Context, memberID, movieID int64, text string) (Review, error) {
	r := Review{
		MemberID: memberID,
		MovieID:  movieID,
		Text:     text,
	}

	err := s.Pool.QueryRow(ctx,
		`INSERT INTO reviews (member_id, movie_id, text)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		memberID, movieID, text,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return Review{}, ErrDuplicate
		case "23503":
			return Review{}, ErrNoMovie
		}
	}

	return r, err
}

func (s *Store) ListReviews(ctx context.Context, movieID int64) ([]Review, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT r.id, r.member_id, r.movie_id, r.text,
		        r.created_at, r.updated_at,
		        m.id, m.username
		 FROM reviews r
		 JOIN members m ON m.id = r.member_id
		 WHERE r.movie_id = $1
		 ORDER BY r.created_at DESC`,
		movieID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []Review{}

	for rows.Next() {
		var r Review
		err := rows.Scan(
			&r.ID,
			&r.MemberID,
			&r.MovieID,
			&r.Text,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.Author.ID,
			&r.Author.Username,
		)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}

func (s *Store) ListMembers(ctx context.Context) ([]Member, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, email, display_name, role FROM members ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.DisplayName, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, rows.Err()
}

func (s *Store) DeleteReview(ctx context.Context, reviewID, memberID int64, isAdmin bool) error {
	tag, err := s.Pool.Exec(ctx,
		`DELETE FROM reviews
		 WHERE id = $1 AND (member_id = $2 OR $3)`,
		reviewID, memberID, isAdmin,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
