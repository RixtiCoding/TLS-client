package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unsafe"


	"github.com/andybalholm/brotli"
)

// client handler is a sync.map to sync.pool

var (
	ctx = context.Background()
	ClientHandler = sync.Map{}
	requestsStores = make(map[string]string)
	serverHost = "http://127.0.0.1"
	pxHost = serverHost + ":7000"
	cfHost = serverHost + ":7001"
	ticketHost = serverHost + ":7002"
	incapsulaHost = serverHost + ":7003"
	shapeHost = serverHost + ":7004"
	akamaiHost = serverHost + ":7005"
	)

type cjsonIn struct{
	Url string `json:"url"`
	Method string `json:"method"`
	Body string `json:"body"`
	H2 string `json:"h2"`
	Hello string `json:"hello"`
	Proxy string `json:"proxy"`
	Headers []map[string]string `json:"headers"`
}
type cjsonOut struct{
	Source string `json:"Source"`
	Headers string `json:"Headers"`
	ResponseUrl string `json:"ResponseUrl"`
	Cookies []*http.Cookie `json:"Cookies"`
	ResponseCode string `json:"ResponseCode"`
	Bodylen int `json:"Bodylen"`
	Totallen int `json:"Totallen"`
}

type genjson struct{
	Proxy string `json:"proxy"`
	Url string `json:"url"`
}

// bodyFormatMap takes a map[string]string and returns
// a body formatted for post/patch request
func bodyFormatMap(mapin map[string]string) string{
	strout := ""
	for k, v := range mapin{
		strout += url.QueryEscape(k) +"=" + url.QueryEscape(v) + "&"
	}
	strout = strout[:len(strout)-1]
	return strout
}
func COOKIESTRINGFORMATTER(carrin []*http.Cookie) string{
	outstr := ""
	for _, cookie := range carrin{
		outstr += cookie.Name + "=" + cookie.Value +";"
	}
	return outstr
}
func jsonOutParse(rawstring *C.char) cjsonOut{
	outjson := &cjsonOut{}
	json.Unmarshal([]byte(C.GoString(rawstring)), outjson)
	return *outjson
}

// Request base method for making requests. this method
// checks if a free client exists, pushes and pops
// client handler accordingly. two nested mutex
//export Request
func Request(rawJsonIn *C.char) *C.char{
	defer func() {
		if err := recover(); err != nil {
		}
	}()

	p := cjsonIn{}
	message :=  C.GoString(rawJsonIn)

	err := json.Unmarshal([]byte(message), &p)
	if err != nil {
		return C.CString("")
	}

	var connType string
	var storeTag string
	if p.Proxy != ""{
		storeTag = p.Proxy
		connType = "proxy"
	}else {
		parsedUrl, err := url.Parse(p.Url)
		if err != nil {
			return nil
		}
		host := parsedUrl.Scheme + "://" + parsedUrl.Host
		connType = "raw"
		storeTag = host
	}

	if _, ok := ClientHandler.Load(storeTag);!ok{
		ClientHandler.Store(storeTag, &sync.Pool{})
	}

	if val, ok := ClientHandler.Load(storeTag); ok {
		var clientused *http.Client
		pool := val.(*sync.Pool)
		client := pool.Get()
		if client != nil{
			clientused = client.(*http.Client)
		}else{
			switch connType{
			case "raw":
				clientused, err = httpClient(p.Hello, storeTag)
				if err != nil {
					return nil
				}
			case "proxy":
				clientused, err = proxyClient(p.Hello, p.Proxy)
				if err != nil {
					return nil
				}
			}
		}

		rb := MakeReq(p, clientused)
		pool.Put(client)
		return rb
	}

	return nil
}
func MakeReq(jsonIn cjsonIn, client *http.Client) *C.char {
	p := jsonIn
	var reqOut *http.Request

	if p.Method == "POST" || p.Method == "PATCH"{
		var reqBody []byte
		bodyOut := strings.Replace(p.Body, "'","\"", -1)
		bodyIn := []byte(strings.Replace(bodyOut, "'","\"", -1))
		reqBody = bodyIn
		reqOut, _ = http.NewRequest(p.Method, p.Url, bytes.NewBuffer(reqBody))
	}else{
		reqOut, _ = http.NewRequest(p.Method, p.Url, nil)
	}

	var headerString string
	for _, v := range p.Headers {
		for key, val := range v{
			headerString += `{` + key + `:` + val + `}`
		}
	}

	headerStringLength := len([]rune(headerString))
	if headerStringLength != 0 {
		headerlist := strings.Split(headerString[1:headerStringLength-1], "}{")
		for _, item := range headerlist{
			pair := strings.SplitN(item, ":", 2)
			keyArr := strings.Split(pair[0], "-")
			var sanitizedKey []string
			for _, val := range keyArr{
				sanitizedKey = append(sanitizedKey,strings.Title(strings.ToLower(val)))
			}
			headerKey := strings.Join(sanitizedKey, "-")

			val := strings.Replace(pair[1], "\\\"","\"", -1)

			reqOut.Header.Set(headerKey, val)
			reqOut.Header["orderedHeaderList"] = append(reqOut.Header["orderedHeaderList"], headerKey)
		}
	}

	if p.H2 == "true"{
		reqOut.Proto = "HTTP/2.0"
		reqOut.ProtoMajor = 2
		reqOut.ProtoMinor = 0
	}else{
		reqOut.Proto = "HTTP/1.1"
		reqOut.ProtoMajor = 1
		reqOut.ProtoMinor = 1
	}

	resp, err := client.Do(reqOut)

	if err != nil {
		return nil
	}

	res := new(cjsonOut)

	var reader io.ReadCloser

	headerstring, _ := json.Marshal(resp.Header)

	switch resp.Header.Get("Content-Encoding"){
	case "gzip":
		reader, _ = gzip.NewReader(resp.Body)

	case "br":
		reader = ioutil.NopCloser(brotli.NewReader(resp.Body))

	default:
		reader = resp.Body
	}

	bodyBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil
	}

	res.Source = string(bodyBytes)
	res.Headers = string(headerstring)
	location, err := resp.Location()
	if err == nil{
		res.ResponseUrl = location.String()
	}else{
		res.ResponseUrl = p.Url
	}
	res.ResponseCode = strconv.Itoa(resp.StatusCode)
	res.Bodylen = len(bodyBytes)
	res.Totallen = res.Bodylen + len(res.Headers)
	res.Cookies = resp.Cookies()
	data, _ := json.Marshal(res)

	return  C.CString(string(data))
}


// Collect collects value at address and destructs memory
//export Collect
func Collect(Output *C.char) {
	C.free(unsafe.Pointer(Output))
}

func main() {
	//cookie, uuid := pxCookie(C.CString("hibbett"), C.CString(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36`), C.CString(`"Google Chrome";v="89", "Chromium";v="89", ";Not A Brand";v="99"`))
	//log.Print(C.GoString(cookie))
	//log.Print(C.GoString(uuid))
	//
	//cookie = pxEvent(uuid)
	//log.Print(C.GoString(cookie))


	input := `{"url":"https://www.grosbasket.com",
"method":"GET",
"headers":[{"Accept":"application/json"},{"Connection":"Keep-Alive"},{"Accept-Encoding":"gzip, deflate, br"},{"Accept-Language":"en-US,en;q=0.9"},{"Sec-Fetch-Site":"none"},{"Sec-Fetch-Mode":"navigate"},{"Sec-Fetch-User":"?1"},{"Sec-Fetch-Dest":"document"},{"x-api-lang":"en-US"},{"x-fl-app-version":"4.5.0"},{"x-api-country":"US"},{"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36"},{"x-api-key":"Rk5dxB9rfK3bIS79l671rIkNAjLJvDVD"},{"x-flapi-api-identifier":"9fe7ce820c884260a3dd6a0a7691d125"},{"x-flapi-session-id":"b0a228f0164f47e3a7a21c9e616ff9ad"},{"x-fl-device-id":"ea13b4fd-f6e5-4175-9ab8-08eff00633a1"}],
"body":"",
"h2":"false",
"hello":"chrome",
"proxy":""}`
	a := Request(C.CString(input))
	log.Print("responsebody ",  C.GoString(a))

	//we establish a connection via first request ^^ above. can be just a HEAD request.
	//our client seed is the proxy: field. if not used, it is the url: field.

	hoststring := C.CString("https://www.grosbasket.com/")
	chuastring := C.CString(`"Google Chrome";v="89", "Chromium";v="89", ";Not A Brand";v="99"`)
	uastring := C.CString(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36`)
	a = cfRequestChain(hoststring, chuastring, uastring, C.CString(""))
	log.Print(C.GoString(a))

}