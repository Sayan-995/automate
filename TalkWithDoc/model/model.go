package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	utils "github.com/Sayan-995/automate/utils"
	"github.com/google/generative-ai-go/genai"
	"github.com/hupe1980/go-huggingface"
	"github.com/joho/godotenv"
	"github.com/philippgille/chromem-go"
	"github.com/tmc/langchaingo/textsplitter"
	"google.golang.org/api/option"
)
func init(){
	godotenv.Load()
}
const END = "END"
var (
	ErrEntryPointNotSet = errors.New("entry point not set")
	ErrNodeNotFound = errors.New("node not found")
	ErrNoOutgoingEdge = errors.New("no outgoing edge found for node")
)

type GraphState struct{
	Text string
	Question string
	GeneratedAnswer string
	Document string
	Model  *huggingface.InferenceClient
	Context context.Context
}
func (g *GraphState)ValidateAnswer(question ,context,answer string)(bool,error){
	client,err:=genai.NewClient(g.Context,option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if(err!=nil){
		return false,err;
	}
	defer client.Close()
	model := client.GenerativeModel("gemini-1.5-flash")
	res,err:=model.GenerateContent(g.Context,genai.Text(fmt.Sprintf(utils.ValidationPromptTemplate,context,question,answer)))
	if err!= nil {
		return false,err;
	}
	for _,cand:=range(res.Candidates){
		if(cand.Content!=nil){
			for _,part:=range(cand.Content.Parts){
				ans:=fmt.Sprintf("%s",part)
				if(strings.Contains(strings.ToLower(ans),"yes")){
					return true,nil
				}else if(strings.Contains(strings.ToLower(ans),"no")){
					return false,nil
				}
			}
		}
	}
	return false,fmt.Errorf("error while validating")
}
func (g *GraphState)GenerateAnswer()error{
	res,err:=g.Model.QuestionAnswering(g.Context,&huggingface.QuestionAnsweringRequest{
		Inputs: huggingface.QuestionAnsweringInputs{
			Question: g.Question,
			Context: g.Document,
		},
	})
	if(err!=nil){
		return err
	}
	var ans string
	Ok,err:=g.ValidateAnswer(g.Question,g.Document,res.Answer);
	if err!=nil{
		return err
	}
	if(Ok){
		ans=res.Answer
	}else{
		ans="The question is irrelevent to the document"
	}
	g.GeneratedAnswer=ans
	return nil
}
func (g *GraphState)CreateModel()error{
	model := huggingface.NewInferenceClient(os.Getenv("HUGGINGFACEHUB_API_TOKEN"))
	g.Model=model
	return nil
}
func (g *GraphState)RetriveDocs()error{
	c,err:=g.BuildVectorStore(g.Text)
	if(err!=nil){
		return err
	}
	res,err:=c.Query(g.Context,g.Question,1,nil,nil)
	if err!=nil{
		return err
	}
	g.Document=res[0].Content
	return nil
}
func (g *GraphState) BuildVectorStore(Text string)(*chromem.Collection,error){
	ctx:=g.Context
	splitter:=textsplitter.NewMarkdownTextSplitter(
		textsplitter.WithChunkSize(200),
		textsplitter.WithChunkOverlap(40),
		textsplitter.WithCodeBlocks(true),
		textsplitter.WithHeadingHierarchy(true),
	)
	splittedText,err:=splitter.SplitText(Text)
	if(err!=nil){
		return nil,err
	}
	db:=chromem.NewDB()
	var doc []chromem.Document
	for i:=0;i<len(splittedText);i++{
		doc = append(doc, chromem.Document{
			ID: strconv.Itoa(i+1),
			Content: splittedText[i],
		})
	}
	c, err := db.CreateCollection("knowledge-base", nil,chromem.NewEmbeddingFuncJina(os.Getenv("JINA_API_KEY"),"jina-clip-v2") )
	if err != nil {
		panic(err)
	}
	err=c.AddDocuments(ctx,doc,runtime.NumCPU())
	if err!=nil{
		return nil,err
	}
	return c,nil
}
type Node struct{
	Name string
	Function func()error
}
type Edge struct{
	From string
	To string
}
type MessageGraph struct{
	Nodes map[string]Node
	Edges map[string][]string
	EntryPoint string
}
func NewMessageGraph()*MessageGraph{
	return &MessageGraph{
		Nodes: make(map[string]Node),
		Edges: make(map[string][]string),
	}
}
func (g *MessageGraph)AddNode(Name string,Fn func()error){
	g.Nodes[Name]=Node{
		Name: Name,
		Function: Fn,
	}
}
func (g *MessageGraph)AddEdge(To ,From string){
	g.Edges[To] = append(g.Edges[To], From)
}
func (g *MessageGraph) SetEntryPoint(name string) {
	g.EntryPoint = name
}
type Runnable struct {
	graph *MessageGraph
}
func (g *MessageGraph) Compile() (*Runnable, error) {
	if g.EntryPoint == "" {
		return nil, ErrEntryPointNotSet
	}
	fmt.Println(g)
	return &Runnable{
		graph: g,
	}, nil
}
func (r *Runnable)Invoke(g *GraphState)(string,error){
	var st []string 
	st=append(st,r.graph.EntryPoint)
	for(len(st)>0){
		CurrentNode:=st[len(st)-1]
		st=st[:len(st)-1] 
		if CurrentNode==END{
			continue
		}
		node,ok:=r.graph.Nodes[CurrentNode]
		if !ok{
			return "",fmt.Errorf("error while traversing graph")
		}
		err:=node.Function()
		if err!=nil{
			return "",err
		}
		found:=false
		for _,Children:=range(r.graph.Edges[CurrentNode]){
			st=append(st, Children)
			found=true
		}
		if !found{
			return "",fmt.Errorf("%s %s",ErrNoOutgoingEdge,CurrentNode)
		}
	}
	fmt.Println(g.GeneratedAnswer)
	return g.GeneratedAnswer,nil
}

func CreateRunable(g *GraphState)(*Runnable,error){
	graph:=NewMessageGraph()
	graph.AddNode("retrive_docs",g.RetriveDocs)
	graph.AddNode("create_model",g.CreateModel)
	graph.AddNode("generate_answer",g.GenerateAnswer)
	graph.SetEntryPoint("retrive_docs")
	graph.AddEdge("retrive_docs","create_model")
	graph.AddEdge("create_model","generate_answer")
	graph.AddEdge("generate_answer",END)
	r,err:=graph.Compile()
	if(err!=nil){
		return nil,err
	}
	return r,nil
}