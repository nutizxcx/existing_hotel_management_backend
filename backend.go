package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type RegisterData struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ConfPw    string `json:"confPw"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	BirthDay  string `json:"birthDay"`
	Tel       string `json:"tel"`
}

type LoginData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResponseStatus struct {
	Res string `json:"res"`
}

type HotelListReqData struct {
	Province     string `json:"province"`
	District     string `json:"district"`
	CheckinDate  string `json:"checkin_date"`
	CheckoutDate string `json:"checkout_date"`
}

type ProvinceAndDistrictData struct {
	PvAndDt map[string][]string `json:"pv_and_dt"`
}

type BasicHotelData struct {
	HotelID       string `json:"hotelID"`
	HotelName     string `json:"hotelName"`
	PictureURL    string `json:"picURL"`
	PricePerNight string `json:"pricePerNight"`
}

type HotelInfoReq struct {
	HotelID string `json:"hotelID"`
}

type HotelDetailData struct {
	HotelID        string   `json:"hotelID"`
	HotelName      string   `json:"hotelName"`
	Address        string   `json:"address"`
	Tel            string   `json:"tel"`
	Province       string   `json:"province"`
	District       string   `json:"district"`
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	Description    string   `json:"description"`
	PricePerNight  int      `json:"pricePerNight"`
	MaxGuest       int      `json:"maxGuest"`
	BedroomAmount  int      `json:"bedroomAmount"`
	BedAmount      int      `json:"bedAmount"`
	BathroomAmount int      `json:"bathroomAmount"`
	HotelAllPic    []string `json:"hotelPicture"`
	HotelFac       []string `json:"hotelFac"`
}

type SearchReq struct {
	SearchString string `json:"searchString"`
}

type SearchContent struct {
	HotelID       int    `json:"hotelID"`
	HotelName     string `json:"hotelName"`
	Province      string `json:"province"`
	District      string `json:"district"`
	PricePerNight int    `json:"pricePerNight"`
}

type BookingReqData struct {
	Email              string `json:"email"`
	BookingDateAndTime string `json:"bookingDateAndTime"`
	DateIn             string `json:"checkinDate"`
	DateOut            string `json:"checkoutDate"`
	GuestAmount        int    `json:"guestAmount"`
	HotelID            int    `json:"hotelID"`
	TotalPrice         int    `json:"totalPrice"`
}

type UserCheckBooking struct {
	Email string `json:"email"`
}

type BookingHistory struct {
	CheckinDate  string `json:"checkinDate"`
	CheckoutDate string `json:"checkoutDate"`
	GuestAmount  int    `json:"guestAmount"`
	HotelName    string `json:"hotelName"`
	TotalPrice   int    `json:"totalPrice"`
}

var db *sql.DB
var err error

func main() {
	e := echo.New()
	e.Use(middleware.CORS())

	//connect database
	db, err = sql.Open("mysql", "root:@/the_existing_hostel_management")
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Successfully Connected MySQL database")

	e.POST("/register", register)
	e.POST("/bookingData", bookingData)
	e.POST("/login", login)
	e.POST("/hotelList", hotelList)
	e.GET("/provinceAndDistrict", pvAndDt)
	e.POST("/hotelInfo", hotelInfo)
	e.POST("/search", searchContent)
	e.POST("/bookingHistory", bookingHistory)
	e.Start(":8080")
	defer db.Close()
}

func bookingHistory(c echo.Context) error {
	fmt.Println("this is booking history service")

	userBookingHist := make([]BookingHistory, 0)
	userEmail := UserCheckBooking{}

	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &userEmail)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	rows, _ := db.Query(`SELECT bh.checkin_date, bh.checkout_date, bh.guest_amount, hd.hotel_name, bh.total_payment
	FROM booking_history bh
	INNER JOIN hotel_detail hd
	ON hd.hotel_id = bh.hotel_id
	WHERE bh.email = ?`, userEmail.Email)

	for rows.Next() {
		var dateIn, dateOut, hotelName string
		var guestAmount, totalPrice int
		_ = rows.Scan(&dateIn, &dateOut, &guestAmount, &hotelName, &totalPrice)
		userBookingHist = append(userBookingHist, BookingHistory{CheckinDate: dateIn, CheckoutDate: dateOut, GuestAmount: guestAmount, HotelName: hotelName, TotalPrice: totalPrice})
	}

	return c.JSON(http.StatusOK, userBookingHist)

}

func bookingData(c echo.Context) error {
	fmt.Println("this is booking service")
	bookingReq := BookingReqData{}

	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &bookingReq)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	fmt.Println(bookingReq)

	rows, _ := db.Query(`SELECT hfd.date
	FROM hotel_free_day hfd
	WHERE hfd.hotel_id = ? AND hfd.date >= ? AND hfd.date <= ? 
	AND hfd.available_rooms = 0`, bookingReq.HotelID, bookingReq.DateIn, bookingReq.DateOut)
	var fullDate string
	for rows.Next() {
		err = rows.Scan(&fullDate)
		if err != nil {
			return c.String(http.StatusInternalServerError, "")
		}
	}

	if fullDate == "" {
		stmt, err := db.Prepare("INSERT INTO booking_history VALUES (?,?,?,?,?,?,?)")
		stmt.Exec(bookingReq.Email, bookingReq.BookingDateAndTime, bookingReq.DateIn, bookingReq.DateOut, bookingReq.GuestAmount, bookingReq.HotelID, bookingReq.TotalPrice)
		if err != nil {
			log.Printf("Failed inserting the request body for register: %s", err)
			return c.String(http.StatusInternalServerError, "")
		}
		stmt, err = db.Prepare(`UPDATE hotel_free_day
		SET available_rooms = available_rooms - 1
		WHERE date >= ? AND date <= ? AND hotel_id = ?`)
		stmt.Exec(bookingReq.DateIn, bookingReq.DateOut, bookingReq.HotelID)
		if err != nil {
			log.Printf("Failed inserting the request body for register: %s", err)
			return c.String(http.StatusInternalServerError, "")
		}

		return c.JSON(http.StatusOK, ResponseStatus{Res: "success"})
	} else {
		return c.JSON(http.StatusOK, ResponseStatus{Res: fullDate})
	}

}

func searchContent(c echo.Context) error {
	fmt.Println("this is search content service")
	searchContentData := SearchReq{}
	searchResult := make([]SearchContent, 0)

	//retrieve data
	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &searchContentData)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	rows, _ := db.Query(`SELECT hd.hotel_id, hd.hotel_name, hd.province, hd.district, hd.price_per_night 
						FROM hotel_detail hd
						WHERE hd.hotel_name LIKE ?`, "%"+searchContentData.SearchString+"%")
	var res2, res3, res4 string
	var res1, res5 int
	for rows.Next() {
		err = rows.Scan(&res1, &res2, &res3, &res4, &res5)
		searchResult = append(searchResult, SearchContent{HotelID: res1, HotelName: res2, Province: res3, District: res4, PricePerNight: res5})
	}
	return c.JSON(http.StatusOK, searchResult)

}

func hotelInfo(c echo.Context) error {
	fmt.Println("this is hotel info service")

	hotelID := HotelInfoReq{}
	hotelDetail := HotelDetailData{}

	//retrieve data
	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &hotelID)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//convert hotel id from string to int
	hotelIDInt, _ := strconv.Atoi(hotelID.HotelID)

	//first query: common hotel data
	rows, _ := db.Query("SELECT * FROM hotel_detail WHERE hotel_id = ?", hotelIDInt)
	for rows.Next() {
		var hotelID, hotelName, address, tel, province, district, description string
		var latitude, longitude float64
		var pricePerNight, maxGuest, bedAmount, bedroomAmount, bathroomAmount int
		err = rows.Scan(&hotelID, &hotelName, &address, &tel, &province, &district, &latitude, &longitude,
			&description, &pricePerNight, &maxGuest, &bedroomAmount, &bedAmount, &bathroomAmount)

		hotelDetail = HotelDetailData{HotelID: hotelID, HotelName: hotelName, Address: address, Tel: tel,
			Province: province, District: district, Latitude: latitude, Longitude: longitude, Description: description,
			PricePerNight: pricePerNight, MaxGuest: maxGuest, BedroomAmount: bedroomAmount, BedAmount: bedAmount, BathroomAmount: bathroomAmount}
	}

	//second query: hotel pics
	rows, _ = db.Query("SELECT pic_url FROM hotel_pic WHERE hotel_id = ?", hotelIDInt)
	for rows.Next() {
		var picURL string
		rows.Scan(&picURL)
		hotelDetail.HotelAllPic = append(hotelDetail.HotelAllPic, picURL)
	}

	//third query: hotel facility
	rows, _ = db.Query("SELECT facility FROM hotel_facilities WHERE hotel_id = ?", hotelIDInt)
	for rows.Next() {
		var hotelFac string
		rows.Scan(&hotelFac)
		hotelDetail.HotelFac = append(hotelDetail.HotelFac, hotelFac)
	}
	fmt.Println(hotelDetail)

	return c.JSON(http.StatusOK, hotelDetail)

}

func pvAndDt(c echo.Context) error {
	fmt.Println("this is province and district service")

	//query data
	rows, _ := db.Query(`SELECT DISTINCT hd.province, hd.district FROM hotel_detail hd`)

	//create map of slice
	x := make(map[string][]string)

	for rows.Next() {
		var province string
		var district string
		_ = rows.Scan(&province, &district)
		fmt.Printf("%s %s \n", province, district)
		x[province] = append(x[province], district)
	}

	return c.JSON(http.StatusOK, x)

}

func hotelList(c echo.Context) error {
	fmt.Println("this is hotel list")

	hotelListReqData := HotelListReqData{}
	hotelListResData := make([]BasicHotelData, 0)

	//retrieve request data
	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &hotelListReqData)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//change checkin and checkout date format
	layout := "2006-01-02"
	checkin, _ := time.Parse(layout, hotelListReqData.CheckinDate)
	checkout, _ := time.Parse(layout, hotelListReqData.CheckoutDate)

	//query data
	rows, err := db.Query(`SELECT hd.hotel_id, hd.hotel_name, hd.price_per_night, hc.pic_url 
	FROM hotel_detail hd 
	INNER JOIN hotel_pic hc 
	ON hd.hotel_id = hc.hotel_id 
	WHERE hd.province = ? AND hd.district = ? AND hd.hotel_id != ALL 
	(SELECT DISTINCT hfd.hotel_id
	 FROM hotel_free_day hfd
	 WHERE hfd.date >= ? AND hfd.date <= ? AND  hfd.available_rooms = 0
	) 
	GROUP BY hd.hotel_id`, hotelListReqData.Province, hotelListReqData.District, checkin, checkout)
	var hotel_id string
	var hotel_name string
	var price_per_night string
	var pic_url string
	for rows.Next() {
		err = rows.Scan(&hotel_id, &hotel_name, &price_per_night, &pic_url)
		fmt.Printf("%s %s %s %s\n", hotel_id, hotel_name, price_per_night, pic_url)
		hotelListResData = append(hotelListResData, BasicHotelData{HotelID: hotel_id, HotelName: hotel_name, PictureURL: pic_url, PricePerNight: price_per_night})
	}

	//return data
	return c.JSON(http.StatusOK, hotelListResData)

}

func login(c echo.Context) error {
	fmt.Println("this is login")
	//create login interface
	loginData := LoginData{}

	//retrieve request data
	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &loginData)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	log.Printf("regis data: %#v", loginData)

	// query data
	rows, err := db.Query("SELECT email FROM user_account WHERE email = ? AND password = ?", loginData.Email, loginData.Password)
	var email string
	for rows.Next() {
		err = rows.Scan(&email)
		fmt.Printf("email: %s\n", email)
	}

	if email == "" {
		res := ResponseStatus{
			Res: "invalid email or password",
		}
		return c.JSON(http.StatusOK, res)
	} else {
		res := ResponseStatus{
			Res: "login success",
		}
		return c.JSON(http.StatusOK, res)
	}

}

func register(c echo.Context) error {

	fmt.Println("this is register")
	//crate register interface
	regisData := RegisterData{}

	//retrieve request data
	x, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("Failed reading the request body for register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//JSON decode
	err = json.Unmarshal(x, &regisData)
	if err != nil {
		log.Printf("Failed unmarshalling in register: %s", err)
		return c.String(http.StatusInternalServerError, "")
	}

	//change birthDay date format
	layout := "2006-01-02"
	birthDay, _ := time.Parse(layout, regisData.BirthDay)

	log.Printf("regis data: %#v", regisData)

	// query data
	rows, err := db.Query("SELECT email FROM user_account WHERE email = ?", regisData.Email)
	var email string
	for rows.Next() {
		err = rows.Scan(&email)
		fmt.Printf("email: %s\n", email)
	}

	if email == "" {
		// insert data to database
		stmt, err := db.Prepare("INSERT INTO user_account VALUES (?,?,?,?,?,?)")
		stmt.Exec(regisData.Email, regisData.FirstName, regisData.LastName, birthDay, regisData.Password, regisData.Tel)
		if err != nil {
			log.Printf("Failed inserting the request body for register: %s", err)
			return c.String(http.StatusInternalServerError, "")
		}
	} else {
		res := &ResponseStatus{
			Res: "duplicate email",
		}
		return c.JSON(http.StatusOK, res)
	}

	//create response JSON value
	res := &ResponseStatus{
		Res: "insert success",
	}
	defer c.Request().Body.Close()
	return c.JSON(http.StatusOK, res)

}
