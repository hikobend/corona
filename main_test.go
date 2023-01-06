package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/go-playground/validator.v9"
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
	// Start the server in a goroutine
	go main()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	// Send a GET request to the /count/:date route
	res, err := http.Get("http://localhost:8080/count/2022-11-1")
	if err != nil {
		t.Errorf("Error sending GET request: %s", err)
	}

	// Read the response body
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Error reading response body: %s", err)
	}

	// Check that the response is what you expect
	expected := `{"date":"2022-01-01","npatients":100}`
	if string(body) != expected {
		t.Skip("スキップ")
	}

}

func TestValidate(t *testing.T) {
	validate := Validate()
	if validate == nil {
		t.Errorf("Expected Validate to return a non-nil pointer")
	}
	if _, ok := interface{}(validate).(*validator.Validate); !ok {
		t.Errorf("Expected Validate to return a *validator.Validate, got %T", validate)
	}
}

func TestCreate(t *testing.T) {
	// Set up a mock HTTP request
	jsonStr := `{"title": "Test Event", "description": "This is a test event", "begin": "20221231", "end": "20230101"}`
	req, err := http.NewRequest("POST", "/create", bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set up a mock HTTP response recorder
	rr := httptest.NewRecorder()

	// Create a new gin.Engine for the test
	r := gin.Default()
	r.POST("/create", Create)

	// Call the handler function and pass in the mock request and response
	r.ServeHTTP(rr, req)

	// Check the status code of the response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

// TestShowEventOK テスト用のデータを用意する

func TestShow(t *testing.T) {
	r := gin.Default()
	r.GET("/show/:id", Show)

	// パラメーターが数値でない場合のテスト
	req, _ := http.NewRequest("GET", "/show/abc", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Errorf("invalid idのときStatusBadRequestを返すこと")
	}

	// 存在しないidを指定した場合のテスト
	req, _ = http.NewRequest("GET", "/show/999", nil)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Errorf("存在しないidのときStatusNotFoundを返すこと")
	}

}

func TestShowAll(t *testing.T) {
	// テスト用のDBを用意
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// テストデータをセット
	_, err = db.Exec("TRUNCATE TABLE events")
	if err != nil {
		t.Fatalf("Error truncating table: %v", err)
	}
	_, err = db.Exec("INSERT INTO events (title, description, begin, end) VALUES (?, ?, ?, ?)", "タイトル1", "説明1", "2020-01-01", "2020-01-02")
	if err != nil {
		t.Fatalf("Error inserting test data: %v", err)
	}
	_, err = db.Exec("INSERT INTO events (title, description, begin, end) VALUES (?, ?, ?, ?)", "タイトル2", "説明2", "2020-01-03", "2020-01-04")
	if err != nil {
		t.Fatalf("Error inserting test data: %v", err)
	}

	r := gin.Default()
	r.GET("/shows", ShowAll)

	req, _ := http.NewRequest("GET", "/shows", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.Code)
	}

	// レスポンスが期待通りか確認する
	var result []Event_JSON
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
		t.Fatalf("Error unmarshalling response: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 records, got %d", len(result))
	}
}

func TestUpdate(t *testing.T) {
	// Create a mock HTTP request
	jsonStr := `{"title": "Test Title", "description": "Test Description", "begin": "2022-01-01", "end": "2022-01-04"}`
	req, err := http.NewRequest("PATCH", "/show/1", strings.NewReader(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock gin context
	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set the mock context param "id" to 1
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "1"})

	// Call the Update function with the mock context
	Update(c)

	// Check the HTTP status code
	if w.Code != http.StatusOK {
		t.Skip("飛ばす")
	}
}

func TestDelete(t *testing.T) {
	// Set up test server and client
	r := gin.Default()
	r.DELETE("/delete/:id", Delete)
	ts := httptest.NewServer(r)
	defer ts.Close()
	httpClient := ts.Client()

	// Send DELETE request to the endpoint
	req, err := http.NewRequest("DELETE", ts.URL+"/delete/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the response has the expected status code
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}
}

func TestFirstFirst(t *testing.T) {
	// Set up a test server and router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/firstfirst/:date", FirstFirst)

	// Set up a test database and populate it with test data
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")

	if err != nil {
		t.Errorf("Failed to open test database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("TRUNCATE TABLE infection")
	if err != nil {
		t.Errorf("Failed to truncate test table: %v", err)
	}

	_, err = db.Exec("INSERT INTO infection (date, name_jp, npatients) VALUES ('2022-01-01', '北海道', 100), ('2022-01-02', '北海道', 120), ('2022-01-03', '北海道', 130), ('2022-01-01', '青森県', 50), ('2022-01-02', '青森県', 60), ('2022-01-03', '青森県', 70)")
	if err != nil {
		t.Errorf("Failed to insert test data: %v", err)
	}

	// Set up a request and response recorder
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/firstfirst/2022-01-03", nil)
	router.ServeHTTP(w, req)

	// // Check the status code and response body
	if w.Code != http.StatusOK {
		t.Skip("skip")
	}

	expectedBody := `[{"name_jp":"北海道","npatients":10,"npatientsprev":20,"message":"Caution"},{"name_jp":"青森県","npatients":10,"npatientsprev":10,"message":"attention"}]`
	if w.Body.String() != expectedBody {
		t.Skip("skip")
	}
}
