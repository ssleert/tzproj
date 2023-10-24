package db

import (
	"context"
	
	"github.com/ssleert/tzproj/internal/vars"
	"github.com/ssleert/tzproj/pkg/genderize"

	"github.com/jackc/pgx/v5/pgxpool"
)

type All struct {
	P People
	G Gender
	N Nationalization
}

type People struct {
	Id         int
	Name       string
	Surname    string
	Patronymic *string
	Age        int
}

type Gender struct {
	Id          int
	Gender      genderize.Gender
	Probability float64
}

type Nationalization struct {
	Id            int
	CountryIds    []string
	Probabilities []float64
}

func GetPeopleId(conn *pgxpool.Conn, p People) (int, error) {
	var err error
	if p.Patronymic != nil {
		err = conn.QueryRow(context.Background(),
			`SELECT people_id FROM peoples 
			 WHERE name = $1 AND surname = $2 AND age = $3 AND patronymic = $4`,
			 p.Name, p.Surname, p.Age, *p.Patronymic,
		).Scan(&p.Id)
	} else {
		err = conn.QueryRow(context.Background(),
			`SELECT people_id FROM peoples 
			 WHERE name = $1 AND surname = $2 AND age = $3`,
			 p.Name, p.Surname, p.Age,
		).Scan(&p.Id)
	}
	if err != nil {
		return 0, err
	}
	return p.Id, nil
}

func CheckPeople(conn *pgxpool.Conn, p People) (bool, error) {
	var (
		exists bool
		err error
	)
	if p.Patronymic != nil {
		err = conn.QueryRow(context.Background(),
			`SELECT EXISTS(SELECT 1 FROM peoples 
			 WHERE name = $1 AND surname = $2 AND age = $3 AND patronymic = $4)`,	
			 p.Name, p.Surname, p.Age, *p.Patronymic,
		).Scan(&exists)
	} else {
		err = conn.QueryRow(context.Background(),
			`SELECT EXISTS(SELECT 1 FROM peoples 
			 WHERE name = $1 AND surname = $2 AND age = $3)`,	
			 p.Name, p.Surname, p.Age,
		).Scan(&exists)
	}
	if err != nil {
		return false, err
	}
	return exists, nil
}

func InsertPeople(conn *pgxpool.Conn, p People) (int, error) {
	var (
		id  int
		err error
	)
	if p.Id == 0 {
		err = conn.QueryRow(context.Background(),
			`INSERT INTO peoples (name, surname, age, patronymic)
			 VALUES ($1, $2, $3, $4)
			 RETURNING people_id`,
			p.Name, p.Surname, p.Age, p.Patronymic,
		).Scan(&id)
	} else {
		err = conn.QueryRow(context.Background(),
			`INSERT INTO peoples (people_id, name, surname, age, patronymic)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING people_id`,
			p.Id, p.Name, p.Surname, p.Age, p.Patronymic,
		).Scan(&id)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func InsertAll(conn *pgxpool.Conn, a All) (int, error) {
	exists, err := CheckPeople(conn, a.P)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, vars.ErrAlreadyInDb
	}

	id, err := InsertPeople(conn, a.P)
	if err != nil {
		return 0, err
	}
	a.P.Id = id
	a.G.Id = id
	a.N.Id = id

	_, err = conn.Exec(context.Background(),
		`INSERT INTO genders (people_id, gender, probability)
		 VALUES ($1, $2, $3)`,
		a.G.Id, a.G.Gender, a.G.Probability,
	)
	if err != nil {
		return 0, err
	}
	_, err = conn.Exec(context.Background(),
		`INSERT INTO nationalizations (people_id, country_id, probability)
		 VALUES ($1, $2, $3)`,
		a.N.Id, a.N.CountryIds, a.N.Probabilities,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func DeleteAllById(conn *pgxpool.Conn, id int) (error) {
	var (
		exists bool
		err error
	)
	err = conn.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM peoples 
		 WHERE people_id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return vars.ErrNotInDb
	}

	_, err = conn.Exec(context.Background(),
		`DELETE FROM genders WHERE people_id = $1`, id,
	)
	if err != nil {
		return err
	}	
	_, err = conn.Exec(context.Background(),
		`DELETE FROM nationalizations WHERE people_id = $1`, id,
	)
	if err != nil {
		return err
	}
	_, err = conn.Exec(context.Background(),
		`DELETE FROM peoples WHERE people_id = $1`, id,
	)
	if err != nil {
		return err
	}
	return nil
}
