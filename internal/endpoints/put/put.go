package put

import (
	"net/http"
	"encoding/json"
	"context"
	"io"

	"github.com/ssleert/tzproj/internal/utils"
	"github.com/ssleert/tzproj/internal/vars"
	"github.com/ssleert/tzproj/internal/db"
	"github.com/ssleert/tzproj/internal/conversions"

	"github.com/ssleert/tzproj/pkg/agify"
	"github.com/ssleert/tzproj/pkg/genderize"
	"github.com/ssleert/tzproj/pkg/nationalize"
	
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/ssleert/limiter"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// api endpoint like /put
	name   string
	logger zerolog.Logger
	limit  *limiter.Limiter[string]
	dbConn *pgxpool.Pool

	agifyClient       *agify.Client
	genderizeClient   *genderize.Client
	nationalizeClient *nationalize.Client
)

type input struct {
	Name       string  `json:"name"`
	Surname    string  `json:"surname"`
	Patronymic *string `json:"patronymic,omitempty"`
}

type output struct {
	WritedId int `json:"writed_id"`
	Err string   `json:"error"`
}

func Start(n string, log *zerolog.Logger) error {
	var err error
	logger = *log
	name = n
	logger.Trace().Msg("creating req limiter")
	limit = limiter.New[string](vars.LimitPerHour, 3600, 2048, 4096, 20)

	logger.Trace().Msg("creating db connection")
	dbConn, err = pgxpool.New(context.Background(), db.GetConnString())
	if err != nil {
		logger.Error().
			Err(err).
			Msg("error with db connection")
		return err
	}

	logger.Trace().Msg("creating agify.io, genderize.io and nationalize.io clients")
	agifyClient, err = agify.New()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("agify.io client creation failed")
		return err
	}
	genderizeClient, err = genderize.New()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("genderize.io client creation failed")
		return err
	}
	nationalizeClient, err = nationalize.New()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("nationalize.io client creation failed")
		return err
	}

	logger.Info().Msgf("%s endpoint started", name)
	return nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log := hlog.FromRequest(r)
	log.Info().Msg("connected")

	httpStatus := http.StatusOK
	in := input{}
	out := output{Err: "null"}
	defer func() { 
		utils.WriteJsonAndStatusInRespone(w, &out, httpStatus)
	}()

	log.Trace().Msg("checking req limiter")
	if !limit.Try(utils.GetAddrFromStr(&r.RemoteAddr)) {
		log.Warn().Msg("action limited")
		out.Err = vars.ErrActionLimited.Error()
		httpStatus = http.StatusTooManyRequests
		return
	}
	log.Trace().Msg("checking content len")
	if r.ContentLength > vars.MaxBodyLen {
		log.Warn().
			Int64("content_length", r.ContentLength).
			Int64("max_content_length", vars.MaxBodyLen).
			Msg("content length is too big")
		out.Err = vars.ErrBodyLenIsTooBig.Error()
		httpStatus = http.StatusRequestEntityTooLarge
		return
	}	
	log.Trace().Msg("reading body")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("cant read all body")
		out.Err = vars.ErrBodyReadingFailed.Error()
		httpStatus = http.StatusInsufficientStorage
		return
	}
	log.Trace().Msg("unmarshaling json")
	err = json.Unmarshal(body, &in)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("cant unmarshal body to json")
		out.Err = vars.ErrInputJsonIsIncorrect.Error()
		httpStatus = http.StatusUnprocessableEntity
		return
	}

	var (
		ageChan = make(chan utils.Result[agify.Output])
		genderChan = make(chan utils.Result[genderize.Output])
		nationChan = make(chan utils.Result[nationalize.Output])
	)
	go func() {
		log.Trace().Msg("getting agify.io data")
		o, err := agifyClient.Get(in.Name)
		ageChan <- utils.Result[agify.Output]{o, err}
		close(ageChan)
	}()
	go func() {
		log.Trace().Msg("getting genderize.io data")
		o, err := genderizeClient.Get(in.Name)
		genderChan <- utils.Result[genderize.Output]{o, err}
		close(genderChan)
	}()
	go func() {
		log.Trace().Msg("getting nationalize.io data")
		o, err := nationalizeClient.Get(in.Name)
		nationChan <- utils.Result[nationalize.Output]{o, err}
		close(nationChan)
	}()

	ageResult := <-ageChan
	genderResult := <-genderChan
	nationResult := <-nationChan
	if ageResult.Err != nil {
		log.Warn().
			Err(err).
			Msg("agify.io error")
		out.Err = vars.ErrWithExternalApi.Error()
		httpStatus = http.StatusInternalServerError
		return
	}
	if genderResult.Err != nil {
		log.Warn().
			Err(err).
			Msg("genderize.io error")
		out.Err = vars.ErrWithExternalApi.Error()
		httpStatus = http.StatusInternalServerError
		return
	}
	if nationResult.Err != nil {
		log.Warn().
			Err(err).
			Msg("nationalize.io error")
		out.Err = vars.ErrWithExternalApi.Error()
		httpStatus = http.StatusInternalServerError
		return
	}
	age := ageResult.Val
	gender := genderResult.Val
	nation := nationResult.Val
	
	allData := db.All{
		P: db.People{
			Name: in.Name,
			Surname: in.Surname,
			Patronymic: in.Patronymic,
			Age: age.Age,
		},
		G: db.Gender{
			Gender: gender.Gender,
			Probability: gender.Probability,
		},
		N: db.Nationalization{
			CountryIds: conversions.CountryToIds(nation.Countries),
			Probabilities: conversions.CountryToProbalities(nation.Countries),
		},
	}

	log.Trace().
		Interface("all_data", allData).
		Msg("writing data to db")	

	id := 0
	err = dbConn.AcquireFunc(context.Background(), func(c *pgxpool.Conn) error {
		id, err = db.InsertAll(c, allData)
		return err
	})
	if err == vars.ErrAlreadyInDb {
		log.Warn().
			Err(err).
			Msg("data already in db")
		out.Err = vars.ErrAlreadyInDb.Error()
		httpStatus = http.StatusInternalServerError
		return
	}
	if err != nil {
		log.Warn().
			Err(err).
			Msg("an error with database")
		out.Err = vars.ErrWithDb.Error()
		httpStatus = http.StatusInternalServerError
		return
	}
	out.WritedId = id

	log.Debug().
		RawJSON("body", body).
		Interface("input_json", in).
		Send()
	log.Trace().
		Bool("is_patronymic", in.Patronymic != nil).
		Send()
	log.Debug().
		Interface("output_json", out).
		Send()
}

func Stop() error {
	if dbConn != nil {
		dbConn.Close()
	}
	logger.Info().Msgf("%s endpoint stoped", name)
	return nil
}
