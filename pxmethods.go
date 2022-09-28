package main
import (
	"C"
	"bytes"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type UUIDSTORE struct{
	SID		 string
	VID		 string
	CS		 string
	PXHD	 string
	SEQ		 string
	UA		 string
	SEC 	 string
	SITE	 string
	TAG	 	 string
	VERSION  string
	COOKIES  string
	ENDPOINT string
	SITEKEY  string
}

var (
	globalUhandler = sync.Map{}

	uuidreadmap = make(map[string]UUIDSTORE)
	)

func PX2RESHANDLER(rawres string) map[string]string {
	resjson := struct {
		Do []string `json:"do"`
	}{}
	json.Unmarshal([]byte(rawres), &resjson)
	objOut := make(map[string]string)
	for _, v := range resjson.Do{
		splitstr := strings.SplitN(v, "|", 2)
		objOut[splitstr[0]] = splitstr[1]
	}
	return objOut
}
func PX3COOKIEHANDLER(rawstring string) string{
	return strings.Split(rawstring, "|")[2]
}

//export pxCookie
func pxCookie(siteP, userAgentP, secP *C.char) (*C.char, *C.char){
	site := C.GoString(siteP)
	ua := C.GoString(userAgentP)
	sec := C.GoString(secP)
	var version, tag, collectorendpoint, sitekey string
	if ua == ""{
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36"
	}
	if sec == ""{
		sec = `"Google Chrome";v="89", "Chromium";v="89", ";Not A Brand";v="99"`
	}
	switch(site) {
	case "hibbett":
		version = "v6.5.0"
		tag = "200"
		collectorendpoint = "https://collector-pxajdckzhd.px-cloud.net/api/v2/collector"
		sitekey = "pxajdckzhd"
	case "walmart":
		version = "v6.2.6"
		tag = "188"
		collectorendpoint = "https://collector-pxu6b0qd2s.px-cloud.net/api/v2/collector"
		sitekey = "pxu6b0qd2s"
	case "ssense":
		version = "v6.5.0"
		tag = "200"
		collectorendpoint = "https://www.ssense.com/58Asv359/xhr/api/v2/collector"
		sitekey = "58Asv359"
	}

	client, err := httpClient("chrome89", collectorendpoint)
	if err != nil {
		return nil, nil
	}

	rawbody := map[string]interface{}{
		"site": site,
		"pxtype": "PX2",
		"version": version,
		"tag": tag,
		"ua": ua,
		"count": "0",
	}
	rawbodybytes, _ := json.Marshal(rawbody)
	resp, _ := http.Post(pxHost, "application/json", bytes.NewBuffer(rawbodybytes))
	sessionObj := make(map[string]string)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &sessionObj)

	px2collectorrequest := cjsonIn{}
	px2collectorrequest.H2 = "true"
	px2collectorrequest.Url = collectorendpoint
	px2collectorrequest.Method = "POST"
	px2collectorrequest.Headers = append(px2collectorrequest.Headers, []map[string]string{{"Connection": "keep-alive"},{"Pragma": "no-cache"},{"Cache-Control": "no-cache"},{"Sec-Ch-Ua": sec},{"Sec-Ch-Ua-Mobile": "?0"},{"User-Agent": ua},{"Content-type": "application/x-www-form-urlencoded"},{"Accept": "*/*"},{"Sec-Fetch-Site": "same-origin"},{"Sec-Fetch-Mode": "cors"},{"Sec-Fetch-Dest": "empty"},{"Accept-Encoding": "gzip, deflate, br"}}...)
	px2collectorrequest.Body = bodyFormatMap(sessionObj)
	px2collectorresponse := &cjsonOut{}
	json.Unmarshal([]byte(C.GoString(MakeReq(px2collectorrequest, client))), px2collectorresponse)

	parsedPx2 := PX2RESHANDLER(px2collectorresponse.Source)
	count, _ := strconv.Atoi(sessionObj["seq"])
	
	rawbody = map[string]interface{}{
		"site": site,
		"pxtype": "PX3",
		"version": version,
		"tag": tag,
		"ua": ua,
		"count": strconv.Itoa(count+1),
		"uuid":  sessionObj["uuid"],
		"sid": 	 parsedPx2["sid"],
		"vid": 	 strings.SplitN(parsedPx2["vid"],"|",2)[0],
		"cs": 	 parsedPx2["cs"],
		"varobject": px2collectorresponse.Source,
	}
	if parsedPx2["pxhd"] != ""{
		rawbody["pxhd"] = parsedPx2["pxhd"]
	}
	rawbodybytes, _ = json.Marshal(rawbody)
	resp, _ = http.Post(pxHost, "application/json", bytes.NewBuffer(rawbodybytes))
	sessionObj = make(map[string]string)
	bodyBytes, _ = ioutil.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &sessionObj)

	px3collectorrequest := cjsonIn{}
	px3collectorrequest.H2 = "true"
	px3collectorrequest.Url = collectorendpoint
	px3collectorrequest.Method = "POST"
	px3collectorrequest.Headers = append(px3collectorrequest.Headers, []map[string]string{{"Connection": "keep-alive"},{"Pragma": "no-cache"},{"Cache-Control": "no-cache"},{"Sec-Ch-Ua": sec},{"Sec-Ch-Ua-Mobile": "?0"},{"User-Agent": ua},{"Content-type": "application/x-www-form-urlencoded"},{"Accept": "*/*"},{"Sec-Fetch-Site": "same-origin"},{"Sec-Fetch-Mode": "cors"},{"Sec-Fetch-Dest": "empty"},{"Accept-Encoding": "gzip, deflate, br"}}...)
	px3collectorrequest.Body = bodyFormatMap(sessionObj)

	resbody := jsonOutParse(MakeReq(px3collectorrequest, client))

	px3cookie := PX3COOKIEHANDLER(PX2RESHANDLER(resbody.Source)["bake"])
	cookiestring := COOKIESTRINGFORMATTER(resbody.Cookies)

	globalUhandler.Store(sessionObj["uuid"], UUIDSTORE{
		SID: sessionObj["sid"],
		VID: sessionObj["vid"],
		CS: sessionObj["cs"],
		PXHD: sessionObj["pxhd"],
		SEQ: sessionObj["seq"],
		UA: ua,
		SEC: sec,
		SITE: site,
		TAG: tag,
		VERSION: version,
		COOKIES: cookiestring + "_px3=" + px3cookie,
		ENDPOINT: collectorendpoint,
		SITEKEY: sitekey,
	})

	return C.CString(px3cookie), C.CString(sessionObj["uuid"])
}

//export pxEvent
func pxEvent(uuidP *C.char) *C.char{
	var uuidobj UUIDSTORE

	uuid := C.GoString(uuidP)
	if uuidinterface, ok := globalUhandler.Load(uuid); ok{
		uuidobj = uuidinterface.(UUIDSTORE)
	}


	client, err := httpClient("chrome89", uuidobj.ENDPOINT)
	if err != nil {
		return nil
	}

	count, _ := strconv.Atoi(uuidobj.SEQ)
	rawbody := map[string]interface{}{
		"site": 		uuidobj.SITE,
		"pxtype": 		"EVENT",
		"version": 		uuidobj.VERSION,
		"tag": 			uuidobj.TAG,
		"ua": 			uuidobj.UA,
		"count": 		strconv.Itoa(count+1),
		"uuid":  		uuid,
		"sid": 	 		uuidobj.SID,
		"vid": 	 		uuidobj.VID,
		"cs": 	 		uuidobj.CS,
	}
	if uuidobj.PXHD != ""{
		rawbody["pxhd"] = uuidobj.PXHD
	}
	rawbodybytes, _ := json.Marshal(rawbody)
	resp, _ := http.Post(pxHost, "application/json", bytes.NewBuffer(rawbodybytes))
	sessionObj := make(map[string]string)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &sessionObj)

	px3collectorrequest := cjsonIn{}
	px3collectorrequest.H2 = "true"
	px3collectorrequest.Url = uuidobj.ENDPOINT
	px3collectorrequest.Method = "POST"
	px3collectorrequest.Headers = append(px3collectorrequest.Headers, []map[string]string{{"Connection": "keep-alive"},{"Pragma": "no-cache"},{"Cache-Control": "no-cache"},{"Sec-Ch-Ua": uuidobj.SEC},{"Sec-Ch-Ua-Mobile": "?0"},{"User-Agent": uuidobj.UA},{"Content-type": "application/x-www-form-urlencoded"},{"Accept": "*/*"},{"Sec-Fetch-Site": "same-origin"},{"Sec-Fetch-Mode": "cors"},{"Sec-Fetch-Dest": "empty"},{"Accept-Encoding": "gzip, deflate, br"}}...)
	px3collectorrequest.Headers = append(px3collectorrequest.Headers, map[string]string{"Cookie": uuidobj.COOKIES})
	px3collectorrequest.Body = bodyFormatMap(sessionObj)

	resbody := jsonOutParse(MakeReq(px3collectorrequest, client))

	px3cookie := PX3COOKIEHANDLER(PX2RESHANDLER(resbody.Source)["bake"])
	cookiestring := COOKIESTRINGFORMATTER(resbody.Cookies)

	globalUhandler.LoadOrStore(uuid, UUIDSTORE{
		SID: 		uuidobj.SID,
		VID: 		uuidobj.VID,
		CS: 		uuidobj.CS,
		PXHD: 		uuidobj.PXHD,
		SEQ: 		strconv.Itoa(count+1),
		UA: 		uuidobj.UA,
		SEC: 		uuidobj.SEC,
		SITE: 		uuidobj.SITE,
		TAG: 		uuidobj.TAG,
		VERSION: 	uuidobj.VERSION,
		COOKIES: 	cookiestring + "_px3=" + px3cookie,
		ENDPOINT: 	uuidobj.ENDPOINT,
		SITEKEY:    uuidobj.SITEKEY,
	})

	return C.CString(px3cookie)
}

//export pxCaptcha
func pxCaptcha(uuidP, tokenP, captchaTypeP *C.char) *C.char{
	var uuidobj UUIDSTORE

	uuid := C.GoString(uuidP)
	if uuidinterface, ok := globalUhandler.Load(uuid); ok{
		uuidobj = uuidinterface.(UUIDSTORE)
	}

	captchaEndpoint := fmt.Sprintf("https://collector-%s.perimeterx.net/assets/js/bundle",uuidobj.SITEKEY)

	token := C.GoString(tokenP)
	captchaType := C.GoString(captchaTypeP)

	client, err := httpClient("chrome89", uuidobj.ENDPOINT)
	if err != nil {
		return nil
	}

	count, _ := strconv.Atoi(uuidreadmap[uuid].SEQ)
	rawbody := map[string]interface{}{
		"site": 		uuidobj.SITE,
		"version": 		uuidobj.VERSION,
		"tag": 			uuidobj.TAG,
		"ua": 			uuidobj.UA,
		"count": 		strconv.Itoa(count+1),
		"uuid":  		uuid,
		"sid": 	 		uuidobj.SID,
		"vid": 	 		uuidobj.VID,
		"cs": 	 		uuidobj.CS,
	}
	if uuidreadmap[uuid].PXHD != ""{
		rawbody["pxhd"] = uuidreadmap[uuid].PXHD
	}

	switch(captchaType){
		case "hcap":
			rawbody["pxtype"] = "HCAPHIGH"
		case "hcaplow":
			rawbody["pxtype"] = "HCAPLOW"
		case "recap":
			rawbody["pxtype"] = "RECAP"
			rawbody["token"] = token
	}

	rawbodybytes, _ := json.Marshal(rawbody)
	resp, _ := http.Post(pxHost, "application/json", bytes.NewBuffer(rawbodybytes))
	sessionObj := make(map[string]string)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &sessionObj)

	captcharequest := cjsonIn{}
	captcharequest.H2 = "true"
	captcharequest.Url = captchaEndpoint
	captcharequest.Method = "POST"
	captcharequest.Headers = append(captcharequest.Headers, []map[string]string{{"Connection": "keep-alive"},{"Pragma": "no-cache"},{"Cache-Control": "no-cache"},{"Sec-Ch-Ua": uuidobj.SEC},{"Sec-Ch-Ua-Mobile": "?0"},{"User-Agent": uuidobj.UA},{"Content-type": "application/x-www-form-urlencoded"},{"Accept": "*/*"},{"Sec-Fetch-Site": "same-origin"},{"Sec-Fetch-Mode": "cors"},{"Sec-Fetch-Dest": "empty"},{"Accept-Encoding": "gzip, deflate, br"}}...)
	captcharequest.Headers = append(captcharequest.Headers, map[string]string{"Cookie": uuidobj.COOKIES})
	captcharequest.Body = bodyFormatMap(sessionObj)

	resbody := jsonOutParse(MakeReq(captcharequest, client))

	px3cookie := PX3COOKIEHANDLER(PX2RESHANDLER(resbody.Source)["bake"])
	cookiestring := COOKIESTRINGFORMATTER(resbody.Cookies)

	globalUhandler.LoadOrStore(uuid, UUIDSTORE{
		SID: 		uuidobj.SID,
		VID: 		uuidobj.VID,
		CS: 		uuidobj.CS,
		PXHD: 		uuidobj.PXHD,
		SEQ: 		strconv.Itoa(count+1),
		UA: 		uuidobj.UA,
		SEC: 		uuidobj.SEC,
		SITE: 		uuidobj.SITE,
		TAG: 		uuidobj.TAG,
		VERSION: 	uuidobj.VERSION,
		COOKIES: 	cookiestring + "_px3=" + px3cookie,
		ENDPOINT: 	uuidobj.ENDPOINT,
		SITEKEY:    uuidobj.SITEKEY,
	})

	return C.CString(px3cookie)
}


//export retrieve
func retrieve(valin *C.char) *C.char{
	message :=  C.GoString(valin)
	return C.CString(message)
}