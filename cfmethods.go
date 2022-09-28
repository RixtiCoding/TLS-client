package main
import (
	"C"
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

//export loadChallenges
func loadChallenges(infoIn *C.char) *C.char {
	defer func() {
		if err := recover(); err != nil {
		}
	}()

	l := launcher.New().Headless(false)

	defer l.Cleanup()

	curl := l.MustLaunch()

	browser := rod.New().
		ControlURL(curl).
		MustConnect()


	page := stealth.MustPage(browser)
	page.MustSetWindow(-500,-500,1,1)
	page.MustWindowMinimize()
	router := page.HijackRequests()
	defer router.Stop()

	p := genjson{}
	message :=  C.GoString(infoIn)
	err := json.Unmarshal([]byte(message), &p)
	if err != nil {
		return C.CString("")
	}

	var client *http.Client
	var clientString string

	if p.Proxy != ""{
		client, err = proxyClient("chrome", p.Proxy)
		if err != nil {
			return C.CString("")
		}
		clientString = p.Proxy
	}else{
		parsedUrl, err := url.Parse(p.Url)
		if err != nil {
			return C.CString("")
		}
		host := parsedUrl.Scheme + "://" + parsedUrl.Host

		client, err = httpClient("chrome", host)
		if err != nil {
			return C.CString("")
		}
		clientString = host
	}

	cleared := false

	router.MustAdd("*", func(ctx *rod.Hijack) {
		if strings.Contains(ctx.Request.URL().String(), "__cf_chl_jschl_tk__") {
			cleared = true
			go sendForm(*ctx.Request.Req().Clone(context.Background()), client, clientString)
		}else{
			ctx.LoadResponse(client, true)
		}
	})

	go router.Run()
	page.MustNavigate(p.Url)
	time.Sleep(500 * time.Millisecond)
	page.MustEval("window._cf_chl_done && window._cf_chl_done()")
	time.Sleep(500 * time.Millisecond)
	for !cleared{
		page.MustEval("window._cf_chl_done && window._cf_chl_done()")
		time.Sleep(500 * time.Millisecond)
	}
	browser.MustClose()
	return C.CString(clientString)
}
func sendForm(reqIn http.Request, clientIn *http.Client, cs string) {
	time.Sleep(4 * time.Second)
	resp, _ := clientIn.Do(&reqIn)

	//headerstring, _ := json.Marshal(resp.Header)

	for _, val := range resp.Header["Set-Cookie"]{
		if strings.Contains(val, "cf_clearance"){
			pool, _ := ClientHandler.Load(cs)
			storepool := pool.(*sync.Pool)
			storepool.Put(clientIn)
			requestsStores[cs] = strings.Split(val, " ")[0]
		}
	}
}

// I used seedP - inst first - because you wont be able to specify proxy otherwise

//export cfRequestChain
func cfRequestChain(hostUrl, chua, ua, proxyP *C.char) *C.char{
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	sessionObj := make(map[string]string)
	var cookieJar []*http.Cookie

	proxy := C.GoString(proxyP)
	host := C.GoString(hostUrl)
	secchua := C.GoString(chua)
	secua := C.GoString(ua)
	parsedUrl, _ := url.Parse(host)
	baseUrl := parsedUrl.Scheme + "://" + parsedUrl.Host

	var client *http.Client
	var pool *sync.Pool

	var connType string
	var storeTag string
	if proxy != ""{
		storeTag = proxy
		connType = "proxy"
	}else {
		storehost := parsedUrl.Scheme + "://" + parsedUrl.Host
		connType = "raw"
		storeTag = storehost
	}

	if _, ok := ClientHandler.Load(storeTag);!ok{
		ClientHandler.Store(storeTag, &sync.Pool{})
	}

	if val, ok := ClientHandler.Load(storeTag); ok {
		pool = val.(*sync.Pool)
		clientused := pool.Get()
		if clientused != nil{
			client = clientused.(*http.Client)
		}else{
			var err error
			switch connType{
				case "raw":
					client, err = httpClient("chrome89", storeTag)
					if err != nil {
						return nil
					}
				case "proxy":
					client, err = proxyClient("chrome89", proxy)
					if err != nil {
						return nil
					}
				}
		}
	}


	htmlReq := *new(cjsonIn)
	htmlReq.H2 = "true"
	htmlReq.Url = host
	htmlReq.Method = "GET"
	htmlReq.Headers = append(htmlReq.Headers, []map[string]string{{`sec-ch-ua`: secchua},{`sec-ch-ua-mobile`: `?0`},{`upgrade-insecure-requests`: `1`},{`user-agent`: secua},{`accept`: `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`},{`sec-fetch-site`: `none`},{`sec-fetch-mode`: `navigate`},{`sec-fetch-user`: `?1`},{`sec-fetch-dest`: `document`},{`accept-encoding`: `gzip, deflate, br`},{`accept-language`: `en-US,en;q=0.9`}}...)
	htmlResp := jsonOutParse(MakeReq(htmlReq, client))


	cookieJar = htmlResp.Cookies
	htmlBytes, _ := json.Marshal(map[string]string{"body":htmlResp.Source})
	htmlServerResp, _ := http.Post(cfHost + "/htmlinit", "application/json", bytes.NewBuffer(htmlBytes))
	bodyBytes, _ := ioutil.ReadAll(htmlServerResp.Body)
	json.Unmarshal(bodyBytes, &sessionObj)

	jsReq := *new(cjsonIn)
	jsReq.H2 = "true"
	jsReq.Url = baseUrl + "/cdn-cgi/challenge-platform/h/b/orchestrate/jsch/v1"
	jsReq.Method = "GET"
	setCookieStr := ""
	for _, cookie := range cookieJar{
		setCookieStr += cookie.Name+"="+cookie.Value+";"
	}
	jsReq.Headers = append(jsReq.Headers, []map[string]string{{`sec-ch-ua`: secchua},{`sec-ch-ua-mobile`: `?0`},{`user-agent`: secua},{`accept`: `*/*`},{`sec-fetch-site`: `same-origin`},{`sec-fetch-mode`: `no-cors`},{`sec-fetch-dest`: `script`},{`referer`: `https://www.topps.com/`},{`accept-encoding`: `gzip, deflate, br`},{`accept-language`: `en-US,en;q=0.9`}}...)
	jsResp := jsonOutParse(MakeReq(jsReq, client))

	jsBytes, _ := json.Marshal(map[string]string{"body": jsResp.Source, "rayhash":sessionObj["cRay"]+sessionObj["cHash"]})
	jsServerResp, _  := http.Post(cfHost + "/jsinit", "application/json", bytes.NewBuffer(jsBytes))
	bodyBytes, _ = ioutil.ReadAll(jsServerResp.Body)
	jsstruct := make(map[string]string)
	json.Unmarshal(bodyBytes, &jsstruct)
	sessionObj["challengendpoint"] = jsstruct["endpoint"]

	jsImageReq := *new(cjsonIn)
	jsImageReq.H2 = "true"
	jsImageReq.Url = baseUrl + "/cdn-cgi/images/trace/jschal/js/transparent.gif?ray=" + sessionObj["cRay"]
	jsImageReq.Method = "GET"
	jsImageReq.Headers = append(jsImageReq.Headers, []map[string]string{{`sec-ch-ua`: secchua}, {`sec-ch-ua-mobile`: `?0`}, {`user-agent`: secua}, {`accept`: `image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8`}, {`sec-fetch-site`: `same-origin`}, {`sec-fetch-mode`: `no-cors`}, {`sec-fetch-dest`: `image`}, {`referer`: `https://www.topps.com/`}, {`accept-encoding`: `gzip, deflate, br`}, {`accept-language`: `en-US,en;q=0.9`}, {"Cookie": setCookieStr}}...)
	MakeReq(jsImageReq, client)

	nojsImageReq := *new(cjsonIn)
	nojsImageReq.H2 = "true"
	nojsImageReq.Url = baseUrl + "/cdn-cgi/images/trace/jschal/nojsImage/transparent.gif?ray=" + sessionObj["cRay"]
	nojsImageReq.Method = "GET"
	nojsImageReq.Headers = append(nojsImageReq.Headers, []map[string]string{{`sec-ch-ua`: secchua},{`sec-ch-ua-mobile`: `?0`}, {`user-agent`: secua}, {`accept`: `image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8`}, {`sec-fetch-site`: `same-origin`}, {`sec-fetch-mode`: `no-cors`}, {`sec-fetch-dest`: `image`}, {`referer`: `https://www.topps.com/`}, {`accept-encoding`: `gzip, deflate, br`}, {`accept-language`: `en-US,en;q=0.9`}, {"Cookie": setCookieStr}}...)
	MakeReq(nojsImageReq, client)

	challengeOneReq := *new(cjsonIn)
	challengeOneReq.H2 = "true"
	challengeOneReq.Url = baseUrl + sessionObj["challengendpoint"]
	challengeOneReq.Method = "POST"
	challengeOneReq.Headers = append(challengeOneReq.Headers, []map[string]string{{"Host": parsedUrl.Host}, {"Connection": "keep-alive"}, {"Content-Length": strconv.Itoa(len(jsstruct["payload"])+2)}, {"Pragma": "no-cache"}, {"Cache-Control": "no-cache"}, {"Sec-Ch-Ua": secchua}, {"Sec-Ch-Ua-Mobile": "?0"}, {"User-Agent": secua}, {"CF-Challenge": sessionObj["cHash"]}, {"Content-type": "application/x-www-form-urlencoded"}, {"Accept": "*/*"}, {"Origin": baseUrl}, {"Sec-Fetch-Site": "same-origin"}, {"Sec-Fetch-Mode": "cors"}, {"Sec-Fetch-Dest": "empty"}, {"Referer": host}, {"Accept-Encoding": "gzip, deflate, br"}, {"Accept-Language": "en-US,en;q=0.9"}, {"Cookie": setCookieStr+" cf_chl_prog=e"}}...)
	payload := strings.Replace(jsstruct["payload"], "+","%2b",1)
	challengeOneReq.Body = payload
	challengeOneResp := jsonOutParse(MakeReq(challengeOneReq, client))
	for _, cookie := range challengeOneResp.Cookies{
		cookieJar = append(cookieJar, cookie)
	}

	if challengeOneResp.Source == "Invalid request"{
		return C.CString("Invalid First Request")
	}

	challengeOneBytes, _ := json.Marshal(map[string]string{"body": challengeOneResp.Source, "rayhash":sessionObj["cRay"]+sessionObj["cHash"], "host": baseUrl, "useragent": secua})
	challengeOneServerResp, _  := http.Post(cfHost + "/challengeone", "application/json", bytes.NewBuffer(challengeOneBytes))
	bodyBytes, _ = ioutil.ReadAll(challengeOneServerResp.Body)
	json.Unmarshal(bodyBytes, &jsstruct)

	challengetwoReq := *new(cjsonIn)
	challengetwoReq.H2 = "true"
	challengetwoReq.Url = baseUrl + sessionObj["challengendpoint"]
	challengetwoReq.Method = "POST"
	challengetwoReq.Headers = append(challengetwoReq.Headers, []map[string]string{{"Host": parsedUrl.Host},{"Connection": "keep-alive"},{"Content-Length": strconv.Itoa(len(jsstruct["payload"])+2)},{"Pragma": "no-cache"},{"Cache-Control": "no-cache"},{"Sec-Ch-Ua": secchua},{"Sec-Ch-Ua-Mobile": "?0"},{"User-Agent": secua},{"CF-Challenge": sessionObj["cHash"]},{"Content-type": "application/x-www-form-urlencoded"},{"Accept": "*/*"},{"Origin": baseUrl},{"Sec-Fetch-Site": "same-origin"},{"Sec-Fetch-Mode": "cors"},{"Sec-Fetch-Dest": "empty"},{"Referer": host},{"Accept-Encoding": "gzip, deflate, br"},{"Accept-Language": "en-US,en;q=0.9"}}...)
	setCookieStr = cookieJar[1].Name+"="+cookieJar[1].Value+"; "+cookieJar[0].Name+"="+cookieJar[0].Value+"; "+"cf_chl_prog=a"+jsstruct["a"]
	challengetwoReq.Headers = append(challengetwoReq.Headers, map[string]string{"Cookie": setCookieStr})
	payload = strings.Replace(jsstruct["payload"], "+","%2b",1)
	challengetwoReq.Body = payload
	challengetwoResp := jsonOutParse(MakeReq(challengetwoReq, client))

	challengeTwoBytes, _ := json.Marshal(map[string]string{"body": challengetwoResp.Source, "rayhash":sessionObj["cRay"]+sessionObj["cHash"], "host": baseUrl})
	challengeTwoServerResp, _  := http.Post(cfHost + "/challengetwo", "application/json", bytes.NewBuffer(challengeTwoBytes))
	bodyBytes, _ = ioutil.ReadAll(challengeTwoServerResp.Body)
	if string(bodyBytes) == "reload"{
		return C.CString("reload")
	}
	json.Unmarshal(bodyBytes, &jsstruct)

	finalFormSubmit := *new(cjsonIn)
	finalFormSubmit.H2 = "true"
	finalFormSubmit.Url = baseUrl + jsstruct["endpoint"]
	finalFormSubmit.Method = "POST"
	challengetwoReq.Headers = append(challengetwoReq.Headers, []map[string]string{{"Host": parsedUrl.Host},{"Connection": "keep-alive"},{"Content-Length": strconv.Itoa(len(jsstruct["payload"])+2)},{"Pragma": "no-cache"},{"Cache-Control": "no-cache"},{"Sec-Ch-Ua": secchua},{"Sec-Ch-Ua-Mobile": "?0"},{"User-Agent": secua},{"CF-Challenge": sessionObj["cHash"]},{"Content-type": "application/x-www-form-urlencoded"},{"Accept": "*/*"},{"Origin": baseUrl},{"Sec-Fetch-Site": "same-origin"},{"Sec-Fetch-Mode": "cors"},{"Sec-Fetch-Dest": "empty"},{"Referer": host},{"Accept-Encoding": "gzip, deflate, br"}}...)
	finalPayload := "r="+jsstruct["r"]+"&jschl_vc="+jsstruct["vc"]+"&pass="+jsstruct["pass"]+"&jschl_answer="+jsstruct["answer"]+"&cf_ch_verify=plat"
	finalFormSubmit.Body = finalPayload
	finalFormResp := jsonOutParse(MakeReq(finalFormSubmit, client))

	time.Sleep(4*time.Second)
	log.Print(finalFormResp.Cookies)
	resString := ""
	for _, cookie := range finalFormResp.Cookies{
		if cookie.Name == "cf_clearance"{
			resString = cookie.Name + "=" + cookie.Value
		}
	}
	pool.Put(client)
	return C.CString(resString)
}