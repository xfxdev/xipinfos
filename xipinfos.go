package xipinfos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

var (
	ipQueryURL = "http://ip-api.com/batch?fields=query,status,country,city,org,isp"
)

type ipQueryItem struct {
	IP string `json:"query,omitempty"`
}
type IPQueryResult struct {
	IP      string `json:"query,omitempty"`
	Status  string `json:"status,omitempty"`
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
	Org     string `json:"org,omitempty"`
	ISP     string `json:"isp,omitempty"`
}

func (ret *IPQueryResult) String() string {
	return fmt.Sprintf("ip : %q, country : %q, org : %q, isp : %q", ret.IP, ret.Country+"-"+ret.City, ret.Org, ret.ISP)
}

func doQuery(items []*ipQueryItem, retChannel chan<- *IPQueryResult) error {
	buf, err := json.Marshal(items)
	if err != nil {
		return err
	}

	resp, err := http.Post(ipQueryURL, "application/json; charset=utf-8", bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	var queryRets []*IPQueryResult
	err = json.Unmarshal(respBuf, &queryRets)
	if err != nil {
		return err
	}

	for _, ret := range queryRets {
		retChannel <- ret
	}

	return nil
}

func BatchQuery(ips []string) []*IPQueryResult {
	var batchedIPQueryItems []*ipQueryItem
	mapIps := make(map[string]bool)
	for _, ip := range ips {
		if _, ok := mapIps[ip]; !ok {
			batchedIPQueryItems = append(batchedIPQueryItems, &ipQueryItem{IP: ip})
			mapIps[ip] = true
		}
	}

	wg := sync.WaitGroup{}

	chanQueryRet := make(chan *IPQueryResult, len(batchedIPQueryItems))

	startIdx := 0
	for {
		endIdx := startIdx + 100
		if endIdx > len(batchedIPQueryItems) {
			endIdx = len(batchedIPQueryItems)
		}

		wg.Add(1)
		go func(start, end int) {
			doQuery(batchedIPQueryItems[start:end], chanQueryRet)
			wg.Done()
		}(startIdx, endIdx)

		startIdx = endIdx

		if endIdx >= len(batchedIPQueryItems) {
			break
		}
	}

	wg.Wait()
	close(chanQueryRet)

	rets := make([]*IPQueryResult, 0, len(chanQueryRet))
	for ret := range chanQueryRet {
		rets = append(rets, ret)
	}

	return rets
}
