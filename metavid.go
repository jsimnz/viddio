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
)

var (
	rootDir string
	port    int

	//Arguments
	arg0 string = "-print_format"
	arg1 string = "json"
	arg2 string = "-show_format"
	arg3 string = ">"
)

type response struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

type MetaDataService struct{}

func (m MetaDataService) Register() {

	ws := new(restful.WebService)

	ws.
		Path("/metadata").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/{filename}").To(m.getMetaData))

	restful.Add(ws)
}

func (m MetaDataService) getMetaData(request *restful.Request, r *restful.Response) {
	filename := request.PathParameter("filename")
	if filename == "" {
		fmt.Println("Could not get filename")
		writeErrorResponse(r, http.StatusBadRequest, "Invalid filename")
		return
	}

	//create the necessary file
	filepath := fmt.Sprintf("%v/%v", rootDir, filename)
	//tempJSON := fmt.Sprintf("/tmp/%v.json", filename)
	//cmdParams := fmt.Sprintf("-print_format json -show_format %v", filepath)

	cmd := exec.Command("ffprobe", arg0, arg1, arg2, filepath)

	// open the out file for writing
	/*outfile, err := os.Create("./out.txt")
	if err != nil {
		panic(err)
	}
	defer outfile.Close()*/

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	//writer := bufio.NewWriter(outfile)
	//defer writer.Flush()

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

func writeErrorResponse(r *restful.Response, status int, msg string) {
	r.WriteHeader(status)
	r.Header().Set("Status", strconv.Itoa(status))
	r.WriteAsJson(response{Status: 0, Msg: msg})
}

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
	metaDataService := MetaDataService{}
	metaDataService.Register()

	fmt.Println("Starting server...")
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
