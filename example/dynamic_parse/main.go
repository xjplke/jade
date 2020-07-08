package main

import (
	"bytes"
	"fmt"
	"github.com/xjplke/jade"
	"html/template"
	"io/ioutil"
)

func main_old(){
	//需要再index.jade中打开extend注释,并且将 head /foot的mix修改为include
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

	funcMap := template.FuncMap{
		"bold": func(content string) (template.HTML, error) {
			return template.HTML("<b>" + content + "</b>"), nil
		},
	}

	goTpl, e := template.New("html").Funcs(funcMap).Parse(b.String())
	if e!=nil{
		fmt.Println("go Template Parse err",e)
	}


	c := new(bytes.Buffer)
	err = goTpl.Execute(c, &struct{}{})

	fmt.Println(c.String())

}


func main(){
	//先装载mixin和 layout
	t := jade.New("index")


	for _, f := range []string{"head.jade","foot.jade","layout.jade"} {
		bs, err := ioutil.ReadFile(f)
		if err != nil {
			fmt.Println("ReadFile ",f," err:",err)
			return
		}

		t.Parse(bs)
		if err != nil {
			fmt.Print("Parse file ", f,"  err:",err)
			return
		}
	}

	//调用ParsePage
	bs, err := ioutil.ReadFile("index.jade")
	if err != nil {
		fmt.Println("ReadFile index.jade err:",err)
		return
	}
	t.ParsePage(bs)


	b := new(bytes.Buffer)
	t.WriteIn(b)

	funcMap := template.FuncMap{
		"bold": func(content string) (template.HTML, error) {
			return template.HTML("<b>" + content + "</b>"), nil
		},
	}

	goTpl, e := template.New("html").Funcs(funcMap).Parse(b.String())
	if e!=nil{
		fmt.Println("go Template Parse err",e)
	}


	c := new(bytes.Buffer)
	err = goTpl.Execute(c, &struct{}{})

	fmt.Println(c.String())




	//调用ParsePage
	bs, err = ioutil.ReadFile("page.jade")
	if err != nil {
		fmt.Println("ReadFile page.jade err:",err)
		return
	}
	t.ParsePage(bs)


	b = new(bytes.Buffer)
	t.WriteIn(b)

	goTpl2, e := template.New("html").Funcs(funcMap).Parse(b.String())
	if e!=nil{
		fmt.Println("go Template Parse err",e)
	}


	c = new(bytes.Buffer)
	err = goTpl2.Execute(c, &struct{}{})

	fmt.Println(c.String())


}