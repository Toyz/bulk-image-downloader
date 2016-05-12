package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/wushilin/threads"
)

var auth string
var search string
var tCount int
var hxwstore bool
var current_page = 1
var current_download = 1
var page_limit = -1
var minWidth = -1
var minHeight = -1

var base_folder string = "./Output"
var thread_pool *threads.ThreadPool

type WallPaperJSON struct {
	Success    bool `json:"success"`
	Wallpapers []struct {
		ID       string `json:"id"`
		Width    string `json:"width"`
		Height   string `json:"height"`
		FileType string `json:"file_type"`
		FileSize string `json:"file_size"`
		URLImage string `json:"url_image"`
		URLThumb string `json:"url_thumb"`
		URLPage  string `json:"url_page"`
	} `json:"wallpapers"`
	TotalMatch string `json:"total_match"`
}

func main() {
	flag.StringVar(&auth, "auth", "", "API key from AlphaCoders")
	flag.StringVar(&search, "search", "", "Search key for wallpapers to download")
	flag.StringVar(&base_folder, "output", "./Output", "Output folder for images")
	flag.IntVar(&page_limit, "pl", -1, "Max pages to download from")
	flag.BoolVar(&hxwstore, "save-by-size", true, "Store images in different folders by width and height")
	flag.IntVar(&minHeight, "min-height", -1, "Min height of the image to download")
	flag.IntVar(&minWidth, "min-width", -1, "Min width of the image to download")
	flag.IntVar(&tCount, "threads", runtime.NumCPU(), "Number of threads to download on concurrently")

	flag.Parse()

	if _, err := os.Stat("./auth.txt"); err == nil {
		b, _ := ioutil.ReadFile("./auth.txt")
		auth = string(b)
	}

	if auth == "" {
		flag.Usage()
		fmt.Print("Auth (API) is not set use -auth=api")
		return
	}

	if search == "" {
		flag.Usage()
		fmt.Print("Search param not set use -search='keyword'")
		return
	}

	if search == "?" || search == "help" {
		flag.Usage()
		return
	}

	thread_pool = threads.NewPool(tCount, 1000000)

	os.MkdirAll(base_folder+"/"+search, 0777)

	thread_pool.Start()

	GetAllWallpapers(ReadJSONFromAPI(current_page))

	thread_pool.Shutdown()
	thread_pool.Wait()

	fmt.Print("Finished downloading all Wallpapers")
}

func GetAllWallpapers(contents string) {
	var dat WallPaperJSON
	if err := json.Unmarshal([]byte(contents), &dat); err != nil {
		panic(err)
	}

	for i := 0; i < len(dat.Wallpapers); i++ {
		var folder = ""

		if !hxwstore {
			folder = base_folder + "/" + search
		} else {
			folder = base_folder + "/" + search + "/" + dat.Wallpapers[i].Width + "x" + dat.Wallpapers[i].Height
		}

		var wp = dat.Wallpapers[i]
		var ur = wp.URLImage

		if wp.Height < minHeight || wp.Width < minWidth {
			continue
		}

		os.MkdirAll(folder, 0777)

		if _, err := os.Stat(folder + "/" + filepath.Base(ur)); err == nil {
			continue
		}

		current_download++
		fmt.Printf("Downloading Wallpaper: %s (%v/%s)\n", ur, current_download, dat.TotalMatch)
		thread_pool.Submit(func() interface{} {
			downloadFile(folder+"/"+filepath.Base(ur), ur)

			fmt.Printf("Downloaded Wallpaper: %s to %s \n", filepath.Base(ur), folder+"/"+filepath.Base(ur))
			return "done"
		})
	}

	if page_limit > -1 {
		if current_page > page_limit {
			return
		}
	}

	if len(dat.Wallpapers) < 30 {
		return
	}

	time.Sleep(time.Duration(2) * time.Second)
	current_page = current_page + 1
	GetAllWallpapers(ReadJSONFromAPI(current_page))
}

func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func ReadJSONFromAPI(page int) string {
	var p = strconv.Itoa(page)
	var url = "https://wall.alphacoders.com/api2.0/get.php?auth=" + auth + "&method=search&term=" + search + "&page=" + p
	response, err := http.Get(url)

	if err != nil {
		fmt.Printf("%s", err)
		return ReadJSONFromAPI(page)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			return ReadJSONFromAPI(page)
		}
		return string(contents)
	}
}
