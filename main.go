package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"

	n "github.com/npenkov/neterra-tv-proxy/neterra"
	"golang.org/x/net/publicsuffix"
)

const (
	EPGRedirectURL = "http://epg.kodibg.org/dl.php"
)

func main() {
	var channelsDataFile, username, password, host, port string
	var logging bool

	username = os.Getenv("USERNAME")
	password = os.Getenv("PASSWORD")
	host = os.Getenv("HOST")
	port = os.Getenv("PORT")

	if username == "" {
		log.Fatal("USERNAME env variable not defined")
	}

	if password == "" {
		log.Fatal("PASSWORD env variable not defined")
	}

	if host == "" {
		log.Fatal("HOST env variable not defined")
	}

	if port == "" {
		log.Fatal("PORT env variable not defined")
	}

	flag.StringVar(&channelsDataFile, "ch", "./data/channels.json", "Cannels data file")
	flag.BoolVar(&logging, "v", false, "Verbose")

	flag.Parse()

	raw, err := ioutil.ReadFile(channelsDataFile)
	if err != nil {
		log.Fatal(err)
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	var tvChannels n.Channels
	json.Unmarshal(raw, &tvChannels)

	nClient := n.NewClient(client, username, password, host, port, tvChannels)

	mux := http.NewServeMux()
	mux.HandleFunc("/epg.xml", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, EPGRedirectURL, http.StatusMovedPermanently)
	})
	mux.HandleFunc("/playlist.m3u8", func(w http.ResponseWriter, req *http.Request) {
		if logging {
			log.Println("== playlist.m3u8 called")
		}
		ch, okCh := req.URL.Query()["ch"]
		chName, okName := req.URL.Query()["name"]

		if okCh && len(ch) > 0 {
			if !okName || len(chName) < 1 {
				if logging {
					log.Println("Url Param 'chName' is missing")
				}
				return
			}

			w.Header().Set("Content-Type", "application/x-mpegURL")
			url, err := nClient.GetStream(ch[0])
			if err != nil {
				if logging {
					log.Printf("Error fetching stream url for channel %s: %v\n", ch[0], err)
				}
				return
			}
			if logging {
				log.Printf("Serving: %s", url)
			}
			http.Redirect(w, req, url, http.StatusMovedPermanently)
			return
		}

		// Server playlist
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Content-Disposion", "filename=\"playlist.m3u8\"")

		res, err := nClient.GetM3U8()
		if err != nil {
			if logging {
				log.Printf("Error fetching list: %v\n", err)
			}
		}
		if logging {
			log.Printf("Serving: %s", res)
		}
		w.Write([]byte(res))
	})
	if logging {
		log.Println("----- Starting Neterra.tv Proxy ----")
	}
	http.ListenAndServe(fmt.Sprintf(":%s", port), mux)

}
