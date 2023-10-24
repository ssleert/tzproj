package del

import (
	"net/http"
	"encoding/json"
	"context"
	"io"

	"github.com/ssleert/tzproj/internal/utils"
	"github.com/ssleert/tzproj/internal/vars"
	"github.com/ssleert/tzproj/internal/db"
	
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
)

type input struct {
	Id int `json:"delete_id"`
}

type output struct {
	Err string `json:"error"`
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

	log.Trace().
		Int("people_id", in.Id).
		Msg("writing data to db")	
	err = dbConn.AcquireFunc(context.Background(), func(c *pgxpool.Conn) error {
		err = db.DeleteAllById(c, in.Id)
		return err
	})
	if err == vars.ErrNotInDb {
		log.Warn().
			Err(err).
			Msg("data not in db")
		out.Err = vars.ErrNotInDb.Error()
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

	log.Debug().
		RawJSON("body", body).
		Interface("input_json", in).
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
