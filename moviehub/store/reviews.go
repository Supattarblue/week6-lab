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
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
}

type Review struct {
	ID        int64        `json:"id"`
	MemberID  int64        `json:"memberId"`
	MovieID   int64        `json:"movieId"`
	Text      string       `json:"text"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Member    ReviewAuthor `json:"member"`
	Score     *int16       `json:"score"`
}

// ListReviews ดึงรีวิวทั้งหมดของหนังเรียงจากใหม่ไปเก่า
func (s *Store) ListReviews(ctx context.Context, movieID int64) ([]Review, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT r.id, r.member_id, r.movie_id, r.text, r.created_at, r.updated_at,
		        m.display_name, v.score
		 FROM reviews r
		 JOIN members m ON m.id = r.member_id
		 LEFT JOIN votes v ON v.member_id = r.member_id AND v.movie_id = r.movie_id
		 WHERE r.movie_id = $1
		 ORDER BY r.created_at DESC`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Review{}
	for rows.Next() {
		var r Review
		if err := rows.Scan(&r.ID, &r.MemberID, &r.MovieID, &r.Text,
			&r.CreatedAt, &r.UpdatedAt, &r.Member.DisplayName, &r.Score); err != nil {
			return nil, err
		}
		r.Member.ID = r.MemberID
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddReview เพิ่มรีวิวใหม่ พร้อมดึง display_name และ score กลับมาทันที
func (s *Store) AddReview(ctx context.Context, memberID, movieID int64, text string) (Review, error) {
	var r Review
	r.MemberID = memberID
	r.MovieID = movieID
	r.Text = text
	r.Member.ID = memberID

	err := s.Pool.QueryRow(ctx,
		`WITH inserted AS (
			INSERT INTO reviews (member_id, movie_id, text)
			VALUES ($1, $2, $3)
			RETURNING id, member_id, movie_id, text, created_at, updated_at
		)
		SELECT i.id, i.created_at, i.updated_at, m.display_name, v.score
		FROM inserted i
		JOIN members m ON m.id = i.member_id
		LEFT JOIN votes v ON v.member_id = i.member_id AND v.movie_id = i.movie_id`,
		memberID, movieID, text).
		Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.Member.DisplayName, &r.Score)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // ซ้ำ (เขียนรีวิวไปแล้ว)
			return Review{}, ErrDuplicate
		case "23503": // คีย์นอกผิด (ไม่มีหนังเรื่องนี้อยู่จริง)
			return Review{}, ErrNoMovie
		}
	}
	return r, err
}

// DeleteReview ลบรีวิวของผู้ใช้ตาม ID
func (s *Store) DeleteReview(ctx context.Context, id, memberID int64) error {
	res, err := s.Pool.Exec(ctx,
		`DELETE FROM reviews WHERE id = $1 AND member_id = $2`, id, memberID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) PutVote(ctx context.Context, memberID, movieID int64, score int16) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO votes (member_id, movie_id, score)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (member_id, movie_id)
		 DO UPDATE SET score = EXCLUDED.score, updated_at = now()`,
		memberID, movieID, score,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrNoMovie
	}

	return err
}

func (s *Store) GetWishlist(ctx context.Context, memberID int64) ([]Movie, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT m.id, m.title, m.title_th,
		        (SELECT coalesce(g.name_th, g.name_en)
		         FROM movie_genres mg
		         JOIN genres g ON g.id = mg.genre_id
		         WHERE mg.movie_id = m.id AND mg.position = 0),
		        extract(year FROM m.release_date)::int,
		        round(m.vote_average, 1)::float8,
		        CASE
		            WHEN m.poster_path LIKE 'http%' THEN m.poster_path
		            WHEN m.poster_path IS NULL THEN NULL
		            ELSE 'https://image.tmdb.org/t/p/w500' || m.poster_path
		        END
		 FROM wishlist_items w
		 JOIN movies m ON m.id = w.movie_id
		 WHERE w.member_id = $1
		 ORDER BY w.added_at DESC`,
		memberID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Movie{}
	for rows.Next() {
		var m Movie
		if err := rows.Scan(
			&m.ID, &m.Title, &m.TitleTh, &m.Genre,
			&m.Year, &m.Rating, &m.Poster,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}

	return items, rows.Err()
}

func (s *Store) AddWishlist(ctx context.Context, memberID, movieID int64) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO wishlist_items (member_id, movie_id)
		 VALUES ($1, $2)
		 ON CONFLICT (member_id, movie_id) DO NOTHING`,
		memberID, movieID,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrNoMovie
	}

	return err
}

func (s *Store) RemoveWishlist(ctx context.Context, memberID, movieID int64) error {
	_, err := s.Pool.Exec(ctx,
		`DELETE FROM wishlist_items
		 WHERE member_id = $1 AND movie_id = $2`,
		memberID, movieID,
	)
	return err
}
