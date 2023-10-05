package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/ManoMartins/rinha-go/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isValidDateFormat(date string) bool {
	datePattern := `^\d{4}-\d{2}-\d{2}$`
	re := regexp.MustCompile(datePattern)
	return re.MatchString(date)
}

func isValidDateValue(date string) bool {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}

	year, month, day := parsedDate.Date()
	if year < 1900 || year > time.Now().Year() || month < 1 || month > 12 || day < 1 || day > 31 {
		return false
	}

	return true
}

func main() {
	app := fiber.New()

	var psqlconn string = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", "db",
		"postgres", "12345678", "root")

	poolConfig, err := pgxpool.ParseConfig(psqlconn)

	if err != nil {
		log.Fatal(err)
	}

	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	_, err = db.Query(context.Background(), "CREATE EXTENSION IF NOT EXISTS pg_trgm")

	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	_, err = db.Query(context.Background(), `CREATE OR REPLACE FUNCTION generate_searchable(_nome VARCHAR, _apelido VARCHAR, _stack JSON)
		RETURNS TEXT AS $$
		BEGIN
		RETURN _nome || _apelido || _stack;
		END;
	$$ LANGUAGE plpgsql IMMUTABLE`)

	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	_, err = db.Query(context.Background(), `CREATE TABLE IF NOT EXISTS pessoas (
		id uuid UNIQUE NOT NULL,
		apelido TEXT UNIQUE NOT NULL,
		nome TEXT NOT NULL,
		nascimento DATE NOT NULL,
		stack JSON,
		searchable text GENERATED ALWAYS AS (generate_searchable(nome, apelido, stack)) STORED
	)`)

	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	_, err = db.Query(context.Background(), `CREATE INDEX IF NOT EXISTS idx_pessoas_searchable ON public.pessoas USING gist (searchable public.gist_trgm_ops (siglen='64'))`)

	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	_, err = db.Query(context.Background(), `CREATE UNIQUE INDEX IF NOT EXISTS pessoas_apelido_index ON public.pessoas USING btree (apelido)`)

	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	repo := repository.NewRepository(db)

	// GET /api/register
	app.Post("/pessoas", func(c *fiber.Ctx) error {
		payload := struct {
			Apelido    *string    `json:"apelido"`
			Nome       *string    `json:"nome"`
			Nascimento *string    `json:"nascimento"`
			Stack      *[]*string `json:"stack"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return c.SendStatus(400)
		}

		if payload.Apelido == nil || payload.Nome == nil || payload.Nascimento == nil {
			return c.SendStatus(422)
		}

		if len([]rune(*payload.Apelido)) > 32 || len([]rune(*payload.Nome)) > 100 {
			return c.SendStatus(422)
		}

		if !isValidDateFormat(*payload.Nascimento) {
			return c.SendStatus(400)
		}

		if !isValidDateValue(*payload.Nascimento) {
			return c.SendStatus(422)
		}

		// Deixar passar se tiver null mas não se tiver vazio
		if payload.Stack == nil {
			return c.SendStatus(400)
		}

		if payload.Stack != nil && len(*payload.Stack) > 0 {
			for _, stack := range *payload.Stack {
				if stack == nil || len([]rune(*stack)) > 32 {
					return c.SendStatus(422)
				}
			}
		}

		id := uuid.New().String()

		repo.Save(context.Background(), repository.Pessoa{
			ID:         &id,
			Apelido:    payload.Apelido,
			Nome:       payload.Nome,
			Nascimento: payload.Nascimento,
			Stack:      payload.Stack,
		})

		c.SendStatus(201)
		c.Response().Header.Add("Location", fmt.Sprintf("/pessoas/%s", id))
		return nil
	})

	app.Get("/pessoas/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		pessoa, err := repo.Get(context.Background(), id)

		if err != nil {
			return c.SendStatus(404)
		}

		return c.JSON(pessoa)
	})

	app.Get("/pessoas", func(c *fiber.Ctx) error {
		t := c.Params("t")

		pessoas, err := repo.SearchByTerm(context.Background(), t)

		if err != nil {
			return c.SendStatus(400)
		}

		return c.JSON(pessoas)
	})

	app.Get("/contagem-pessoas", func(c *fiber.Ctx) error {
		count, err := repo.Count(context.Background())

		if err != nil {
			return c.SendStatus(400)
		}

		return c.SendString(strconv.Itoa(count))
	})

	log.Fatal(app.Listen(":8080"))
}
