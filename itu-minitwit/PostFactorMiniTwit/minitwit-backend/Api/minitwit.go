package main

import (
	"crypto/sha256"
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	router := gin.Default()

	// endpoints
	router.GET("/", getTimeline)
	router.GET("/public", getPublicTimeline)
	router.GET("/user/:username", getUsersTweets)
	router.POST("/user/:username/follow", followUser)
	router.POST("/user/:username/unfollow", unfollowUser)
	router.POST("/add_message", postMessage)
	router.POST("/login", login)
	router.POST("/register", register)
	router.GET("/logout", logout)
	router.GET("/RESET", init_db)

	// middleware
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "DELETE"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Run("0.0.0.0:8080")
}

// Capitalized names are public, lowercase are privat
type User struct {
	User_id  int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Pw_hash  string `json:"pw_hash"`
}

type follower struct {
	Who_id  int `json:"who_id"`
	Whom_id int `json:"whom_id"`
}

type Message struct {
	Message_id int    `json:"message_id"`
	Author_id  int    `json:"author_id"`
	Text       string `json:"text"`
	Pub_date   int    `json:"pub_date"`
	Flagged    int    `json:"flagged"`
	Author     User   `json:"author"`
}

var dbPath = "./../../tmp/minitwit.db"
var DB *sql.DB // global DB variable
var PER_PAGE = 30
var DEBUG = true
var SECRET_KEY = "development key"

func connect_db() error {
	db, err := sql.Open("sqlite3", dbPath)
	errorCheck(err)

	DB = db
	return nil
}

func init_db(c *gin.Context) {
	connect_db()
	// create tables
	sqlStmt2 := `
	drop table if exists user;
	drop table if exists message;
	drop table if exists follower;
	create table if not exists user (user_id integer not null primary key, username text, email text, pw_hash text);
	create table if not exists message (message_id integer not null primary key, author_id integer, text text, pub_date integer, flagged integer);
	create table if not exists follower (who_id integer, whom_id integer);
	INSERT INTO user (username, email, pw_hash)
	VALUES
	("Benjamin", "bekj@itu.dk", "1a86a3c5991086b3afc27fa1341d9e80e68ab278ab836ca4a7c9498db7f607bf"),
	("Oliver", "ojoe@itu.dk", "1a86a3c5991086b3afc27fa1341d9e80e68ab278ab836ca4a7c9498db7f607bf"),
	("Silas", "sipn@itu.dk", "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"),
	("Janus", "januh@itu.dk", "850ca8e0ec3f2106b6a230cd36a86cf163b3b005dd2b6c94eafaf604cbdf03a8");
	INSERT INTO message (author_id, text, pub_date, flagged)
	VALUES
	(1, "I like apples", 123456789, 0),
	(2, "I like tarteletter", 123456789, 0),
	(3, "I like Pizza", 123456789, 0),
	(4, "I like bananas ", 123456789, 0);
	INSERT INTO follower (who_id, whom_id)
	VALUES
	(1, 2),
	(1, 3),
	(1, 4),
	(2, 1),
	(2, 3),
	(2, 4),
	(3, 1),
	(3, 2),
	(3, 4),
	(4, 1),
	(4, 2);
	`
	_, err := DB.Exec(sqlStmt2)
	errorCheck(err)

}

func errorCheck(err error) {
	if err != nil {
		log.Println(err)
	}
}

// endpoints
func getTimeline(c *gin.Context) {
	connect_db()
	//query database
	// check cookie for session,
	userID := getUserIdIfLoggedIn(c)

	rows, err := DB.Query(`select message.*, user.* from message, user
		where message.flagged = 0 and message.author_id = user.user_id and (
		user.user_id = ? or
		user.user_id in (select whom_id from follower
		where who_id = ?))
		order by message.pub_date desc limit ?`, userID, userID, PER_PAGE)
	errorCheck(err)
	defer rows.Close()
	messages := make([]Message, 0)
	for rows.Next() {
		msg := Message{}
		user := User{}
		err = rows.Scan(&msg.Message_id, &msg.Author_id, &msg.Text, &msg.Pub_date, &msg.Flagged, &user.User_id, &user.Username, &user.Email, &user.Pw_hash)
		errorCheck(err)
		log.Println(msg)
		msg.Author = user

		messages = append(messages, msg)
	}
	c.JSON(200, gin.H{"tweets": messages})
}

func getPublicTimeline(c *gin.Context) {
	connect_db()
	log.Println("connect_db done")
	rows, err := DB.Query(`select message.*, user.* from message, user
	where message.flagged = 0 and message.author_id = user.user_id
	order by message.pub_date desc limit ?`, PER_PAGE)
	errorCheck(err)

	// make a empty slice of messages
	messages := make([]Message, 0)

	for rows.Next() {
		msg := Message{}
		user := User{}
		err = rows.Scan(&msg.Message_id, &msg.Author_id, &msg.Text, &msg.Pub_date, &msg.Flagged, &user.User_id, &user.Username, &user.Email, &user.Pw_hash)
		log.Println(msg)
		msg.Author = user

		messages = append(messages, msg)
	}
	log.Println("printing messages")
	log.Println(messages)

	errorCheck(err)

	c.JSON(200, gin.H{"tweets": messages})

	defer rows.Close()
}

func getUsersTweets(c *gin.Context) {
	name := c.Param("username")
	connect_db()
	userID := getUserIdByName(name)
	if userID == "-1" {
		c.JSON(200, gin.H{"message": "user does not exist"})
		return
	}
	rows, err := DB.Query(`select message.*, user.* from message, user where message.author_id = ? and message.author_id = user.user_id order by message.pub_date desc limit ?`, userID, PER_PAGE)
	errorCheck(err)
	defer rows.Close()
	messages := make([]Message, 0)
	for rows.Next() {
		msg := Message{}
		user := User{}
		err = rows.Scan(&msg.Message_id, &msg.Author_id, &msg.Text, &msg.Pub_date, &msg.Flagged, &user.User_id, &user.Username, &user.Email, &user.Pw_hash)
		errorCheck(err)
		msg.Author = user

		messages = append(messages, msg)
	}
	c.JSON(200, gin.H{"tweets": messages})

}

func followUser(c *gin.Context) {
	connect_db()

	userid := getUserIdIfLoggedIn(c)
	whom_name := c.Param("username")
	whom_id := getUserIdByName(whom_name)
	if doesUsersFollow(userid, whom_id) {
		c.JSON(200, gin.H{"message": "user already followed"})
		return
	}

	if whom_id == "-1" {
		c.JSON(200, gin.H{"message": "user does not exist"})
		return
	}
	stmt, err := DB.Prepare(`insert into follower (who_id, whom_id) values (?, ?)`)
	errorCheck(err)
	defer stmt.Close()

	_, err = stmt.Exec(userid, whom_id)
	errorCheck(err)

	c.JSON(200, gin.H{"message": "followed user"})
}

func doesUsersFollow(who_id string, whom_id string) bool {
	connect_db()

	row := DB.QueryRow(`select * from follower where who_id = ? and whom_id = ?`, who_id, whom_id)

	follower := follower{}
	err := row.Scan(&follower.Who_id, &follower.Whom_id)
	return err == nil

}

func unfollowUser(c *gin.Context) {
	connect_db()

	userid := getUserIdIfLoggedIn(c)
	whom_name := c.Param("username")
	whom_id := getUserIdByName(whom_name)
	if !doesUsersFollow(userid, whom_id) {
		c.JSON(200, gin.H{"message": "user dosent follow the target"})
		return
	}
	log.Println(whom_id)
	if whom_id == "-1" {
		c.JSON(200, gin.H{"message": "user does not exist"})
		return
	}

	stmt, err := DB.Prepare(`Delete FROM follower WHERE who_id = ? and whom_id = ?`)
	errorCheck(err)
	defer stmt.Close()

	_, err = stmt.Exec(userid, whom_id)
	errorCheck(err)

	c.JSON(200, gin.H{"message": "unfollowed user"})

}

func postMessage(c *gin.Context) {
	connect_db()

	userid := getUserIdIfLoggedIn(c)

	if userid == "-1" {
		c.JSON(401, gin.H{"message": "user not logged in"})
		return
	}

	text := c.PostForm("text")
	authorid := userid
	pub_date := time.Now().Unix()
	flagged := 0
	log.Println("text:" + text)

	stmt, err := DB.Prepare(`insert into message (author_id, text, pub_date, flagged) values (?, ?, ?, ?)`)
	errorCheck(err)
	defer stmt.Close()

	_, err = stmt.Exec(authorid, text, pub_date, flagged)
	errorCheck(err)

	c.JSON(200, gin.H{"message": "message posted"})

}

func login(c *gin.Context) {
	connect_db()

	username := c.PostForm("username")
	password := c.PostForm("password")
	//convert password to byte[]

	passwordHash := sha256.Sum256([]byte(password)) //hash password1
	passwordHashString := string(passwordHash[:])
	//check if username and password are empty
	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "username or password is empty"})
		return
	}

	userid := DB.QueryRow(`select user_id from user where user.Username = ? and user.pw_hash = ?`, username, passwordHashString)

	var userIdAsInt int
	err := userid.Scan(&userIdAsInt)
	if err != nil {
		c.SetCookie("user_id", "", -1, "/", "localhost", false, false)
		return
	}
	// succes: set cookie
	c.SetCookie("user_id", strconv.Itoa(userIdAsInt), 3600, "/", "localhost", false, false)
	c.JSON(200, gin.H{"user_id": userIdAsInt})
}

func register(c *gin.Context) {
	connect_db()

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

	passwordHash := sha256.Sum256([]byte(password))
	//convert back to string
	passwordHashString := string(passwordHash[:])
	log.Println(passwordHashString)

	stmt, err := DB.Prepare(`insert into user (username, email, pw_hash) values (?, ?, ?)`)
	errorCheck(err)
	defer stmt.Close()
	_, err = stmt.Exec(username, email, passwordHashString)
	errorCheck(err)

	c.JSON(200, gin.H{"message": "user registered"})
}

func getUserByName(userName string) *sql.Row {
	connect_db()
	row := DB.QueryRow(`select * from user where user.username = ?`, userName)
	user := User{}
	err := row.Scan(&user.User_id, &user.Username, &user.Email, &user.Pw_hash)
	if err != nil {
		return nil
	}
	return row
	// if user exists, return user, else return nil

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

func getUserIdByName(username string) string {
	connect_db()
	row := DB.QueryRow(`select * from user where user.username = ?`, username)

	user := User{}
	err := row.Scan(&user.User_id, &user.Username, &user.Email, &user.Pw_hash)
	if err != nil {
		return "-1"
	}

	return strconv.Itoa(user.User_id)
}

func logout(c *gin.Context) {
	c.SetCookie("user_id", "", -1, "/", "localhost", false, false)
}
