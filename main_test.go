package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCountOfPatients(t *testing.T) {
	// テスト用のgin.Engineを作成
	r := gin.New()
	r.GET("/count/:date", CountOfPatients)

	// テスト用のリクエストを作成
	req, err := http.NewRequest("GET", "/count/2022-12-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	// テスト
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

// 以下、他の関数も同様にテストを追加していけばよいです。
