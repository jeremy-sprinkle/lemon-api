package rest

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
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
	s.engine.GET("api/feedback/:ID", s.GetFeedbackByID)
	s.engine.PUT("api/feedback/:ID", s.MarkReadFeedback)

	s.engine.POST("api/register", s.NewUser)
	s.engine.GET("api/taken/:Username", s.UserAvailableCheck)
	s.engine.POST("api/login", s.Login)
	s.engine.GET("api/logout", s.Logout)
	s.engine.PUT("api/save", s.UpdateUser)
	s.engine.PUT("api/elevate", s.ElevateUser)
	s.engine.GET("api/save/:ID", s.GetUser)
	s.engine.DELETE("api/save", s.DeleteUser)

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
	returnedID, err := s.database.InsertFeedback(feedback)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to insert feedback")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	webhookBody, err := json.Marshal(map[string]string{
		"username":   "Feedback Piggy",
		"avatar_url": "https://www.discordavatars.com/wp-content/uploads/2020/07/disney-character-avatar-074.jpg",
		"content": "New feedback given! Rating: " + strconv.FormatInt(feedback.Rating, 10) + "\n " +
			"Type: " + feedback.Type + "\n" +
			"https://lemon.indiedev.io/feedback/" + strconv.FormatInt(returnedID, 10) + " to read full feedback.",
	})
	if err != nil {
		log.Error(err)
	}

	hook, err := http.Post(s.config.Webhooks.FeedbackURL, "application/json", bytes.NewBuffer(webhookBody))
	if err != nil {
		log.Error(err, hook)
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

func (s *Server) GetFeedbackByID(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
	}

	newID, err := strconv.ParseInt(ID, 10, 32)
	if err != nil {
		log.Error(err)
	}

	feedback, err := s.database.GetFeedbackByID(newID)
	if err != nil {
		log.Error(err)
	}

	c.JSON(http.StatusOK, feedback)
}

func (s *Server) MarkReadFeedback(c *gin.Context) {
	tokenAccountID, err := security.GetTokenAccountID(s.config, c.GetHeader("Authorization"))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed to get token.ID")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	user, err := s.database.GetUserByID(*tokenAccountID)
	if err != nil {log.Error(err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if user.Role != lemon_api.DeveloperRole.Name {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

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

	unHashed := user.Hash
	newHash := sha256.Sum256([]byte(unHashed + s.config.Security.Salt + user.Username))
	newHashSlice := newHash[:]
	user.Hash = bytes.NewBuffer(newHashSlice).String()

	user.Role = lemon_api.UserRole.Name

	_, err := s.database.NewUser(user)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to insert new user")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	token, err := s.GenerateToken(user.Username, unHashed)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to generate token")
	}

	webhookBody, err := json.Marshal(map[string]string{
		"username":   "Big Brother",
		"avatar_url": "https://www.discordavatars.com/wp-content/uploads/2020/10/cctv-camera-avatar-150x150.jpg",
		"content":    user.Username + " has joined the party!",
	})
	if err != nil {
		log.Error(err)
	}

	hook, err := http.Post(s.config.Webhooks.NewUserURL, "application/json", bytes.NewBuffer(webhookBody))
	if err != nil {
		log.Error(err, hook)
	}

	c.SetCookie("lemon-token", token.Value, 604800, "/", ".indiedev.io", true, false)
	c.JSON(http.StatusOK, token)
}

func (s *Server) UserAvailableCheck(c *gin.Context){
	username := c.Param("Username")
	if username == ""{
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	_, err := s.database.GetUserByUsername(username)
	if err == nil {
		c.AbortWithStatus(http.StatusConflict)
		return
	}
	c.AbortWithStatus(http.StatusOK)
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
	c.SetCookie("lemon-token", token.Value, 604800, "/", ".indiedev.io", true, false)
	c.JSON(http.StatusOK, token)
}

func (s *Server) Logout(c *gin.Context) {
	c.SetCookie("lemon-token", "", 0, "/", ".indiedev.io", true, false)
	c.Redirect(http.StatusPermanentRedirect, "/login")
}

func (s *Server) GetUser(c *gin.Context) {
	tokenAccountID, err := security.GetTokenAccountID(s.config, c.GetHeader("Authorization"))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed to get token.ID")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	data, err := s.database.GetUserByID(*tokenAccountID)
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
	tokenAccountID, err := security.GetTokenAccountID(s.config, c.GetHeader("Authorization"))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed to get token.ID")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	user.ID = *tokenAccountID
	err = s.database.UpdateUser(user)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to update user in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) ElevateUser(c *gin.Context) {
	var request lemon_api.RoleRequest
	if err := c.BindJSON(&request); err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	tokenAccountID, err := security.GetTokenAccountID(s.config, c.GetHeader("Authorization"))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed to get token.ID")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	user, err := s.database.GetUserByID(*tokenAccountID)
	if err != nil {log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return}

	if request.Secret != s.config.Security.Secret {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	user.Role = lemon_api.DeveloperRole.Name
	err = s.database.ElevateUser(*user)
	if err != nil {log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return}

	var token lemon_api.Token

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"iss":   "https://lemon.indiedev.io",
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
		"sub":    user.ID,
		"aud":   "https://lemon.indiedev.io",
		"nbf":   time.Now().Unix(),
		"id":    user.ID,
		"guest": false,
		"roles": lemon_api.DeveloperRole,
		"name":  user.Username,
	})

	signedString, err := tkn.SignedString([]byte(s.config.Security.Secret))
	if err != nil {
		log.Error(err)
		return
	}
	token.Value = signedString

	c.JSON(http.StatusOK, token)
}

func (s *Server) DeleteUser(c *gin.Context) {
	tokenAccountID, err := security.GetTokenAccountID(s.config, c.GetHeader("Authorization"))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Failed to get token.ID")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	err = s.database.DeleteUser(*tokenAccountID)
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

	var role lemon_api.Role
	if existingAccount.Role == "DEVELOPER" {
		role = lemon_api.DeveloperRole
	} else if existingAccount.Role == "USER"{
		role = lemon_api.UserRole
	}

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"iss":   "https://lemon.indiedev.io",
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
		"sub":   existingAccount.ID,
		"aud":   "https://lemon.indiedev.io",
		"nbf":   time.Now().Unix(),
		"id":    existingAccount.ID,
		"guest": false,
		"roles": role,
		"name":  existingAccount.Username,
	})

	signedString, err := tkn.SignedString([]byte(s.config.Security.Secret))
	if err != nil {
		return nil, err
	}

	token.Value = signedString
	return &token, nil
}
