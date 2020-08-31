package main

import (
	"encoding/json"
	"fmt"
	"github.com/LeakIX/LeakIXClient"
	"gitlab.nobody.run/tbi/core"
	"net/http"
	"strconv"
	"time"
)

var done map[string]int64
// Make sure ALL_PROXY is supported
var SurveyHttpClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           core.ProxiedPlugin{}.DialContext,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	},
	Timeout:   5 * time.Second,
}

func main() {
	done = make(map[string]int64)
	// Create a searcher
	LeakIXSearch := LeakIXClient.SearchResultsClient{
		Scope: "leak",
		Query: "+plugin:ElasticSearchExplorePlugin +\"meow\"",
	}
	// Iterate, the lib will query further pages if needed
	for LeakIXSearch.Next() {
		// Use the result
		leak := LeakIXSearch.SearchResult()
		// if already queried, continue
		if _, found := done[leak.Ip + leak.Port]; found {
			continue
		}
		// Check remote server & print data
		// TODO : Limit processes
		go investigate(leak.Ip, leak.Port)
		done[leak.Ip + leak.Port] = time.Now().Unix()
	}
	// Just stop me when there's no more output ( sorry )
	time.Sleep(600*time.Second)
}
// Connects to remote server and print out indices details
func investigate(ip, port string) {
	url := fmt.Sprintf("http://%s:%s/*", ip,  port)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "LeakIX-Survey-Meow/0.0.0 (+https://leakix.net)")
	resp, err := SurveyHttpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	esReply := EsReply{}
	jsonDecoder := json.NewDecoder(resp.Body)
	err = jsonDecoder.Decode(&esReply)
	if err != nil {
		return
	}
	for indexName, indexSettings := range esReply {
		unixDate, err := strconv.ParseInt(indexSettings.Settings.Index.CreationDate[0:10], 10, 64)
		if err != nil {
			continue
		}
		date := time.Unix(unixDate, 0)
		fmt.Printf("%s : [%s:%s] %s\n", date.String(), ip, port, indexName)
	}
}

type EsReply map[string]struct{
	Settings struct{
		Index struct {
			CreationDate string `json:"creation_date"`
		} `json:"index"`
	} `json:"settings"`
}