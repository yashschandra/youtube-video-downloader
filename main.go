package main

import (
	"net/http"
	"io/ioutil"
	"io"
	"regexp"
	"errors"
	"os"
	"encoding/json"
)

var client *http.Client

type Format struct {
	Url string `json:"url"`
	QualityLabel string `json:"qualityLabel"`
}

type StreamingData struct {
	Formats []Format `json:"formats"`
	AdaptiveFormats []Format `json:"adaptiveFormats"`
}

type YTArgs struct {
	PlayerResponse string `json:"player_response"`
}

type YTConfig struct {
	Args YTArgs `json:"args"`
}

func init() {
	client = &http.Client{}
}

func call(url, method string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func getHTML(url string) (string, error) {
	resp, err := call(url, http.MethodGet)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(html), nil
}

func getYTConfig(html string) (YTConfig, error) {
	re, err := regexp.Compile(`;ytplayer\.config\s*=\s*(?P<config>{.+?});ytplayer`)
	if err != nil {
		return YTConfig{}, err
	}
	if !re.MatchString(string(html)) {
		return YTConfig{}, errors.New("ytconfig not found")
	}
	ytconfig := re.FindStringSubmatch(string(html))[1]
	var conf YTConfig
	err = json.Unmarshal([]byte(ytconfig), &conf)
	if err != nil {
		return YTConfig{}, err
	}
	return conf, nil
}

func getStreamingData(playerResponse []byte) (StreamingData, error) {
	var data StreamingData
	var response map[string]interface{}
	err := json.Unmarshal(playerResponse, &response)
	if err != nil {
		return StreamingData{}, err
	}
	b, err := json.Marshal(response["streamingData"])
	if err != nil {
		return StreamingData{}, err
	}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return StreamingData{}, err
	}
	return data, nil
}

func getFormats(data StreamingData) []Format {
	formats := make([]Format, 0, len(data.Formats)+len(data.AdaptiveFormats))
	for _, f := range data.Formats {
		formats = append(formats, f)
	}
	for _, f := range data.AdaptiveFormats {
		formats = append(formats, f)
	}
	return formats
}

func saveVideo(url, filename string) error {
	resp, err := call(url, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func download(url, filename string) error {
	html, err := getHTML(url)
	if err != nil {
		return err
	}
	conf, err := getYTConfig(html)
	if err != nil {
		return err
	}
	data, err := getStreamingData([]byte(conf.Args.PlayerResponse))
	if err != nil {
		return err
	}
	formats := getFormats(data)
	return saveVideo(formats[0].Url, filename)
}

func main() {
	url := os.Args[1]
	filename := os.Args[2]
	err := download(url, filename)
	if err != nil {
		panic(err)
	}
}