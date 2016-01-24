package main

import (
	"code.google.com/p/gcfg"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	General struct {
		ListenPort  string
		StaticDir   string
		ApplyScript string
		AuthToken   string
	}
}

var cfg Config

type applyEntry struct {
	ipAddress net.IP
	expireAt  time.Time
}

func (a *applyEntry) expire() {
}

var applyList []applyEntry
var listLock sync.Mutex

type applyResponse struct {
	Success bool `json:"success"`
}

type purgeResponse struct {
	Success bool `json:"success"`
}

type pingResponse struct {
	NoState bool      `json:"no_state"`
	Valid   bool      `json:"valid"`
	Expires time.Time `json:"expires"`
}

func getClientIP(req *http.Request) (net.IP, error) {
	var ret net.IP
	args := strings.Split(req.RemoteAddr, ":")
	if len(args) != 2 {
		return ret, fmt.Errorf("error obtaining client address")
	}

	ret = net.ParseIP(args[0])
	return ret, nil
}

func handlePurge(rw http.ResponseWriter, req *http.Request) {
	pr := purgeResponse{Success: true}

	cip, err := getClientIP(req)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	listLock.Lock()
	newlist := make([]applyEntry, 0)
	for _, x := range applyList {
		if cip.Equal(x.ipAddress) {
			fmt.Fprintf(os.Stdout, "purge %v\n", x)
			continue
		}
		newlist = append(newlist, x)
	}
	applyList = newlist
	listLock.Unlock()

	buf, err := json.Marshal(pr)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, string(buf))
}

func handlePing(rw http.ResponseWriter, req *http.Request) {
	pr := pingResponse{Valid: true, NoState: true}

	cip, err := getClientIP(req)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	listLock.Lock()
	for _, x := range applyList {
		if cip.Equal(x.ipAddress) {
			pr.NoState = false
			pr.Expires = x.expireAt
			break
		}
	}
	listLock.Unlock()

	buf, err := json.Marshal(pr)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, string(buf))
}

func handleApply(rw http.ResponseWriter, req *http.Request) {
	req.ParseMultipartForm(10000)

	durval := req.FormValue("duration")
	if durval == "" {
		http.Error(rw, "no duration specified", 500)
		return
	}
	dur, err := time.ParseDuration(durval)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	pr := applyResponse{Success: true}
	buf, err := json.Marshal(pr)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	cip, err := getClientIP(req)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	listLock.Lock()
	ne := applyEntry{}
	ne.ipAddress = cip
	ne.expireAt = time.Now().Add(dur)

	applyList = append(applyList, ne)
	listLock.Unlock()
	fmt.Fprintf(os.Stdout, "add state %v\n", ne)

	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, string(buf))
}

func reaper() {
	for {
		time.Sleep(time.Second)
		listLock.Lock()
		newlist := make([]applyEntry, 0)
		for _, x := range applyList {
			n := time.Now()
			if n.After(x.expireAt) {
				fmt.Fprintf(os.Stdout, "remove %v\n", x)
				x.expire()
			} else {
				newlist = append(newlist, x)
			}
		}
		applyList = newlist
		listLock.Unlock()
	}
}

func main() {
	var (
		confPath string
	)

	flag.StringVar(&confPath, "f", "./ipcapcom.conf", "path to config file")
	flag.Parse()
	err := gcfg.ReadFileInto(&cfg, confPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/ping", handlePing).Methods("GET")
	r.HandleFunc("/apply", handleApply).Methods("POST")
	r.HandleFunc("/purge", handlePurge).Methods("GET")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(cfg.General.StaticDir)))
	http.Handle("/", context.ClearHandler(r))
	go reaper()
	err = http.ListenAndServe(":"+cfg.General.ListenPort, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
