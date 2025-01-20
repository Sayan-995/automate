package server

import (
	"log"
	"net/http"

	upload "github.com/Sayan-995/automate/upload"
	utils "github.com/Sayan-995/automate/utils"
	"github.com/gorilla/mux"
)
type server struct{
	listenAddress string
}
func CreateServer(listenAddress string)*server{
	return &server{
		listenAddress: listenAddress,
	}
}
func hello(w http.ResponseWriter,r *http.Request)error{
	return utils.WriteJSON(w,http.StatusAccepted,"hellow")
}
func (s *server)RunServer(){
	myRouter:=mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/upload",utils.GenerateHandleFunc(upload.HandleUploadPdf))
	myRouter.HandleFunc("/",utils.GenerateHandleFunc(hello))
	log.Fatal(http.ListenAndServe(s.listenAddress,myRouter))	
}