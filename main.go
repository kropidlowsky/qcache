package main

import (
	"log"
	"net/http"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
	"github.com/kropidlowsky/qcache/pkg/qcache"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type user struct {
	ID   string `json:"id" gorm:"primaryKey,uniqueIndex,column:id"`
	Name string `json:"name"`
}

var db *gorm.DB

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&user{})

	handler := newHandler(db)

	r := gin.New()

	r.GET("/users", handler.listUsersHandler)
	r.POST("/users", handler.createUserHandler)
	r.GET("/users/:id", handler.getUserHandler)

	r.Run()
}

type handler struct {
	db *gorm.DB
	qc *qcache.QCache
}

func newHandler(db *gorm.DB) *handler {
	qc, err := qcache.NewQCache(db.Statement.Context, bigcache.DefaultConfig(10*time.Minute), "user", true)
	if err != nil {
		log.Fatal(err)
	}
	return &handler{db, qc}
}

func (h *handler) getUserHandler(c *gin.Context) {
	id := c.Param("id")
	u := user{}

	err := h.qc.Find(h.db.First, &u, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}

	c.JSON(http.StatusOK, u)
}

func (h *handler) createUserHandler(c *gin.Context) {
	var user user

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if result := db.Create(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, &user)
}

func (h *handler) listUsersHandler(c *gin.Context) {
	var users []user

	if result := h.db.Find(&users); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, &users)
}
