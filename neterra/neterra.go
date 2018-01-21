package neterra

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	LoginUrl            = "http://www.neterra.tv/user/login_page"
	LoginParamUsername  = "login_username"
	LoginParamPassword  = "login_password"
	LoginParamLogin     = "login"
	LoginParamLoginType = "login_type"

	LoginValueLogin     = "1"
	LoginValueLoginType = "1"

	LoginSuccessCriteria = "var LOGGED = '1'"

	LoginTimeoutMs = 28800000

	GetStreamUrl          = "http://www.neterra.tv/content/get_stream"
	GetStreamParamIssue   = "issue_id"
	GetStreamParamQuality = "quality"
	GetStreamParamType    = "type"

	GetStreamValueQuality = "0"
	GetStreamValueType    = "live"

	GetLiveUrl = "http://www.neterra.tv/content/live"
)

type playLink struct {
	Link string `json:"play_link"`
}

// Client represents Neterra Client object
type Client struct {
	username   string
	password   string
	httpClient *http.Client
	lastLogin  time.Time
	loggedIn   bool
	tvChan     map[string]*Channel
	host       string
	port       string
}

// Channel represents information for the particular live stream meta
type Channel struct {
	IssueID string `json:"issue-id"`
	Name    string `json:"name"`
	TvgID   string `json:"tvg-id"`
	TvgName string `json:"tvg-name"`
	Group   string `json:"group"`
	Logo    string `json:"logo"`
}

// Channels is structure encapsulating list of channel definitions
type Channels struct {
	Channels []Channel `json:"channels"`
}

// LiveChannels is structure encapsulating list of live channel definitions - fetched from Neterra server
type LiveChannels struct {
	LiveChannels [][]LiveChannel `json:"tv_choice_result"`
}

// LiveChannel is structure fetched from Neterra TV service and has information for one streamed channel, along with is't program entries
type LiveChannel struct {
	ProductID      string           `json:"product_id"`
	ProductMedia   string           `json:"product_media"`
	ProductGroupID string           `json:"product_group_id"`
	ProductFileTag string           `json:"product_file_tag"`
	ProductName    string           `json:"product_name"`
	IssueName      string           `json:"issues_name"`
	MediaName      string           `json:"media_name"`
	MediaFileTag   string           `json:"media_file_tag"`
	IssueID        string           `json:"issues_id"`
	IssueProdID    string           `json:"issues_prod_id"`
	IssueURL       string           `json:"issues_url"`
	Dvr            string           `json:"dvr"`
	DvrDuration    string           `json:"dvr_duration"`
	HlsURL         string           `json:"hls_url"`
	Program        []ProgramEntries `json:"program"`
}

// ProgramEntries represent list of program entries
type ProgramEntries struct {
	EpgMediaID    string `json:"epg_media_id"`
	EpgProdName   string `json:"epg_prod_name"`
	StartDatetime string `json:"start_datetime"`
	Description   string `json:"description"`
	EpgStart      string `json:"epg_start"`
	EpgEnd        string `json:"epg_end"`
	EpgDuration   string `json:"epg_duration"`
	EpgPosition   string `json:"epg_position"`
	EpgProgress   string `json:"epg_progress"`
	StartTimeUnix string `json:"start_time_unix"`
	EndTimeUnix   string `json:"end_time_unix"`
}

// NewClient creates new Neterra TV client
func NewClient(client *http.Client, username, password, host, port string, tvChannels Channels) *Client {
	tvHashedChannels := make(map[string]*Channel)
	for tcIndex := range tvChannels.Channels {
		ch := tvChannels.Channels[tcIndex]
		key := ch.IssueID
		tvHashedChannels[key] = &ch
	}
	return &Client{username: username, password: password, httpClient: client, loggedIn: false, tvChan: tvHashedChannels, host: host, port: port}
}

func (n *Client) login() (bool, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return false, err
	}

	n.httpClient.Jar = jar

	form := url.Values{}
	form.Set(LoginParamUsername, n.username)
	form.Set(LoginParamPassword, n.password)
	form.Set(LoginParamLogin, LoginValueLogin)
	form.Set(LoginParamLoginType, LoginValueLoginType)

	req, err := http.NewRequest("POST", LoginUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(data), LoginSuccessCriteria), nil
}

func (n *Client) checkLogin() {
	if !n.loggedIn || n.lastLogin.Before(time.Now().Add(-(time.Millisecond * time.Duration(LoginTimeoutMs)))) {
		res, err := n.login()
		if err == nil && res {
			n.loggedIn = true
			n.lastLogin = time.Now()
		}
	}
}

// GetM3U8 fetches the M3U8 list
func (n *Client) GetM3U8() (string, error) {
	n.checkLogin()

	req, err := http.NewRequest("GET", GetLiveUrl, nil)
	if err != nil {
		return "", err
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	m := LiveChannels{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return "", fmt.Errorf("Error unmarshaling %s to LiveChannels %v", string(data), err)
	}

	ret, err2 := n.generatePlaylist(m)

	return ret, err2
}

func (n *Client) generatePlaylist(lschns LiveChannels) (string, error) {
	buf := "#EXTM3U\n"
	for chIdx := range lschns.LiveChannels {
		lch := lschns.LiveChannels[chIdx][0]
		chanID := lch.IssueID
		chanName := lch.IssueName
		tvgID := ""
		tvgName := ""
		group := ""
		logo := ""

		if val, ok := n.tvChan[chanID]; ok {
			chanName = val.Name
			tvgID = val.TvgID
			tvgName = val.TvgName
			group = val.Group
			logo = val.Logo
		}
		encodedChanName := url.QueryEscape(chanName)

		buf = fmt.Sprintf("%s#EXTINF:-1 tvg-id=\"%s\" tvg-name=\"%s\" tvg-logo=\"%s\" group-title=\"%s\",%s\nhttp://%s:%s/playlist.m3u8?ch=%s&name=%s\n",
			buf, tvgID, tvgName, logo, group, chanName, n.host, n.port, chanID, encodedChanName)
	}

	return buf, nil
}

// GetStream returns stream URL
func (n *Client) GetStream(issueID string) (string, error) {
	n.checkLogin()
	form := url.Values{}
	form.Set(GetStreamParamIssue, issueID)
	form.Set(GetStreamParamQuality, GetStreamValueQuality)
	form.Set(GetStreamParamType, GetStreamValueType)

	req, err := http.NewRequest("POST", GetStreamUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	m := playLink{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return "", err
	}

	return m.Link, nil
}
