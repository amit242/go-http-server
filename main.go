package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
	"github.com/russross/blackfriday"
)

var (
	db *sql.DB
)

func dbFunc(c *gin.Context) {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS ticks (tick timestamp)"); err != nil {
		log.Print("Error creating database table:", err)
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error creating database table: %q", err))
		return
	}

	if _, err := db.Exec("INSERT INTO ticks VALUES (now())"); err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error incrementing tick: %q", err))
		return
	}

	rows, err := db.Query("SELECT tick FROM ticks")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("Error reading ticks: %q", err))
		return
	}

	defer rows.Close()
	for rows.Next() {
		var tick time.Time
		if err := rows.Scan(&tick); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error scanning ticks: %q", err))
			return
		}
		c.String(http.StatusOK, fmt.Sprintf("Read from DB: %s\n", tick.String()))
	}
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	var err error

	dburl := os.Getenv("DATABASE_URL")
	log.Print("main | value of DATABASE_URL:", dburl)
	db, err = sql.Open("postgres", dburl)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.GET("/mark", func(c *gin.Context) {
		c.String(http.StatusOK, string(blackfriday.MarkdownBasic([]byte("**hi!**"))))
		log.Print("AMIT:: mark was called")
	})

	router.GET("/db", dbFunc)

	router.POST("/ipaddress", func(c *gin.Context) {
		ip := c.Query("ipaddress")
		hostName := c.Query("host")
		log.Printf("AMIT:: /ipaddress was called with ip=%s, host=%s", ip, hostName)

		if ip == "" {
			c.String(http.StatusBadRequest, string("no ip was present in query string"))
			return
		}

		if hostName == "" {
			log.Printf("hostname not given for ip=%s", ip)
		}

		if _, err := db.Exec("CREATE TABLE IF NOT EXISTS piinfo (ip char(15), hostname varchar(50), timestamp timestamp)"); err != nil {
			log.Print("Error creating database table:", err)
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error creating database table: %q", err))
			return
		}

		if _, err := db.Exec(fmt.Sprintf("INSERT INTO piinfo VALUES ('%s', '%s', now())", ip, hostName)); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("table insert failed for table piinfo: %q", err))
			return
		}

		c.String(http.StatusOK, string(fmt.Sprintf("saved ip=%s, host=%s", ip, hostName)))

	})

	router.GET("/ipaddress", func(c *gin.Context) {
		rows, err := db.Query("SELECT * FROM piinfo order by timestamp desc limit 5")
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error reading piinfo: %q", err))
			return
		}

		defer rows.Close()
		for rows.Next() {
			var (
				ip        string
				hostname  string
				timestamp time.Time
			)
			if err := rows.Scan(&ip, &hostname, &timestamp); err != nil {
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error scanning piinfo: %q", err))
				return
			}
			c.String(http.StatusOK, fmt.Sprintf("ip: %s,\t hostname:%s, \t last updated:%s\n", ip, hostname, timestamp.String()))
		}
	})

	router.Run(":" + port)
}
