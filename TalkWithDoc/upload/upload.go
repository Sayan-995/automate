package upload

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	twd "github.com/Sayan-995/automate/Twd"
	mod "github.com/Sayan-995/automate/model"
	"github.com/Sayan-995/automate/utils"
	u "github.com/Sayan-995/automate/utils"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

const(
	maxSize=10<<20
)
func HandleUploadPdf(w http.ResponseWriter,r *http.Request)error{
	if err:=r.ParseMultipartForm(maxSize);err!=nil{
		return err
	}
	file,header,err:=r.FormFile("pdf")
	if err!=nil{
		return u.WriteJSON(w,http.StatusBadRequest,"Error getting the files")
	}
	defer file.Close()
	if filepath.Ext(header.Filename)!=".pdf"{
		return u.WriteJSON(w,http.StatusBadRequest,"Only pdf files are allowed")
	}
	pdfReader,err:=model.NewPdfReader(file)
	if err!=nil{
		return err
	}
	numPages,err:=pdfReader.GetNumPages()
	if err!=nil{
		return err
	}
	var totText string
	for i:=1;i<=numPages;i++{
		page,err:=pdfReader.GetPage(i)
		if err!=nil{
			return err
		}
		textExtractor,err:=extractor.New(page)
		if err!=nil{
			return err
		}
		text,err:=textExtractor.ExtractText()
		if err!=nil{
			return err
		}
		totText+=text
	}
	twd.AddText(totText)
	return u.WriteJSON(w, http.StatusOK, "PDF uploaded successfully")
}
func HandleUploadQuestion(w http.ResponseWriter, r *http.Request)error{
	var question string
	err:=json.NewDecoder(r.Body).Decode(&question)
	if(err!=nil){
		return err
	}
	twd.AddQuestion(question)
	res,err:=mod.CreateRunable(twd.G)
	if(err!=nil){
		return err
	}
	ans,err:=res.Invoke(twd.G)
	if(err!=nil){
		return err
	}
	return utils.WriteJSON(w,http.StatusOK,ans)
}