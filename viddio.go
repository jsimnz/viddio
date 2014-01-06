package main

import (
	//"encoding/json"
	//"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/jsimnz/go-restful"
	"io"
	//"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	//"time"
)

var (
	rootDir string
	port    int
)

type response struct {
	Status int         `json:"status"`
	Msg    interface{} `json:"msg"`
}

type cropResponse struct {
	Name     string `json:"new_file"`
	Duration int    `json:"duration"`
}

type VideoService struct{}

func (v VideoService) Register() {

	ws := new(restful.WebService)

	ws.
		Path("/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	//Metadata endoint
	ws.Route(ws.GET("/metadata/{filename}").To(v.getMetaData))

	//Video cropping
	ws.Route(ws.POST("/crop/{filename}").To(v.cropVideo))

	restful.Add(ws)
}

func (v VideoService) getMetaData(request *restful.Request, r *restful.Response) {
	filename := request.PathParameter("filename")
	if filename == "" {
		fmt.Println("Could not get filename")
		writeErrorResponse(r, http.StatusBadRequest, "Invalid filename")
		return
	}

	//create the necessary file
	filepath := fmt.Sprintf("%v/%v", rootDir, filename)

	//Execute command and get the Stdout
	cmd := exec.Command("ffprobe", "-print_format", "json", "-show_format", filepath)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	var jsonBuffer bytes.Buffer
	go io.Copy(&jsonBuffer, stdoutPipe)
	cmd.Wait()

	//Write output
	r.Header().Set("Content-Type", "application/json")
	r.Write(jsonBuffer.Bytes())
}

func (v VideoService) cropVideo(request *restful.Request, r *restful.Response) {
	filename := request.PathParameter("filename")
	startTime := request.QueryParameter("start")
	duration := request.QueryParameter("end")

	if filename == "" || startTime == "" || duration == "" {
		fmt.Println("Could not get filename")
		writeErrorResponse(r, http.StatusBadRequest, "Invalid filename")
		return
	}

	//create the full path
	filepath := fmt.Sprintf("%v/%v", rootDir, filename)
	newfile := fmt.Sprintf("63726f70-%v", filename)
	newfilepath := fmt.Sprintf("%v/%v", rootDir, newfile)

	// ffmpeg -ss 00:00:30 -i Dexter.S08E01.HDTV.x264-2HD.mp4 -to 00:00:10 -c copy output-2.mp4
	cmd := exec.Command("ffmpeg", "-ss", startTime, "-i", filepath, "-to", duration, "-c", "copy", newfilepath)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Could not crop file")
		writeErrorResponse(r, http.StatusInternalServerError, "Could not crop file")
		return
	}

	dur, _ := timeToSec(duration)

	writeResponse(r, cropResponse{Name: newfile, Duration: dur})
}

func writeErrorResponse(r *restful.Response, status int, msg string) {
	r.WriteHeader(status)
	r.Header().Set("Status", strconv.Itoa(status))
	r.WriteAsJson(response{Status: 0, Msg: msg})
}

func writeResponse(r *restful.Response, resp interface{}) {
	r.Header().Set("Status", "200")
	r.WriteAsJson(response{Status: 0, Msg: resp})
}

func timeToSec(t string) (int, error) {
	tt := strings.Split(t, ":")
	duration := 0
	if len(tt) == 3 {
		hh, _ := strconv.Atoi(tt[0])
		mm, _ := strconv.Atoi(tt[1])
		ss, _ := strconv.Atoi(tt[2])
		duration = (60 * 60 * hh) + (60 * mm) + (ss)
	} else {
		return 0, errors.New("Invalid time")
	}

	return duration, nil
}

// convert string time: HH:MM:SS to time.Time
/*func convertTime(t string) time.Time {
	tt := strings.Split(t, ":")
	fmt.Println(tt)
	if len(tt) < 3 {
		//error
		panic("Time format not supported")
	}
	hh, _ := strconv.Atoi(tt[0])
	mm, _ := strconv.Atoi(tt[1])
	ss, _ := strconv.Atoi(tt[2])

	return time.Date(1, time.January, 0, hh, mm, ss, 0, time.Local)
}

func Duration*/

func init() {
	var err error
	flag.StringVar(&rootDir, "root", "DEFAULT", "The data directory")
	if rootDir == "DEFAULT" {
		rootDir, err = os.Getwd()
		if err != nil {
			panic(errors.New("Could not get the working directory, please specify one with -root"))
		}
	}
	flag.IntVar(&port, "port", 8080, "The port to run the server on")

	flag.Parse()

	fmt.Println("Root folder: ", rootDir)
	fmt.Println("Port: ", port)
}

func main() {
	videoService := VideoService{}
	videoService.Register()

	fmt.Println("Starting server...")
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
