package postgres

import (
	"database/sql"
	"fmt"
	lemon_api "lemon/lemon-api"
	"lemon/lemon-api/pkg/config"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	config *config.Config

	conn *sqlx.DB

	encryptionKey string

	stmtInsertFeedback   *sqlx.NamedStmt
	stmtGetFeedback      *sqlx.NamedStmt
	stmtMarkReadFeedback *sqlx.NamedStmt

	stmtNewUser    *sqlx.NamedStmt
	stmtGetUser    *sqlx.NamedStmt
	stmtUpdateUser *sqlx.NamedStmt
	stmtDeleteUser *sqlx.NamedStmt
}

func NewService(cfg *config.Config) (*Service, error) {
	if cfg == nil || cfg.Databases == nil || cfg.Databases.Gamejam == nil {
		log.WithFields(log.Fields{
			"config": cfg,
		}).Error("invalid config")
		return nil, config.ErrInvalidConfig
	}

	srv := &Service{
		config:        cfg,
		encryptionKey: cfg.Databases.Gamejam.EncryptionKey,
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
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtCreateFeedback")
		return nil, err
	}

	srv.stmtGetFeedback, err = srv.conn.PrepareNamed(`
	SELECT 
	    id,
		rating,
	    description,
	    type,
	    submitted,
	    read
	FROM
		feedback
`)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtGetFeedback")
		return nil, err
	}

	srv.stmtMarkReadFeedback, err = srv.conn.PrepareNamed(`
	UPDATE feedback
	SET read = true
	WHERE id = :id
`)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtMarkReadFeedback")
		return nil, err
	}

	srv.stmtNewUser, err = srv.conn.PrepareNamed(`
	INSERT INTO usertable (
		id,
	    username,
	    hash,
	    save_state
	    ) VALUES (
	    :id,
		PGP_SYM_ENCRYPT(:username, :encrypt_key),
	    :hash,
	    :save_state
	)
`)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtNewUser")
		return nil, err
	}

	srv.stmtGetUser, err = srv.conn.PrepareNamed(`
	SELECT 
	    id,
		PGP_SYM_DECRYPT(username, :encrypt_key) AS username,
	    hash,
	    save_state
	FROM
		usertable
`)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtGetUser")
		return nil, err
	}

	srv.stmtUpdateUser, err = srv.conn.PrepareNamed(`
	UPDATE usertable
	SET 
	 hash = :hash,
	 save_state = save_state
	WHERE id = :id
`)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtUpdateUser")
		return nil, err
	}

	srv.stmtDeleteUser, err = srv.conn.PrepareNamed(`
	DELETE FROM usertable
	WHERE id = :id
`)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed stmtDeleteUser")
		return nil, err
	}

	return srv, nil
}

func (s *Service) InsertFeedback(feedback lemon_api.Feedback) (sql.Result, error) {
	now := time.Now().UTC()
	feedback.Submitted = &now
	return s.stmtInsertFeedback.Exec(feedback)
}

func (s *Service) GetFeedback() ([]*lemon_api.Feedback, error) {
	var feedback []*lemon_api.Feedback
	query := struct{}{}
	err := s.stmtGetFeedback.Select(&feedback, query)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to Select GetFeedback")
		return nil, err
	}
	return feedback, err
}

func (s *Service) MarkReadFeedback(ID int64) error {
	query := struct {
		ID int64 `db:"id"`
	}{
		ID: ID,
	}
	_, err := s.stmtMarkReadFeedback.Exec(query)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to Exec MarkReadFeedback")
		return err
	}
	return nil
}

func (s *Service) NewUser(user lemon_api.User) (sql.Result, error) {
	query := struct {
		ID            string `db:"id"`
		Username      string `db:"username"`
		Hash          string `db:"hash"`
		SaveState     string `db:"savestate"`
		EncryptionKey string `db:"encrypt_key"`
	}{
		ID:            user.ID,
		Username:      user.Username,
		Hash:          user.Hash,
		SaveState:     user.SaveState,
		EncryptionKey: s.encryptionKey,
	}
	return s.stmtNewUser.Exec(query)
}

func (s *Service) GetUser(ID string) (*lemon_api.User, error) {
	var user *lemon_api.User
	query := struct {
		ID            string `db:"id"`
		EncryptionKey string `db:"encrypt_key"`
	}{
		ID:            ID,
		EncryptionKey: s.encryptionKey,
	}
	err := s.stmtGetFeedback.Get(&user, query)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to Get GetUser")
		return nil, err
	}
	return user, err
}

func (s *Service) UpdateUser(ID string) error {
	query := struct {
		ID            string `db:"id"`
		EncryptionKey string `db:"encrypt_key"`
	}{
		ID:            ID,
		EncryptionKey: s.encryptionKey,
	}
	_, err := s.stmtUpdateUser.Exec(query)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to Exec UpdateUser")
		return err
	}
	return nil
}

func (s *Service) DeleteUser(ID string) error {
	query := struct {
		ID string `db:"id"`
	}{
		ID: ID,
	}
	_, err := s.stmtDeleteUser.Exec(query)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to Exec DeleteUser")
		return err
	}
	return nil
}
