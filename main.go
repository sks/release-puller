package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

type Release struct {
	Assets []asset `json:"assets"`
}

type asset struct {
	BrowserDownloadUrl string `json:"browser_download_url"`
	Name               string
}

func main() {
	var (
		repo                 string
		matchingRegexp       string
		includeDraftReleases bool
		downlaodTo           string
		err                  error
	)
	flag.StringVar(&repo, "repo", "", "repo to pull the latest binary from")
	flag.StringVar(&matchingRegexp, "regex", ".*", "file that matches regexp (Default to all)")
	flag.StringVar(&downlaodTo, "download-to", "", "Download the binary to this path")
	flag.BoolVar(&includeDraftReleases, "include-draft", false, "Include draft releases (default to false)")
	flag.Parse()

	if repo == "" {
		flag.PrintDefaults()
		log.Fatal("Repo must be a passed in")
	}
	if downlaodTo == "" {
		downlaodTo, err = os.Getwd()
		if err != nil {
			log.Fatal("Could not get the current working directory, please try passing in --download-to flag", err)
		}
	}

	regex, err := regexp.Compile(matchingRegexp)
	if err != nil {
		log.Fatalf("Could not create a valid regular expression with %s, Please give an appropriate one", matchingRegexp)
	}

	log.Printf("Checking for the latest available releases from %s", repo)

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	release := Release{}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Fatalf("Could not decode the response from request: %s, %s", url, err)
	}

	if len(release.Assets) == 0 {
		log.Fatalf("Could not find any assets associated with the repo %s, request: %s", repo, url)
	}

	for _, asset := range release.Assets {
		if regex.MatchString(asset.Name) {
			log.Printf("Downloading the asset %s to the location %s", asset.Name, downlaodTo)
			err = downloadAsset(asset, downlaodTo)
			if err != nil {
				log.Fatal("Error saving the file %s to %s", asset.Name, downlaodTo)
			}
		} else {
			log.Printf("Skipping the asset %s", asset.Name)
		}
	}
}

func downloadAsset(asset asset, downloadTo string) error {
	output, err := os.Create(filepath.Join(downloadTo, asset.Name))
	if err != nil {
		return err
	}
	defer output.Close()
	response, err := http.Get(asset.BrowserDownloadUrl)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	_, err = io.Copy(output, response.Body)
	if err != nil {
		return err
	}
	log.Printf("Downloaded the file %s to %s", asset.Name, downloadTo)
	return nil
}
