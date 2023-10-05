package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pessoa struct {
	ID         *string    `json:"id"`
	Apelido    *string    `json:"apelido"`
	Nome       *string    `json:"nome"`
	Nascimento *string    `json:"nascimento"`
	Stack      *[]*string `json:"stack"`
}

type Repository interface {
	Save(ctx context.Context, p Pessoa) error
	Get(ctx context.Context, id string) (Pessoa, error)
	SearchByTerm(ctx context.Context, t string) ([]Pessoa, error)
	Count(ctx context.Context) (int, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Save(ctx context.Context, p Pessoa) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO
     pessoas(
        id,
        apelido,
        nome,
        nascimento,
        stack
     )
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5::json
    )`,
		p.ID, p.Apelido, p.Nome, p.Nascimento, p.Stack)

	if err != nil {
		return err
	}

	return nil
}

func (r *repository) Get(ctx context.Context, id string) (Pessoa, error) {
	var p Pessoa
	err := r.db.QueryRow(ctx, `
		SELECT
				id,
				apelido,
				nome,
				to_char(nascimento, 'YYYY-MM-DD') as nascimento,
				stack
		FROM
				pessoas
		WHERE "id" = $1;`, id).Scan(
		&p.ID, &p.Apelido, &p.Nome, &p.Nascimento, &p.Stack)

	if err != nil {
		return p, err
	}

	return p, nil
}

func (r *repository) SearchByTerm(ctx context.Context, t string) ([]Pessoa, error) {
	var ps []Pessoa
	rows, err := r.db.Query(ctx, `
		SELECT
				id,
				apelido,
				nome,
				to_char(nascimento, 'YYYY-MM-DD') as nascimento,
				stack
		FROM
				pessoas
		WHERE
				searchable ILIKE $1
		LIMIT 50;`, t)

	if err != nil {
		return ps, err
	}

	for rows.Next() {
		var p Pessoa
		err := rows.Scan(&p.ID, &p.Apelido, &p.Nome, &p.Nascimento, &p.Stack)

		if err != nil {
			return ps, err
		}

		ps = append(ps, p)
	}

	return ps, nil
}

func (r *repository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(1) FROM pessoas`).Scan(&count)

	if err != nil {
		return count, err
	}

	return count, nil
}
