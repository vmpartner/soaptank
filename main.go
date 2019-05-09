package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"net/http"
	"os"
	"soaptank/modules"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Stat struct {
	Req   string
	Res   string
	URL   string
	Code  int
	Stack int
	Time  int64
	Err   string
}

type Params map[string]interface{}

var chStat chan Stat

func main() {

	params := make(Params)
	var configPath string
	var varPath string
	var reqPath string
	chStat = make(chan Stat, 100)

	// Parse flags
	flag.StringVar(&configPath, "config", "case1\\app.conf", "set file config path")
	flag.StringVar(&reqPath, "req", "case1\\req.xml", "set file req path")
	flag.StringVar(&varPath, "var", "case1\\vars.txt", "set file var path")
	flag.Parse()

	// Parse config
	pwd, _ := os.Getwd()
	configPath = pwd + `\` + configPath
	varPath = pwd + `\` + varPath

	cfg, err := ini.ShadowLoad(configPath)
	modules.CheckErr(err)

	// Load vars
	f, err := os.Open(varPath)
	if err != nil {
		panic(err)
	}
	params["varData"], _ = ioutil.ReadAll(f)
	f.Close()

	// Load request
	f, err = os.Open(reqPath)
	if err != nil {
		panic(err)
	}
	params["reqData"], _ = ioutil.ReadAll(f)
	f.Close()

	// Get urls
	urls := cfg.Section("soap").Key("url").ValueWithShadows()

	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	go func() {

		defer wg2.Done()

		os.Remove("stat.csv")
		f, err := os.Create("stat.csv")
		modules.CheckErr(err)
		defer f.Close()

		w := bufio.NewWriter(f)
		defer w.Flush()

		logRow := "URL;REQ;RES;CODE;TIME;STACK\r\n"
		fmt.Println(logRow)
		_, err = w.WriteString(logRow)

		for {
			row, more := <-chStat
			if more {
				logRow := row.URL + ";" + row.Req + ";" + row.Res + ";" + strconv.Itoa(row.Code) + ";" + strconv.Itoa(int(row.Time)) + ";" + strconv.Itoa(int(row.Stack)) + "\r\n"
				fmt.Println(logRow)
				_, err = w.WriteString(logRow)
				modules.CheckErr(err)
			} else {
				fmt.Println("finish")
				return
			}
		}
	}()

	wg := sync.WaitGroup{}
	for i, url := range urls {
		wg.Add(1)
		i++
		go runTask(i, url, cfg, varPath, &wg, params)
	}
	wg.Wait()
	close(chStat)
	wg2.Wait()
}

func runTask(ii int, url string, cfg *ini.File, varPath string, wg *sync.WaitGroup, params Params) {

	defer wg.Done()

	fmt.Println("start task " + strconv.Itoa(ii))

	// Get config threads
	threads, err := cfg.Section("main").Key("threads").Int()
	if err != nil {
		panic(err)
	}
	requests, err := cfg.Section("main").Key("requests").Int()
	if err != nil {
		panic(err)
	}

	// Start channels
	wg1 := sync.WaitGroup{}
	chTask := make(chan string, requests)
	for i := 1; i <= threads; i++ {
		wg1.Add(1)
		go runThread(strconv.Itoa(ii)+"_"+strconv.Itoa(i), url, chTask, &wg1, params)
	}

	// Generate tasks
	for i := 0; i < requests; i++ {
		textData := strings.Split(string(params["varData"].([]byte)), "\r\n")
		for _, rowText := range textData {
			rowData := strings.Split(rowText, "?")
			reqText := string(params["reqData"].([]byte))
			for _, vv := range rowData {
				reqText = strings.Replace(reqText, "?", vv, 1)
			}
			if i >= requests {
				break
			}
			i++
			time.Sleep(1 * time.Second)
			chTask <- reqText
		}
	}

	fmt.Println("finish task " + strconv.Itoa(ii))

	// for all vars and req
	close(chTask)
	wg1.Wait()
}

func runThread(i string, url string, chTask chan string, wg1 *sync.WaitGroup, params Params) {

	defer wg1.Done()

	fmt.Println("start thread " + i)

	for {
		req, more := <-chTask
		if more {

			t := time.Now()

			resp, err := soapCall(url, req)
			modules.CheckErr(err)
			//body, err := ioutil.ReadAll(resp.Body)

			d := time.Now().Sub(t)

			var stat Stat
			//stat.Req = strings.ReplaceAll(req, "\r\n", "")
			//stat.Res = strings.ReplaceAll(string(body), "\n", "")
			stat.Stack = len(chTask)
			stat.Time = d.Nanoseconds() / 1000000
			stat.Code = resp.StatusCode
			stat.URL = url
			stat.URL = url
			if err != nil {
				stat.Err = err.Error()
			}

			chStat <- stat

		} else {
			fmt.Println("finish thread " + i)
			return
		}
	}
}

func soapCall(url string, data string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{}

	req.ContentLength = int64(len(data))

	req.Header.Add("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Add("Accept", "text/xml")
	req.Header.Add("SOAPAction", url)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
