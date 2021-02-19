package postgres

import (
	"database/sql"
	"fmt"
	lemon_api "lemon/lemon-api"
	"lemon/lemon-api/pkg/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	config *config.Config

	conn *sqlx.DB

	stmtInsertFeedback        *sqlx.NamedStmt
	stmtGetFeedback				*sqlx.NamedStmt
	stmtMarkReadFeedback		*sqlx.NamedStmt
}

func NewService(cfg *config.Config) (*Service, error) {
	if cfg == nil || cfg.Databases == nil || cfg.Databases.Gamejam == nil {
		log.WithFields(log.Fields{
			"config": cfg,
		}).Error("invalid config")
		return nil, config.ErrInvalidConfig
	}

	srv := &Service{
		config: cfg,
	}

	conn, err := sqlx.Connect("postgres", fmt.Sprintf("postgres://%v:%v@%v:%d/%v",
		cfg.Databases.Gamejam.Username,
		cfg.Databases.Gamejam.Password,
		cfg.Databases.Gamejam.Hostname,
		cfg.Databases.Gamejam.Port,
		cfg.Databases.Gamejam.Database))
	if err != nil {
		return nil, err
	}

	srv.conn = conn

	srv.stmtInsertFeedback, err = srv.conn.PrepareNamed(`
	INSERT INTO feedback (
		rating,
	    description,
	    type,
	    submitted
	    ) VALUES (
	    :rating,
		:description,
	    :type,
	    :submitted
	)
`)
	if err != nil {
		log.WithFields(log.Fields{"err":err}).Error("Failed stmtCreateFeedback")
		return nil, err
	}

	srv.stmtGetFeedback, err = srv.conn.PrepareNamed(`
	SELECT 
	    id,
		rating,
	    description,
	    type,
	    submitted
	FROM
		feedback
	WHERE
		read = false
`)
	if err != nil {
		log.WithFields(log.Fields{"err":err}).Error("Failed stmtGetFeedback")
		return nil, err
	}

	srv.stmtMarkReadFeedback, err = srv.conn.PrepareNamed(`
	UPDATE feedback
	SET read = true
	WHERE id = :id
`)
	if err != nil {
		log.WithFields(log.Fields{"err":err}).Error("Failed stmtMarkReadFeedback")
		return nil, err
	}

	return srv, nil
}

func (s *Service) InsertFeedback(feedback lemon_api.Feedback) (sql.Result, error) {
	return s.stmtInsertFeedback.Exec(feedback)
}