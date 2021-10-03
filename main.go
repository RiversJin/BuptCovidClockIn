package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// FileName 用户配置文件
	FileName = "user.json"
	// IndexUrl 初始页
	IndexUrl = "https://app.bupt.edu.cn/ncov/wap/default/index"
	// LoginUrl 登录URL 通过向其发送post请求登录,获得cookie
	LoginUrl = "https://app.bupt.edu.cn/uc/wap/login/check"
	// SaveUrl 打卡 url
	SaveUrl   = "https://app.bupt.edu.cn/ncov/wap/default/save"
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36 Edg/92.0.902.67"
)

type User struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	VaccineCondition string `json:"vaccine_condition"`
}

type ResponseJson struct {
	E int    `json:"e"`
	M string `json:"m"`
	D struct {
	} `json:"d"`
}

const (
	PositionPayload = " {\"type\":\"complete\",\"position\":{\"Q\":%.12f,\"R\":%.12f,\"lng\":%.5f,\"lat\":%.5f},\"location_type\":\"html5\",\"message\":\"Get geolocation success.Convert Success.Get address success.\",\"accuracy\":55,\"isConverted\":true,\"status\":1,\"addressComponent\":{\"citycode\":\"010\",\"adcode\":\"110108\",\"businessAreas\":[{\"name\":\"北下关\",\"id\":\"110108\",\"location\":{\"Q\":39.955976,\"R\":116.33873,\"lng\":116.33873,\"lat\":39.955976}},{\"name\":\"西直门\",\"id\":\"110102\",\"location\":{\"Q\":39.942856,\"R\":116.34666099999998,\"lng\":116.346661,\"lat\":39.942856}},{\"name\":\"小西天\",\"id\":\"110108\",\"location\":{\"Q\":39.957147,\"R\":116.364058,\"lng\":116.364058,\"lat\":39.957147}}],\"neighborhoodType\":\"科教文化服务;学校;高等院校\",\"neighborhood\":\"北京邮电大学\",\"building\":\"\",\"buildingType\":\"\",\"street\":\"西土城路\",\"streetNumber\":\"31号院\",\"country\":\"中国\",\"province\":\"北京市\",\"city\":\"\",\"district\":\"海淀区\",\"township\":\"北太平庄街道\"},\"formattedAddress\":\"北京市海淀区北太平庄街道北京邮电大学北京邮电大学海淀校区\",\"roads\":[],\"crosses\":[],\"pois\":[],\"info\":\"SUCCESS\"}"
	Q               = 39.962380642362
	R               = 116.35539930555603
	PostPayload1    = "ismoved=0&jhfjrq=&jhfjjtgj=&jhfjhbcc=&sfxk=0&xkqq=&szgj=&szcs=&zgfxdq=0&mjry=0&csmjry=0&ymjzxgqk="
	PostPayload2    = "&xwxgymjzqk=3&tw=2&sfcxtz=0&sfjcbh=0&sfcxzysx=0&qksm=&sfyyjc=0&jcjgqr=0&remark=&address=%E5%8C%97%E4%BA%AC%E5%B8%82%E6%B5%B7%E6%B7%80%E5%8C%BA%E5%8C%97%E5%A4%AA%E5%B9%B3%E5%BA%84%E8%A1%97%E9%81%93%E5%8C%97%E4%BA%AC%E9%82%AE%E7%94%B5%E5%A4%A7%E5%AD%A6%E5%8C%97%E4%BA%AC%E9%82%AE%E7%94%B5%E5%A4%A7%E5%AD%A6%E6%B5%B7%E6%B7%80%E6%A0%A1%E5%8C%BA&geo_api_info="
	PostPayload3    = "&area=%E5%8C%97%E4%BA%AC%E5%B8%82++%E6%B5%B7%E6%B7%80%E5%8C%BA&province=%E5%8C%97%E4%BA%AC%E5%B8%82&city=%E5%8C%97%E4%BA%AC%E5%B8%82&sfzx=1&sfjcwhry=0&sfjchbry=0&sfcyglq=0&gllx=&glksrq=&jcbhlx=&jcbhrq=&bztcyy=&sftjhb=0&sftjwh=0&sfsfbh=0&xjzd=&jcwhryfs=&jchbryfs=&szsqsfybl=0&sfygtjzzfj=0&gtjzzfjsj=&sfjzxgym=1&sfjzdezxgym=1&jcjg=&created_uid=0&date=20211002&uid=25439&created=1633164831&id=14034594&gwszdd=&sfyqjzgc=&jcqzrq=&sfjcqz=&jrsfqzys=&jrsfqzfy=&sfsqhzjkk=&sqhzjkkys="
)

var headers4Index = map[string]string{
	"UserAgent":       UserAgent,
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
	"Accept-Encoding": "gzip, deflate, br",
	"Cache-Control":   "no-cache",
	"Connection":      "keep-alive",
	"DNT":             "1",
}
var headers4Login = map[string]string{
	"UserAgent":    UserAgent,
	"Accept":       "application/json, text/javascript, */*; q=0.01",
	"DNT":          "1",
	"Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
}
var headers4Post = map[string]string{
	"UserAgent":        UserAgent,
	"Accept":           "application/json, text/javascript, */*; q=0.01",
	"DNT":              "1",
	"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
	"Cache-Control":    "no-cache",
	"Accept-Encoding":  "gzip, deflate, br",
	"Accept-Language":  "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
	"X-Requested-With": "XMLHttpRequest",
}

func readUsersFromFile(filename string) ([]User, error) {
	users := make([]User, 0)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
func RandomPosition(coordinate float64) float64 {
	return coordinate + randFloat(-0.001, 0.001)
}
func login(user *User) ([]*http.Cookie, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", IndexUrl, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers4Index {
		req.Header.Set(key, value)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	Cookies := response.Cookies()

	data := url.Values{
		"username": []string{user.Username},
		"password": []string{user.Password},
	}
	req, err = http.NewRequest("GET", LoginUrl, strings.NewReader(data.Encode()))
	for _, cookie := range Cookies {
		req.AddCookie(cookie)
	}
	for key, value := range headers4Login {
		req.Header.Set(key, value)
	}
	response, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	reqBody := new(ResponseJson)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		return nil, err
	}
	if reqBody.E != 0 {
		return nil, errors.New(reqBody.M)
	}
	Cookies = append(Cookies, response.Cookies()...)
	return Cookies, nil
}
func makePositionPayload() string {
	q := RandomPosition(Q)
	r := RandomPosition(R)
	return url.QueryEscape(fmt.Sprintf(PositionPayload, q, r, q, r))
}
func makePostPayload(user *User) string {
	vaccineConditionPayload := url.QueryEscape(user.VaccineCondition)
	return PostPayload1 + vaccineConditionPayload + PostPayload2 + makePositionPayload() + PostPayload3
}
func postPayload(cookies []*http.Cookie, payload string) (*ResponseJson, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", SaveUrl, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for key, value := range headers4Post {
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	reqBody := new(ResponseJson)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		return nil, err
	}
	return reqBody, nil
}
func post(user *User, group *sync.WaitGroup) {
	defer group.Done()
	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	cookie, err := login(user)
	if err != nil {
		log.Printf("An error \"%v\"was encountered while login %+v", err, user)
		return
	}
	payload := makePostPayload(user)
	reqBody, err := postPayload(cookie, payload)
	if err != nil {
		log.Printf("An error \"%v\"was encountered while post %v", err, user.Username)
		return
	}
	if reqBody.E != 0 {
		log.Printf("Error: \"%v\" while post %v", reqBody.M, user.Username)
		return
	}
	log.Printf("Successfully post user:%v detail:%v", user.Username, reqBody.M)
}

func main() {
	waitGroup := sync.WaitGroup{}
	rand.Seed(time.Now().Unix())
	users, err := readUsersFromFile(FileName)
	if err != nil {
		panic(err)
	}
	waitGroup.Add(len(users))
	for _, user := range users {
		go post(&user, &waitGroup)
	}
	waitGroup.Wait()
}
