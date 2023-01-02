package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/go-playground/validator.v9"
)

type Deceased struct {
	ErrorInfo struct {
		ErrorFlag    string      `json:"errorFlag"`
		ErrorCode    interface{} `json:"errorCode"`
		ErrorMessage interface{} `json:"errorMessage"`
	} `json:"errorInfo"`
	ItemList []struct {
		Date        string `json:"date"`
		DataName    string `json:"dataName"`
		InfectedNum string `json:"infectedNum"`
		DeceasedNum string `json:"deceasedNum"`
	} `json:"itemList"`
}

type Npatients struct {
	ErrorInfo struct {
		ErrorFlag    string `json:"errorFlag"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
	} `json:"errorInfo"`
	ItemList []struct {
		Date      string `json:"date"`
		NameJp    string `json:"name_jp"`
		Npatients string `json:"npatients"`
	} `json:"itemList"`
}

type infection struct {
	Date      time.Time `json:"date"`
	NameJp    string    `json:"name_jp"`
	Npatients int       `json:"npatients"`
}

// type decease struct {
// 	Date        time.Time `json:"date"`
// 	DataName    string    `json:"data_name"`
// 	InfectedNum int       `json:"infected_mum"`
// 	DeceasedNum int       `json:"deceased_num"`
// }

type Event_JSON struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Begin       string `json:"begin" validate:"required"`
	End         string `json:"end" validate:"required"`
}

func main() {
	r := gin.Default()

	// ----------------------------------
	// 1-1
	// ----------------------------------

	r.GET("/firstfirst/:date", FirstFirst)                                        // 都道府県のマップを表示 色で危険地帯を視覚で把握可能 前々日比と前日比を算出して、前日比の方が多い場合、警告文字を変更する。その文字によって色を変える
	r.GET("/areanpatients/:place/:date", AreaNpatients)                           // 地方と日付を入力して、感染者を取得する
	r.GET("/areaaveragenpatients/:place/:date", AreaAverageNpatients)             // 地方と日付を入力して、感染者の平均を取得する
	r.GET("/areaaveragenpatientsover/:place/:date", AreaAverageNpatientsOver)     // 地方と日付を入力して、感染者の平均超えている都道府県を取得する
	r.GET("/leastattachday/:place/:count", LeastAttachDay)                        // 都道府県と日付を入力して、既定の感染者に到達した最短の日程を表示
	r.GET("/npatientsinyear/:place/:date", NpatientsInYear)                       // 年と都道府県を取得して、その年の感染者推移を取得
	r.GET("/averagenpatientsinyear/:place/:date", AverageNpatientsInYear)         // 年と都道府県を取得して、その年の平均感染者数を取得
	r.GET("/npatientsinmonth/:place/:date", NpatientsInMonth)                     // 年月と都道府県を取得して、その月の感染者数推移を取得
	r.GET("/averagenpatientsinmonth/:place/:date", AverageNpatientsInMonth)       // 年月と都道府県を取得して、その月の平均感染者数を取得
	r.GET("/count/:date", CountOfPatients)                                        // 日の感染者の合計
	r.GET("/diff/:place/:date1/:date2", Diff)                                     // 前日比を表示
	r.POST("/create", Create)                                                     // コロナに関するメモを追加
	r.GET("/show/:id", Show)                                                      // コロナに関するメモを表示 -> 日程の感染者数の推移を表示したい -> ボタンを設置してGetInfectionByDateに飛ばせないか？
	r.GET("/shows", ShowAll)                                                      // コロナに関するメモを表示
	r.PATCH("/show/:id", Update)                                                  // コロナに関するメモを変更
	r.DELETE("/delete/:id", Delete)                                               // コロナに関するメモを削除
	r.POST("/import", Import)                                                     // データをimport
	r.POST("/importdeceased", ImportDeceased)                                     // データをimport
	r.GET("/get/:date", GetInfectionByDate)                                       // 日付を選択し、感染者を取得 47都道府県　-> 47都道府県を並列処理で対処できないか
	r.GET("/getplaceanddate/:place/:date", GetInfectionByDateAndPlace)            // 日付と都道府県を選択し、感染者を取得
	r.GET("/getInfection/:date1/:date2", GetBetweenDateNpatients)                 // 期間を選択し、感染者を取得 47都道府県
	r.GET("/npatients/:place/:date", GetDateNpatients)                            // 日付と地域を選択し、感染者を取得
	r.GET("/npatientsthreeday/:place/:date", TheDayBeforeRatioPatients)           // 日付と地域を選択し、3日間の感染者を取得
	r.GET("/npatientsthreedayall/:date", TheDayBeforeRatioPatientsAll)            // 日付を選択し、3日間の感染者を取得 47都道府県
	r.GET("/getnpatients/:place/:date1/:date2", GetBetWeenDateNpatientsWithPlace) // 期間を選択し、感染者を取得
	r.GET("/getnpatientsasc/:date", GetNpatientsWithPlaceAsc)                     // 日付を選択して、感染者が少ない順に表示
	r.GET("/getnpatientsdesc/:date", GetNpatientsWithPlaceDesc)                   // 日付を選択して、感染者が多い順に表示
	r.GET("/setnpatientsasc/:date/:count", SetNpatientsAsc)                       // 日付と感染者を入力して、感染者が上回った都道府県を少ない順に表示
	r.GET("/setnpatientsdesc/:date/:count", SetNpatientsDesc)                     // 日付と感染者を入力して、感染者が上回った都道府県を多い順に表示
	r.GET("/setnpatientsunderasc/:date/:count", SetNpatientsUnderAsc)             // 日付と感染者を入力して、感染者が下回った都道府県を少ない順に表示
	r.GET("/setnpatientsunderdesc/:date/:count", SetNpatientsUnderDesc)           // 日付と感染者を入力して、感染者が下回った都道府県を多い順に表示
	r.GET("/averagenpatients/:date", AverageNpatients)                            // 日付を入力して、全国の感染者を上回った都道府県を表示
	r.GET("/averagenpatientsover/:date", AverageNpatientsOver)                    // 日付を入力して、全国の感染者を下回った都道府県を表示

	r.Run()
}

func FirstFirst(c *gin.Context) {

}

func AreaAverageNpatientsOver(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	var rows *sql.Rows

	switch place {
	case "北海道":
		rows, err = db.Query("select date, name_jp, npatients from infection where name_jp = '北海道' and date = ? and npatients > (select avg(npatients) from infection where name_jp = '北海道' and date = ?)", date, date)

	case "東北":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '青森県' or name_jp = '岩手県' or name_jp = '宮城県' or name_jp = '秋田県' or name_jp ='山形県' or name_jp = '福島県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '青森県' or name_jp = '岩手県' or name_jp = '宮城県' or name_jp = '秋田県' or name_jp ='山形県' or name_jp = '福島県') and date = ?)", date, date)

	case "関東":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '茨城県' or name_jp = '栃木県' or name_jp = '群馬県' or name_jp = '埼玉県' or name_jp ='千葉県' or name_jp = '東京都' or name_jp = '神奈川県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '茨城県' or name_jp = '栃木県' or name_jp = '群馬県' or name_jp = '埼玉県' or name_jp ='千葉県' or name_jp = '東京都' or name_jp = '神奈川県') and date = ?)", date, date)

	case "中部":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '新潟県' or name_jp = '富山県' or name_jp = '石川県' or name_jp = '福井県' or name_jp ='山梨県' or name_jp = '長野県' or name_jp = '岐阜県' or name_jp = '静岡県' or name_jp = '愛知県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '新潟県' or name_jp = '富山県' or name_jp = '石川県' or name_jp = '福井県' or name_jp ='山梨県' or name_jp = '長野県' or name_jp = '岐阜県' or name_jp = '静岡県' or name_jp = '愛知県') and date = ?)", date, date)

	case "近畿":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '三重県' or name_jp = '滋賀県' or name_jp = '京都府' or name_jp = '大阪府' or name_jp ='兵庫県' or name_jp = '奈良県' or name_jp = '和歌山県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '三重県' or name_jp = '滋賀県' or name_jp = '京都府' or name_jp = '大阪府' or name_jp ='兵庫県' or name_jp = '奈良県' or name_jp = '和歌山県') and date = ?)", date, date)

	case "中国":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '鳥取県' or name_jp = '島根県' or name_jp = '岡山県' or name_jp = '広島県' or name_jp ='山口県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '鳥取県' or name_jp = '島根県' or name_jp = '岡山県' or name_jp = '広島県' or name_jp ='山口県') and date = ?)", date, date)

	case "四国":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '徳島県' or name_jp = '香川県' or name_jp = '愛媛県' or name_jp = '高知県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '徳島県' or name_jp = '香川県' or name_jp = '愛媛県' or name_jp = '高知県') and date = ?)", date, date)

	case "九州":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '福岡県' or name_jp = '佐賀県' or name_jp = '長崎県' or name_jp = '熊本県' or name_jp ='大分県' or name_jp = '宮崎県' or name_jp = '鹿児島県' or name_jp = '沖縄県') and date = ? and npatients > (select avg(npatients) from infection where (name_jp = '福岡県' or name_jp = '佐賀県' or name_jp = '長崎県' or name_jp = '熊本県' or name_jp ='大分県' or name_jp = '宮崎県' or name_jp = '鹿児島県' or name_jp = '沖縄県') and date = ?)", date, date)
	}
	// インフェクションを取得
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer rows.Close()

	var result []infection
	for rows.Next() {
		var inf infection
		if err := rows.Scan(&inf.Date, &inf.NameJp, &inf.Npatients); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
		result = append(result, inf)
	}

	c.JSON(http.StatusOK, result) // 200
}

func AreaAverageNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	var avg float64

	switch place {
	case "北海道":
		err = db.QueryRow("select avg(npatients) from infection where name_jp = '北海道' and date = ?", date).Scan(&avg)

	case "東北":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '青森県' or name_jp = '岩手県' or name_jp = '宮城県' or name_jp = '秋田県' or name_jp ='山形県' or name_jp = '福島県') and date = ?", date).Scan(&avg)

	case "関東":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '茨城県' or name_jp = '栃木県' or name_jp = '群馬県' or name_jp = '埼玉県' or name_jp ='千葉県' or name_jp = '東京都' or name_jp = '神奈川県') and date = ?", date).Scan(&avg)

	case "中部":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '新潟県' or name_jp = '富山県' or name_jp = '石川県' or name_jp = '福井県' or name_jp ='山梨県' or name_jp = '長野県' or name_jp = '岐阜県' or name_jp = '静岡県' or name_jp = '愛知県') and date = ?", date).Scan(&avg)

	case "近畿":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '三重県' or name_jp = '滋賀県' or name_jp = '京都府' or name_jp = '大阪府' or name_jp ='兵庫県' or name_jp = '奈良県' or name_jp = '和歌山県') and date = ?", date).Scan(&avg)

	case "中国":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '鳥取県' or name_jp = '島根県' or name_jp = '岡山県' or name_jp = '広島県' or name_jp ='山口県') and date = ?", date).Scan(&avg)

	case "四国":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '徳島県' or name_jp = '香川県' or name_jp = '愛媛県' or name_jp = '高知県') and date = ?", date).Scan(&avg)

	case "九州":
		err = db.QueryRow("select avg(npatients) from infection where (name_jp = '福岡県' or name_jp = '佐賀県' or name_jp = '長崎県' or name_jp = '熊本県' or name_jp ='大分県' or name_jp = '宮崎県' or name_jp = '鹿児島県' or name_jp = '沖縄県') and date = ?", date).Scan(&avg)
	}
	// インフェクションを取得
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}

	c.JSON(http.StatusOK, gin.H{
		// "month":     date,
		// "place":     place,
		"npatients": avg,
	})
}

func AreaNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	var rows *sql.Rows

	switch place {
	case "北海道":
		rows, err = db.Query("select date, name_jp, npatients from infection where name_jp = '北海道' and date = ?", date)

	case "東北":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '青森県' or name_jp = '岩手県' or name_jp = '宮城県' or name_jp = '秋田県' or name_jp ='山形県' or name_jp = '福島県') and date = ?", date)

	case "関東":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '茨城県' or name_jp = '栃木県' or name_jp = '群馬県' or name_jp = '埼玉県' or name_jp ='千葉県' or name_jp = '東京都' or name_jp = '神奈川県') and date = ?", date)

	case "中部":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '新潟県' or name_jp = '富山県' or name_jp = '石川県' or name_jp = '福井県' or name_jp ='山梨県' or name_jp = '長野県' or name_jp = '岐阜県' or name_jp = '静岡県' or name_jp = '愛知県') and date = ?", date)

	case "近畿":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '三重県' or name_jp = '滋賀県' or name_jp = '京都府' or name_jp = '大阪府' or name_jp ='兵庫県' or name_jp = '奈良県' or name_jp = '和歌山県') and date = ?", date)

	case "中国":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '鳥取県' or name_jp = '島根県' or name_jp = '岡山県' or name_jp = '広島県' or name_jp ='山口県') and date = ?", date)

	case "四国":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '徳島県' or name_jp = '香川県' or name_jp = '愛媛県' or name_jp = '高知県') and date = ?", date)

	case "九州":
		rows, err = db.Query("select date, name_jp, npatients from infection where (name_jp = '福岡県' or name_jp = '佐賀県' or name_jp = '長崎県' or name_jp = '熊本県' or name_jp ='大分県' or name_jp = '宮崎県' or name_jp = '鹿児島県' or name_jp = '沖縄県') and date = ?", date)
	}
	// インフェクションを取得
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer rows.Close()

	var result []infection
	for rows.Next() {
		var inf infection
		if err := rows.Scan(&inf.Date, &inf.NameJp, &inf.Npatients); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
		result = append(result, inf)
	}

	c.JSON(http.StatusOK, result) // 200
}

func AverageNpatientsInYear(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	// SQLを実行
	var avg float64
	err = db.QueryRow("select avg(npatients) from infection where name_jp = ? and date like ?", place, date+"%").Scan(&avg)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"year":      date,
		"place":     place,
		"npatients": avg,
	})
}

func NpatientsInYear(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	rows, err := db.Query("select date, name_jp, npatients from infection where name_jp = ? and date like ?", place, date+"%")
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func LeastAttachDay(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	defer db.Close()

	place := c.Param("place")
	count := c.Param("count")

	// SQLを実行
	var min time.Time
	err = db.QueryRow("select min(date), npatients from infection where name_jp = ? and npatients > ?", place, count).Scan(&min)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}

	yyyymmdd := min.Format("2006-01-02")

	c.JSON(http.StatusOK, gin.H{
		"day":   yyyymmdd,
		"place": place,
	})
}

func AverageNpatientsInMonth(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	// SQLを実行
	var avg float64
	err = db.QueryRow("select avg(npatients) from infection where name_jp = ? and date like ?", place, date+"%").Scan(&avg)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"month":     date,
		"place":     place,
		"npatients": avg,
	})
}

func NpatientsInMonth(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place")

	rows, err := db.Query("select date, name_jp, npatients from infection where name_jp = ? and date like ?", place, date+"%")
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func TheDayBeforeRatioPatientsAll(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}
	prevDate := date.AddDate(0, 0, -1)
	NextDate := date.AddDate(0, 0, 1)

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? or date = ? or date = ?", prevDate, date, NextDate)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)
}

func TheDayBeforeRatioPatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}
	prevDate := date.AddDate(0, 0, -1)
	NextDate := date.AddDate(0, 0, 1)

	place := c.Param("place")

	var infection1 infection
	var infection2 infection
	var infection3 infection
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, NextDate).Scan(&infection1.Date, &infection1.NameJp, &infection1.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, date).Scan(&infection2.Date, &infection2.NameJp, &infection2.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	go func() {
		defer wg.Done()
		err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, prevDate).Scan(&infection3.Date, &infection3.NameJp, &infection3.Npatients)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
	}()
	wg.Wait()

	// 取得した感染者数を配列に格納する
	infections := []infection{infection1, infection2, infection3}

	c.JSON(http.StatusOK, infections)
}

func CountOfPatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	date := c.Param("date")

	var sum int
	err = db.QueryRow("select sum(npatients) from infection where date = ?", date).Scan(&sum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 結果をJSONで出力
	c.JSON(http.StatusOK, gin.H{
		"date":      date,
		"npatients": sum,
	})
}

func Diff(c *gin.Context) {
	// データベースへの接続
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	place := c.Param("place")
	date1 := c.Param("date1")
	date2 := c.Param("date2")

	// SELECT文を実行
	rows, err := db.Query("SELECT (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) - (SELECT npatients FROM infection WHERE date = ? AND name_jp = ?) as npatients", date2, place, date1, place)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// 取得した値を表示
	var diff int
	for rows.Next() {
		err := rows.Scan(&diff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"begin":     date1,
			"end":       date2,
			"place":     place,
			"npatients": diff,
		})

	}
}

func Create(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var json Event_JSON
	validate := Validate() //インスタンス生成

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// yyyymmdd形式の文字列をtime.Time型に変換
	layout := "20060102"
	t, err := time.Parse(layout, json.Begin)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// yyyymmdd形式の文字列をtime.Time型に変換
	t2, err := time.Parse(layout, json.End)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err = validate.Struct(&json); err != nil { //バリデーションを実行し、NGの場合、ここでエラーが返る。
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	insert, err := db.Prepare("INSERT INTO events (title, description, begin, end) VALUES (?, ?, ?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer insert.Close()

	// パラメータを設定してクエリを実行
	_, err = insert.Exec(json.Title, json.Description, t.Format("2006-01-02"), t2.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func Show(c *gin.Context) {
	// データベースに接続
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	// パラメーターからIDを取得
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	// イベントを取得
	var json Event_JSON
	err = db.QueryRow("SELECT title, description, begin, end FROM events WHERE id = ?", id).Scan(&json.Title, &json.Description, &json.Begin, &json.End)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"}) // 404
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// イベントをJSONで出力
	c.JSON(http.StatusOK, json)
}

func ShowAll(c *gin.Context) {
	// データベースに接続
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	// イベントを取得
	rows, err := db.Query("SELECT title, description, begin, end FROM events")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer rows.Close()

	var result []Event_JSON
	for rows.Next() {
		var json Event_JSON
		if err := rows.Scan(&json.Title, &json.Description, &json.Begin, &json.End); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
		result = append(result, json)
	}

	c.JSON(http.StatusOK, result) // 200
}

func Update(c *gin.Context) {
	// データベースに接続
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	// JSONをバインド
	var json Event_JSON
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"}) // 400
		return
	}

	// パラメーターからIDを取得
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	// イベントを更新
	update, err := db.Prepare("UPDATE events SET title = ?, description = ?, begin = ?, end = ? WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	if _, err := update.Exec(json.Title, json.Description, json.Begin, json.End, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}

	c.Status(http.StatusOK) // 200
}

func Delete(c *gin.Context) {
	// データベースに接続
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	// パラメーターからIDを取得
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}) // 400
		return
	}

	// イベントを削除
	delete, err := db.Prepare("DELETE FROM events WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	if _, err := delete.Exec(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}

	c.Status(http.StatusOK) // 200
}

func AverageNpatientsOver(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients < (select avg(npatients) from infection)", date)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func AverageNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients > (select avg(npatients) from infection)", date)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func SetNpatientsUnderDesc(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	count := c.Param("count")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients < ? order by npatients DESC", date, count)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func SetNpatientsUnderAsc(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	count := c.Param("count")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients < ? order by npatients ASC", date, count)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func SetNpatientsDesc(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	count := c.Param("count")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients > ? order by npatients DESC", date, count)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func SetNpatientsAsc(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	count := c.Param("count")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? and npatients > ? order by npatients ASC", date, count)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetNpatientsWithPlaceDesc(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? order by npatients DESC", date)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetNpatientsWithPlaceAsc(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ? order by npatients ASC", date)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetBetWeenDateNpatientsWithPlace(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	place := c.Param("place")
	date1 := c.Param("date1")
	date2 := c.Param("date2")

	rows, err := db.Query("select date, name_jp, npatients from infection where name_jp = ? and date between ? and ?;", place, date1, date2)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetBetweenDateNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date1 := c.Param("date1")
	date2 := c.Param("date2")

	rows, err := db.Query("select date, name_jp, npatients from infection where date between ? and ?", date1, date2)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)

}

func GetInfectionByDateAndPlace(c *gin.Context) {
	// データベースに接続
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer db.Close()

	// パラメーターから日付と場所を取得
	date := c.Param("date")
	place := c.Param("place")

	// インフェクションを取得
	rows, err := db.Query("SELECT date, name_jp, npatients FROM infection WHERE date = ? AND name_jp = ?", date, place)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
		return
	}
	defer rows.Close()

	var result []infection
	for rows.Next() {
		var inf infection
		if err := rows.Scan(&inf.Date, &inf.NameJp, &inf.Npatients); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // 500
			return
		}
		result = append(result, inf)
	}

	c.JSON(http.StatusOK, result) // 200
}

func GetInfectionByDate(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")

	rows, err := db.Query("select date, name_jp, npatients from infection where date = ?", date)
	if err != nil {
		log.Fatal(err)
	}
	var resultInfection []infection

	for rows.Next() {
		infection := infection{}
		if err := rows.Scan(&infection.Date, &infection.NameJp, &infection.Npatients); err != nil {
			log.Fatal(err)
		}
		resultInfection = append(resultInfection, infection)
	}

	c.JSON(http.StatusOK, resultInfection)
}

func GetDateNpatients(c *gin.Context) {
	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	date := c.Param("date")
	place := c.Param("place") // place用の別テーブルを作成して、そこのidを選択できないか。プルダウンで選択したい。

	var infection infection

	err = db.QueryRow("SELECT date, name_jp, npatients FROM infection WHERE name_jp = ? and date = ?", place, date).Scan(&infection.Date, &infection.NameJp, &infection.Npatients)

	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, infection)

}

func Validate() *validator.Validate {
	validate := validator.New()
	return validate
}

func Import(c *gin.Context) { // データ取得、データベースに保存
	log.Print("データ取り込み中")
	url := "https://opendata.corona.go.jp/api/Covid19JapanAll"
	resp, _ := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	byteArray, _ := ioutil.ReadAll(resp.Body)

	jsonBytes := ([]byte)(byteArray)
	data := new(Npatients)

	if err := json.Unmarshal(jsonBytes, data); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
		return
	}
	// fmt.Println(data.ItemList)

	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	delete, err := db.Prepare("DELETE FROM infection")
	if err != nil {
		log.Fatal(err)
	}
	delete.Exec()

	for _, v := range data.ItemList {
		insert, err := db.Prepare("INSERT INTO infection(date, name_jp, npatients) values (?,?,?)")
		if err != nil {
			log.Fatal(err)
		}
		insert.Exec(v.Date, v.NameJp, v.Npatients)
	}

	log.Print("データ取り込み完了")
}

func ImportDeceased(c *gin.Context) { // データ取得、データベースに保存
	log.Print("データ取り込み中")
	url := "https://opendata.corona.go.jp/api/OccurrenceStatusOverseas"
	resp, _ := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	byteArray, _ := ioutil.ReadAll(resp.Body)

	jsonBytes := ([]byte)(byteArray)
	data := new(Deceased)

	if err := json.Unmarshal(jsonBytes, data); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
		return
	}
	// fmt.Println(data.ItemList)

	db, err := sql.Open("mysql", "root:password@(localhost:3306)/local?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	delete, err := db.Prepare("DELETE FROM decease")
	if err != nil {
		log.Fatal(err)
	}
	delete.Exec()

	for _, v := range data.ItemList {
		insert, err := db.Prepare("INSERT INTO decease(date, data_name, infected_num, deceased_num) values (?,?,?,?)")
		if err != nil {
			log.Fatal(err)
		}
		insert.Exec(v.Date, v.DataName, v.InfectedNum, v.DeceasedNum)
	}

	log.Print("データ取り込み完了")
}
