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
	"github.com/andybalholm/brotli"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unsafe"
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	clientConnections = make(map[string]http.Client)
)

var ctx = context.Background()

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
	Source string
	Headers string
	ResponseUrl string
	ResponseCode string
	Bodylen int
	Totallen int
}

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
		panic(err)
	}

	parsedUrl, err := url.Parse(p.Url)
	if err != nil {
		return nil
	}
	host := parsedUrl.Scheme + "://" + parsedUrl.Host
	log.Print("successfully parsed object")

	var client http.Client

	if p.Proxy != ""{
		if val, ok := clientConnections[p.Proxy]; ok {
			return MakeReq(p, val)
		}
	}else if val, ok := clientConnections[host]; ok {
		return MakeReq(p, val)
	}

	if p.Proxy != ""{
		client, err = proxyClient(p.Hello, p.Proxy)
		if err != nil {
			return nil
		}
		clientConnections[p.Proxy] = client
		return MakeReq(p, client)
	}else{
		parsedUrl, err := url.Parse(p.Url)
		if err != nil {
			return nil
		}
		host := parsedUrl.Scheme + "://" + parsedUrl.Host

		client, err = httpClient(p.Hello, host)
		if err != nil {
			return nil
		}
		clientConnections[host] = client
		return MakeReq(p, client)
	}
	log.Print("something went wrong in client creation")
	return nil
}


func MakeReq(jsonIn cjsonIn, client http.Client) *C.char {
	p := jsonIn
	var reqOut *http.Request

	if p.Method == "POST" {
		var reqBody []byte
		bodyOut := strings.Replace(p.Body, "'","\"", -1)
		bodyIn := []byte(strings.Replace(bodyOut, "'","\"", -1))
		reqBody = bodyIn
		reqOut, _ = http.NewRequest( "POST", p.Url, bytes.NewBuffer(reqBody))
	}else{
		reqOut, _ = http.NewRequest(p.Method, p.Url, nil)
	}
	log.Print("request instantiated")

	var headerString string
	for _, v := range p.Headers {
		for key, val := range v{
			headerString += `{` + key + `:` + val + `}`
		}
	}
	headerStringLength := len([]rune(headerString))
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
		log.Print(headerKey, " ", val)
	}
	log.Print("headers set")

	if p.H2 == "true"{
		reqOut.Proto = "HTTP/2.0"
		reqOut.ProtoMajor = 2
		reqOut.ProtoMinor = 0
	}else{
		reqOut.Proto = "HTTP/1.1"
		reqOut.ProtoMajor = 1
		reqOut.ProtoMinor = 1
	}
	log.Print("protocol set\nmaking request")

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
	log.Print("body read")

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
	//resStringBack, _ := json.Marshal(&res)
	//return &resStringBack
	data, _ := json.Marshal(res)

	str := string(data)
	log.Print(str)

	// return string(data)
	return  C.CString(string(data))
}

//export Collect
func Collect(Output *C.char) {
	C.free(unsafe.Pointer(Output))
}

//export DeleteClient
func DeleteClient(clientKey string){
	defer func() {
		if err := recover(); err != nil {
		}
	}()

	delete(clientConnections, clientKey)
}

func main(){
	//input := `{"url":"http://heade.rs",
//"method":"GET",
//"body":"",
//"h2":"false",
//"hello":"chrome",
//"proxy":"",
//"headers":[{"user-agent":"mozilla"},{"hello":"world"}]}`
	//var a = Request(C.CString(input))
	//log.Print(a)
}
