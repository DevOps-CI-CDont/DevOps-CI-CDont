package Api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"minitwit-backend/init/config"
	"minitwit-backend/init/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var Router *gin.Engine

func SetUpRouter() *gin.Engine {
	r := gin.Default()
	return r
}

func Start() {
	Router = SetUpRouter()

	config.Connect_db()

	// router config
	Router.Use(cors.Default()) // cors.Default() should allow all origins
	// it's important to set this before any routes are registered so that the middleware is applied to all routes
	// ALL MY HOMIES HATE CORS :D

	// endpoints
	Router.GET("/mytimeline", getTimeline)
	Router.GET("/public", getPublicTimeline)
	Router.GET("/user/:username", getUsersTweets)
	Router.POST("/user/:username/follow", followUser)
	Router.POST("/user/:username/unfollow", unfollowUser)
	Router.POST("/add_message", postMessage)
	Router.POST("/login", login)
	Router.POST("/register", register)
	Router.GET("/logout", logout)
	Router.GET("/AmIFollowing/:username", amIFollowing)
	Router.GET("/allUsers", getAllUsers)
	Router.GET("AllIAmFollowing", getAllFollowing)
	Router.Run(":8080")
}

// Capitalized names are public, lowercase are private
var PER_PAGE = 30
var DEBUG = true

func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	hexString := hex.EncodeToString(hash[:])
	return hexString
}

func errorCheck(err error) {
	if err != nil {
		log.Println(err)
	}
}

// endpoints

func amIFollowing(c *gin.Context) {
	username := c.Param("username")
	userID := getUserIdIfLoggedIn(c)
	var follower models.Follower
	var user models.User
	err := config.DB.Table("followers").
		Where("who_id = ? AND whom_id = ?", userID, config.DB.Table("users").Select("user_id").Where("username = ?", username).Find(&user)).First(&follower).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(200, false)
			return
		}
		c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(200, true)
}

func getTimeline(c *gin.Context) {
	userID := getUserIdIfLoggedIn(c)

	var messages []models.Message
	result := config.DB.Table("messages").
		Select("messages.*, users.*").
		Joins("JOIN users ON messages.author_id = users.user_id").
		Where("messages.flagged = ? AND (users.user_id = ? OR users.user_id IN (?))",
			0, userID, config.DB.Table("followers").Select("whom_id").Where("who_id = ?", userID)).
		Order("messages.pub_date DESC").
		Limit(PER_PAGE).
		Scan(&messages)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving messages"})
		return
	}

	c.JSON(200, gin.H{"tweets": messages})
}

func getPublicTimeline(c *gin.Context) {
	num_msgs := c.Request.URL.Query().Get("num_msgs")
	int_num_msgs, err := strconv.Atoi(num_msgs)
	if num_msgs == "" || err != nil {
		int_num_msgs = 30
	}

	fmt.Println("int_num_msgs", int_num_msgs)

	var messages []models.Message
	err = config.DB.
		Table("messages").
		Select("messages.*, users.*").
		Joins("JOIN users ON messages.author_id = users.id").
		Where("messages.flagged = ?", 0).
		Order("messages.pub_date desc").
		Limit(int_num_msgs).
		Find(&messages).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "failed to retrieve messages"})
		return
	}

	// if no messages, return 401
	if len(messages) == 0 {
		c.JSON(401, gin.H{"message": "no messages"})
		return
	}

	fmt.Println("messages", messages)

	c.JSON(200, gin.H{"tweets": messages})
}

func getUsersTweets(c *gin.Context) {
	name := c.Param("username")
	user := models.User{}
	if err := config.DB.Where("username = ?", name).First(&user).Error; err != nil {
		c.JSON(200, gin.H{"message": "user does not exist"})
		return
	}

	num_msgs := c.Request.URL.Query().Get("num_msgs")
	int_num_msgs, err := strconv.Atoi(num_msgs)
	if num_msgs == "" || err != nil {
		int_num_msgs = 30
	}

	var messages []models.Message
	if err := config.DB.Where("author_id = ?", user.ID).Order("pub_date desc").Limit(int_num_msgs).Preload("Author").Find(&messages).Error; err != nil {
		errorCheck(err)
	}

	c.JSON(200, gin.H{"tweets": messages})
}

func followUser(c *gin.Context) {
	userID := getUserIdIfLoggedIn(c)
	whomName := c.Param("username")
	whomID := GetUserIdByName(whomName)

	if doesUsersFollow(userID, whomID) {
		c.JSON(200, gin.H{"message": "user already followed"})
		return
	}

	if whomID == "-1" {
		c.JSON(400, gin.H{"message": "user does not exist"})
		return
	}

	//convert userid and whomid to int
	userIDInt, err := strconv.Atoi(userID)
	errorCheck(err)
	whomIDInt, err := strconv.Atoi(whomID)
	errorCheck(err)

	Follower := models.Follower{
		Who_id:  userIDInt,
		Whom_id: whomIDInt,
	}

	if err := config.DB.Create(&Follower).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to follow user"})
		return
	}

	c.JSON(200, gin.H{"message": "followed user"})
}

func doesUsersFollow(whoID string, whomID string) bool {
	var follower models.Follower
	if err := config.DB.Where("who_id = ? and whom_id = ?", whoID, whomID).First(&follower).Error; err != nil {
		return false
	}
	return true
}

func unfollowUser(c *gin.Context) {
	userID := getUserIdIfLoggedIn(c)
	whomName := c.Param("username")
	whomID := GetUserIdByName(whomName)
	if !doesUsersFollow(userID, whomID) {
		c.JSON(200, gin.H{"message": "user doesn't follow the target"})
		return
	}
	log.Println(whomID)
	if whomID == "-1" {
		c.JSON(200, gin.H{"message": "user you are trying to follow does not exist"})
		return
	}

	result := config.DB.Where("who_id = ? AND whom_id = ?", userID, whomID).Delete(&models.Follower{})
	if result.Error != nil {
		errorCheck(result.Error)
	}

	c.JSON(200, gin.H{"message": "unfollowed user"})
}

func postMessage(c *gin.Context) {
	userID := getUserIdIfLoggedIn(c)
	if userID == "-1" {
		c.JSON(401, gin.H{"message": "user not logged in"})
		return
	}

	text := c.PostForm("text")
	//Convert userid to int
	authorID, err := strconv.Atoi(userID)
	errorCheck(err)

	flagged := 0
	log.Println("tweet attempting to be posted:" + text)
	//convert time.Now().Unix() to int
	pubDate := int(time.Now().Unix())

	message := models.Message{
		Author_id:   authorID,
		Author_name: GetUsernameByID(userID),
		Text:        text,
		Pub_date:    pubDate,
		Flagged:     flagged,
	}

	/* result := config.DB.Create(message)
	if result.Error != nil {
		c.JSON(500, gin.H{"message": "error creating message"})
		return
	} */

	if err := config.DB.Create(&message).Error; err != nil {
		c.JSON(400, gin.H{"error": "unable to create message"})
		return
	}

	c.JSON(200, gin.H{"message": "message posted"})
}

func GetUsernameByID(id string) string {
	var user models.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return "-1"
	}
	return user.Username
}

func login(c *gin.Context) {

	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "username or password is empty"})
		return
	}

	var user models.User
	err := config.DB.Where("username = ? AND pw_hash = ?", username, HashPassword(password)).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.SetCookie("user_id", "", -1, "/", "localhost", false, false)
			c.JSON(401, gin.H{"error": "username or password is incorrect"})
			return
		} else {
			errorCheck(err)
		}
	}

	//convert user.ID as uint to string
	userID := strconv.Itoa(int(user.ID))

	c.SetCookie("user_id", userID, 3600, "/", "localhost", false, false)
	c.JSON(200, gin.H{"user_id": user.ID})
}

func register(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	password2 := c.PostForm("password2")

	//check if username and password are empty
	if username == "" || password == "" || password2 == "" {
		c.JSON(400, gin.H{"error": "username or password is empty"})
		return
	} else if email == "" || !strings.Contains(email, "@") {
		c.JSON(400, gin.H{"error": "email is empty or invalid"})
		return
	} else if password != password2 {
		c.JSON(400, gin.H{"error": "passwords don't match"})
		return
	} else if getUserByName(username) != nil {
		c.JSON(400, gin.H{"error": "username already exists"})
		return
	}

	passwordHashString := HashPassword(password)
	log.Println(passwordHashString)

	user := models.User{
		Username: username,
		Email:    email,
		Pw_hash:  passwordHashString,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(400, gin.H{"error": "unable to create user"})
		return
	}

	c.JSON(200, gin.H{"message": "user registered"})
}

func getUserByName(userName string) *models.User {
	user := &models.User{}
	result := config.DB.Where("username = ?", userName).First(user)
	if result.Error != nil {
		fmt.Println("didn't find user with username: " + userName + ": this is expected for new users")
		return nil
	}
	return user
}

func getUserIdIfLoggedIn(c *gin.Context) string {
	userid, err := c.Cookie("user_id")
	log.Println("cookie user_id: " + userid)
	errorCheck(err)
	if userid == "" || userid == "-1" {
		c.JSON(401, gin.H{"error": "not logged in"})
		return "-1"
	}
	return userid

}

func GetUserIdByName(username string) string {
	var user models.User
	err := config.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		return "-1"
	}
	return strconv.Itoa(int(user.ID))
}

func getAllFollowing(c *gin.Context) {
	num_followers := c.Request.URL.Query().Get("num_followers")
	int_followers, err := strconv.Atoi(num_followers)
	if num_followers == "" || err != nil {
		int_followers = 30
	}
	userID := getUserIdIfLoggedIn(c)
	if userID == "-1" {
		c.JSON(401, gin.H{"error": "user not logged in"})
		return
	}

	following := []models.User{}
	err = config.DB.Table("users").
		Select("users.*").
		Joins("JOIN followers ON users.user_id = followers.whom_id").
		Where("followers.who_id = ?", userID).
		Limit(int_followers).
		Scan(&following).
		Error

	if err != nil {
		c.JSON(500, gin.H{"error": "unable to retrieve following"})
		return
	}
	c.JSON(200, gin.H{"following": following})

}

func logout(c *gin.Context) {
	c.SetCookie("user_id", "", -1, "/", "localhost", false, false)
}

func getAllUsers(c *gin.Context) {
	users := []models.User{}
	err := config.DB.Find(&users).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "unable to retrieve users"})
		return
	}
	c.JSON(200, gin.H{"users": users})
}
