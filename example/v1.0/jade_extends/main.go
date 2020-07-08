package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Joker/hpp"
	"github.com/xjplke/jade"
)

func handler(w http.ResponseWriter, r *http.Request) {
	index, err := jade.ParseFile("index.jade")
	if err != nil {
		log.Printf("\nParseFile error: %v", err)
	}
	log.Printf("%s\n\n", hpp.PrPrint(index))

	//

	go_tpl, err := template.New("layout").Parse(index)
	if err != nil {
		log.Printf("\nTemplate parse error: %v", err)
	}

	err = go_tpl.Execute(w, "")
	if err != nil {
		log.Printf("\nExecute error: %v", err)
	}
}

func mainxx() {
	log.Println("open  http://localhost:8080/")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}


func main(){
	bs, err := ioutil.ReadFile("index.jade")
	if err != nil {
		fmt.Print("ReadFile err:",err)
		return
	}

	t := jade.New("index")

	outTpl, err := t.Parse(bs)
	if err != nil {
		fmt.Print("Parse err:",err)
		return
	}
	b := new(bytes.Buffer)
	outTpl.WriteIn(b)

	goTpl, e := template.New("html").Parse(b.String())
	if e!=nil{
		fmt.Println("go Template Parse err",e)
	}

	c := new(bytes.Buffer)
	err = goTpl.Execute(c, &struct{}{})

	fmt.Println(c.String())

}
