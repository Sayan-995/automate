package upload

import (
	"fmt"
	"net/http"
	"path/filepath"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
	u "github.com/Sayan-995/automate/utils"
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
		fmt.Printf("%v",text);
	}
	return nil
}