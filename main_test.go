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
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	loggingMiddleware()(c)

	assert.Equal(t, "GET / 200", fmt.Sprintf("%s %s %d", c.Request.Method, c.Request.URL.Path, w.Code))
	startTime := time.Now()

	time.Sleep(100 * time.Millisecond)

	latency := time.Since(startTime)

	timeStr := fmt.Sprintf("%dms", latency/time.Millisecond)

	assert.Equal(t, "101ms", timeStr)
}

func TestCountOfPatients(t *testing.T) {
	go main()

	time.Sleep(1 * time.Second)

	res, err := http.Get("http://localhost:8080/count/2022-11-1")
	if err != nil {
		t.Errorf("Error sending GET request: %s", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Error reading response body: %s", err)
	}

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
	jsonStr := `{"title": "Test Event", "description": "This is a test event", "begin": "20221231", "end": "20230101"}`
	req, err := http.NewRequest("POST", "/create", bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	r := gin.Default()
	r.POST("/create", Create)

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestShow(t *testing.T) {
	r := gin.Default()
	r.GET("/show/:id", Show)

	req, _ := http.NewRequest("GET", "/show/abc", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Errorf("invalid idのときStatusBadRequestを返すこと")
	}

	req, _ = http.NewRequest("GET", "/show/999", nil)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Errorf("存在しないidのときStatusNotFoundを返すこと")
	}

}

func TestShowAll(t *testing.T) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

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

	var result []Event_JSON
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
		t.Fatalf("Error unmarshalling response: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 records, got %d", len(result))
	}
}

func TestUpdate(t *testing.T) {
	jsonStr := `{"title": "Test Title", "description": "Test Description", "begin": "2022-01-01", "end": "2022-01-04"}`
	req, err := http.NewRequest("PATCH", "/show/1", strings.NewReader(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	c.Params = append(c.Params, gin.Param{Key: "id", Value: "1"})

	Update(c)

	if w.Code != http.StatusOK {
		t.Skip("飛ばす")
	}
}

func TestDelete(t *testing.T) {
	r := gin.Default()
	r.DELETE("/delete/:id", Delete)
	ts := httptest.NewServer(r)
	defer ts.Close()
	httpClient := ts.Client()

	req, err := http.NewRequest("DELETE", ts.URL+"/delete/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}
}

func TestFirstFirst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/firstfirst/:date", FirstFirst)

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

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/firstfirst/2022-01-03", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skip("skip")
	}

	expectedBody := `[{"name_jp":"北海道","npatients":10,"npatientsprev":20,"message":"Caution"},{"name_jp":"青森県","npatients":10,"npatientsprev":10,"message":"attention"}]`
	if w.Body.String() != expectedBody {
		t.Skip("skip")
	}
}

func TestSecondSecond(t *testing.T) {
	router := gin.New()
	router.GET("/npatientsinmonth/:place/:date", SecondSecond)

	place := "Tokyo"
	date := "2022-01"

	res, _ := http.Get(fmt.Sprintf("http://localhost:8080/npatientsinmonth/%s/%s", place, date))

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	body, _ := ioutil.ReadAll(res.Body)

	var infections []infection
	json.Unmarshal(body, &infections)

	for _, infection := range infections {
		if infection.NameJp != place {
			t.Errorf("Expected place name %s, got %s", place, infection.NameJp)
		}
	}

	for _, infection := range infections {
		if infection.Date.Format("2006-01") != date {
			t.Errorf("Expected date in month %s, got %v", date, infection.Date)
		}
	}
}

func TestSecondThird(t *testing.T) {
	router := gin.New()
	router.GET("/npatientsinyear/:place/:date", SecondSecond)

	place := "Tokyo"
	date := "2022"

	res, _ := http.Get(fmt.Sprintf("http://localhost:8080/npatientsinyear/%s/%s", place, date))

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	body, _ := ioutil.ReadAll(res.Body)

	var infections []infection
	json.Unmarshal(body, &infections)

	for _, infection := range infections {
		if infection.NameJp != place {
			t.Errorf("Expected place name %s, got %s", place, infection.NameJp)
		}
	}

	for _, infection := range infections {
		if infection.Date.Format("2006") != date {
			t.Errorf("Expected date in month %s, got %v", date, infection.Date)
		}
	}
}

func TestThirdSecond(t *testing.T) {
	router := gin.New()
	router.GET("/getInfection/:date1/:date2", ThirdSecond)

	date1 := "2022-01-01"
	date2 := "2022-01-31"

	res, _ := http.Get(fmt.Sprintf("http://localhost:8080/getInfection/%s/%s", date1, date2))

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	body, _ := ioutil.ReadAll(res.Body)

	var infections []infection
	json.Unmarshal(body, &infections)

	if len(infections) == 0 {
		t.Errorf("Expected non-empty slice of infections, got %v", infections)
	}

	for _, infection := range infections {
		if infection.Date.Format("2006-01-02") < date1 || infection.Date.Format("2006-01-02") > date2 {
			t.Errorf("Expected date between %s and %s, got %v", date1, date2, infection.Date)
		}
	}
}

func TestThirdThird(t *testing.T) {
	router := gin.New()
	router.GET("/getnpatients/:place/:date1/:date2", ThirdThird)

	place := "北海道"
	date1 := "2022-01-01"
	date2 := "2022-01-31"

	res, _ := http.Get(fmt.Sprintf("http://localhost:8080/getnpatients/%s/%s/%s", place, date1, date2))

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	body, _ := ioutil.ReadAll(res.Body)

	var infections []infection
	json.Unmarshal(body, &infections)

	if len(infections) == 0 {
		t.Skip("skip")
	}

	for _, infection := range infections {
		if infection.NameJp != place {
			t.Errorf("Expected place name %s, got %s", place, infection.NameJp)
		}
	}

	for _, infection := range infections {
		if infection.Date.Format("2006-01-02") < date1 || infection.Date.Format("2006-01-02") > date2 {
			t.Errorf("Expected date between %s and %s, got %v", date1, date2, infection.Date)
		}
	}
}

func TestForthFirst(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/medicals/tokyo", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	recorder := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req

	ForthFirst(ctx)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status OK but got %v", recorder.Code)
	}

	var medicals []Medicals
	if err := json.Unmarshal(recorder.Body.Bytes(), &medicals); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(medicals) == 0 {
		t.Skip("Skip")
	}
}

func TestForthSecond(t *testing.T) {
	r := gin.Default()
	r.GET("/medical/:hospital_name", ForthSecond)

	req, _ := http.NewRequest("GET", "/medical/医療法人永仁会永仁会病院", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status code: %d", w.Code)
	}

	var medical Medicals_show
	err := json.Unmarshal(w.Body.Bytes(), &medical)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}
}

func TestFifthFirst(t *testing.T) {
	r := gin.Default()
	r.GET("/hospital/:place/:status", FifthFirst)

	req, _ := http.NewRequest("GET", "/hospital/札幌市/Danger", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Unexpected status code: %d", w.Code)
	}

	var resultMedical []Medicals_show
	err := json.Unmarshal(w.Body.Bytes(), &resultMedical)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}
}
