package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type extractJob struct {
	id       string
	title    string
	location string
}

func Scrape(term string) {
	var baseURL string = "https://stackoverflow.com/jobs?q=" + term
	var jobs []extractJob
	totalPages := getPages(baseURL)
	c := make(chan []extractJob)

	// stackoverflow의 처음 전체 페이지를 반복
	for i := 0; i < totalPages; i++ {
		go getPage(i, baseURL, c)
	}

	for i := 0; i < totalPages; i++ {
		extractedJobs := <-c
		// getPage는 slice를 return하므로 jobs에 contents만 추가하도록 ...을 사용
		jobs = append(jobs, extractedJobs...)
	}

	writeJobs(jobs)
	fmt.Println("Done, extracted", len(jobs))
}

func writeJobs(jobs []extractJob) {
	// jobs.csv 파일 생성
	file, err := os.Create("jobs.csv")
	checkErr(err)

	// w에 file의 데이터를 입력
	w := csv.NewWriter(file)
	// csv.NewWriter 함수가 끝나면 jobs.csv 파일에 데이터를 저장
	defer w.Flush()

	headers := []string{"LINK", "TITLE", "LOCATION"}
	wErr := w.Write(headers)
	checkErr(wErr)

	for _, job := range jobs {
		jobSlice := []string{"https://stackoverflow.com/jobs/" + job.id, job.title, job.location}
		go w.Write(jobSlice)
	}
}

func getPage(page int, baseURL string, mainC chan<- []extractJob) {
	var jobs []extractJob
	c := make(chan extractJob)
	pageURL := baseURL + "&pg=" + strconv.Itoa(page)
	fmt.Println("Request URL", pageURL)
	res, err := http.Get(pageURL)
	checkErr(err)
	checkStatusCode(res)

	// defer는 근처에 있는 function이 return할 때까지 해당 function 또는 method의 실행을 delay 시킨다
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	searchJobs := doc.Find(".-job")

	searchJobs.Each(func(i int, job *goquery.Selection) {
		// goroutine 생성
		go extractJobData(job, c)
	})

	for i := 0; i < searchJobs.Length(); i++ {
		// extractJobData에서 보낸 메시지를 받음
		job := <-c
		jobs = append(jobs, job)
	}

	mainC <- jobs
}

func extractJobData(job *goquery.Selection, c chan<- extractJob) {
	jobId, _ := job.Attr("data-jobid")
	title := CleanString(job.Find("h2>a").Text())
	location := CleanString(job.Find("h3 .fc-black-500").Text())
	// channel에 extractJob을 전달
	c <- extractJob{id: jobId, title: title, location: location}
}

func getPages(baseURL string) int {
	pages := 0
	res, err := http.Get(baseURL)
	checkErr(err)
	checkStatusCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".s-pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a.s-pagination--item").Length()
	})

	return pages
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkStatusCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with status code:", res.StatusCode)
	}
}

func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}
