package rest

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"fmt"
	lemon_api "lemon/lemon-api"
	"lemon/lemon-api/pkg/security"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"

	"lemon/lemon-api/pkg/config"
	"lemon/lemon-api/pkg/postgres"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	config   *config.Config
	engine   *gin.Engine
	database *postgres.Service
}

func NewServer(cfg *config.Config, e *gin.Engine) *Server {
	rand.Seed(time.Now().UTC().UnixNano())

	return &Server{
		config: cfg,
		engine: e,
	}
}

func (s *Server) Initialise() {

	s.engine.POST("api/feedback", s.InsertFeedback)
	s.engine.GET("api/feedback", s.GetFeedback)
	s.engine.PUT("api/feedback/:ID", s.MarkReadFeedback)

	s.engine.POST("api/register", s.NewUser)
	s.engine.POST("api/login", s.Login)
	s.engine.PUT("api/save/:ID", security.Authenticate(s.config, false), s.UpdateUser)
	s.engine.GET("api/save/:ID", security.Authenticate(s.config, false), s.GetUser)
	s.engine.DELETE("api/save/:ID", security.Authenticate(s.config, false), s.DeleteUser)

	if service, err := postgres.NewService(s.config); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to start database service")
		return
	} else {
		s.database = service
	}

	var filename = "logfile.log"
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	log.SetFormatter(&log.JSONFormatter{})
	if err != nil {
		fmt.Println(err)
	} else {
		log.SetOutput(f)
	}

}

func (s *Server) InsertFeedback(c *gin.Context) {
	var feedback lemon_api.Feedback
	if err := c.BindJSON(&feedback); err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"data": feedback,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	_, err := s.database.InsertFeedback(feedback)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to insert feedback")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) GetFeedback(c *gin.Context) {
	data, err := s.database.GetFeedback()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to get feedback from database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) MarkReadFeedback(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	feedbackID, err := strconv.ParseInt(ID, 10, 32)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to convert ID to Int")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	err = s.database.MarkReadFeedback(feedbackID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to mark read in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) NewUser(c *gin.Context) {
	var user lemon_api.User
	if err := c.BindJSON(&user); err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"data": user,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}

	accountID := uuid.New().String()
	user.ID = accountID

	_, err := s.database.NewUser(user)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to insert new user")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	token, err := s.GenerateToken(user.Username, user.Hash)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to generate token")
	}

	c.JSON(http.StatusOK, token)
}

func (s *Server) Login(c *gin.Context) {
	var loginRequest lemon_api.TokenRequest
	if err := c.BindJSON(&loginRequest); err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"data": loginRequest,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}

	token, err := s.GenerateToken(loginRequest.Username, loginRequest.Hash)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to generate token")
	}

	c.JSON(http.StatusOK, token)
}

func (s *Server) GetUser(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}

	tokenAccountID, err := security.GetTokenAccountID(s.config, c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if ID != *tokenAccountID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	data, err := s.database.GetUserByID(ID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to get user from database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) UpdateUser(c *gin.Context) {
	var user lemon_api.User
	if err := c.BindJSON(&user); err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"data": user,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	err := s.database.UpdateUser(user)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to update user in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) DeleteUser(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	err := s.database.DeleteUser(ID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to update user in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) GenerateToken(username string, hash string) (*lemon_api.Token, error) {
	existingAccount, err := s.database.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, security.ErrInvalidAccount
		}
		return nil, err
	}

	// Salt + ReHash Password
	newHash := sha256.Sum256([]byte(hash + s.config.Security.Salt + existingAccount.Username))
	newHashSlice := newHash[:]
	hashString := bytes.NewBuffer(newHashSlice).String()

	// Password Incorrect
	if existingAccount.Hash != hashString {
		return nil, security.ErrInvalidCredentials
	}

	var token lemon_api.Token

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"iss":   "https://lemon.indiedev.io",
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
		"sub":   existingAccount.ID,
		"aud":   "https://lemon.indiedev.io",
		"nbf":   time.Now().Unix(),
		"id":    existingAccount.ID,
		"guest": false,
		"name":  existingAccount.Username,
	})

	signedString, err := tkn.SignedString([]byte(s.config.Security.Secret))
	if err != nil {
		return nil, err
	}

	token.Value = signedString
	return &token, nil
}
