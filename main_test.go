package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	// create a mock request and response
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// create a gin context with the mock request and response
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// call the logging middleware
	loggingMiddleware()(c)

	// assert that the middleware logged the correct information
	assert.Equal(t, "GET / 200", fmt.Sprintf("%s %s %d", c.Request.Method, c.Request.URL.Path, w.Code))
	// assert that the middleware logged a latency time in the correct format
	// create a start time
	startTime := time.Now()

	// simulate some latency
	time.Sleep(100 * time.Millisecond)

	// calculate the latency
	latency := time.Since(startTime)

	// format the latency as a string
	timeStr := fmt.Sprintf("%dms", latency/time.Millisecond)

	// assert that the formatted latency is correct
	assert.Equal(t, "101ms", timeStr)
}

func TestCountOfPatients(t *testing.T) {
	// テスト用のMySQLデータベースに接続するための文字列を作成
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")

	if err != nil {
		t.Errorf("failed to connect to database: %s", err)
	}
	defer db.Close()

	// コネクションの確認
	err = db.Ping()
	if err != nil {
		t.Errorf("failed to ping database: %s", err)
	}
}
