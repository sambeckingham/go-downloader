package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

func downloadWorker(id int, urls <-chan string, wg *sync.WaitGroup) {
	for url := range urls {
		_, fileName := path.Split(url)
		fmt.Printf("Worker %d is downloading %s\n", id, fileName)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("%s failed!", url)
			return
		}

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("%s failed!", url)
			return
		}
		resp.Body.Close()

		err = os.WriteFile(fileName, bytes, 0644)
		if err != nil {
			fmt.Printf("%s Could not write!", url)
			return
		}

		wg.Done()
	}
}

func getArgs() (string, string) {
	args := os.Args[1:]
	if len(args) != 2 {
		fmt.Println("Usage: go-downloader <URL> <EXTENSION>")
		fmt.Println("Example: go-downloader my-great-site.com .png")
		os.Exit(0)
	}

	return args[0], args[1]
}

func main() {
	pageUrl, extension := getArgs()

	resp, err := http.Get(pageUrl)
	if err != nil {
		log.Fatalf("Unable to query url: %s", pageUrl)
	}

	htmlDoc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal("Unable to parse HTML")
	}

	var fileUrls []string

	var findLinks func(*html.Node)
	findLinks = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && strings.HasSuffix(a.Val, extension) {
					fileUrls = append(fileUrls, a.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findLinks(c)
		}
	}
	findLinks(htmlDoc)

	var wg sync.WaitGroup

	urls := make(chan string, len(fileUrls))

	for i := 1; i <= runtime.NumCPU(); i++ {
		go downloadWorker(i, urls, &wg)
	}

	for _, url := range fileUrls {
		wg.Add(1)
		urls <- url
	}

	close(urls)
	wg.Wait()
}
