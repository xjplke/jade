package jade

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (t *tree) topParse() {
	t.Root = t.newList(t.peek().pos)
	var (
		ext   bool
		token = t.nextNonSpace()
	)
	if token.typ == itemExtends {
		ext = true
		t.Root.append(t.parseSubFile(token.val))//parseSubFile最终也会调用topParse方法，但是在有extends的文件里面，逻辑稍有不同。
												//基类文件的根节点，直接挂载到派生文件tree的root下面，这个没问题。
												//派生文件对应的block内容怎么填充到基类文件的节点上的呢？
												//派生文件parseBlock会直接把block的节点挂载到tree的block这个map上。基类文件的block节点内容是空的
												//这里会建立起一个关联关系。  最后在输出代码的时候再使用这个关系
												//mixin应该类似， 但是include是直接将节点挂载到
												//如何动态load？  不从文件读取，预先创建tree，load 所有的mixin和block(layout)，然后再page里面使用mixin和block
												//用户不需要关心block(layout)，默认使用content，直接在page里面使用已有的mixin来定制自己的页面就可以了！！
												//不使用include，使用mixin代替。
												//因为exend默认会去读文件，所以动态load不能让其读取文件，
												//如果page页面不使用extend方法，只有一个block逻辑，那怎么保证layout的block节点正常处理？
												// 				编写一个pageParse的方法，默认启用extend的标记？其他逻辑不变？？？

		token = t.nextNonSpace()
	}
	for {
		switch token.typ {
		case itemInclude:
			t.Root.append(t.parseInclude(token))
		case itemBlock, itemBlockPrepend, itemBlockAppend:
			if ext {
				t.parseBlock(token)//有extends的情况，说明当前是在处理派生类。 没看明白处理基类文件和派生类文件的时候，相关的节点是怎么关联上去的？
			} else {
				t.Root.append(t.parseBlock(token))//没有extends的情况，说明是在parseSubFile中，在处理基类，通常是layout文件
				//这里因为递归以及树的结构，实际extends也不一定要在最终的layout上
			}
		case itemMixin:
			t.mixin[token.val] = t.parseMixin(token)//这里应该是mixin的声明， mixin的调用在哪儿呢？？？tree的hub函数中处理itemMixinCall
		case itemEOF:
			return
		case itemExtends:
			t.errorf(`Declaration of template inheritance ("extends") should be the first thing in the file. There can only be one extends statement per file.`)
		case itemError:
			t.errorf("%s line: %d\n", token.val, token.line)
		default:
			if ext {
				t.errorf(`Only import, named blocks and mixins can appear at the top level of an extending template`)
			}
			t.Root.append(t.hub(token))
		}
		token = t.nextNonSpace()
	}
}

func (t *tree) hub(token item) (n node) {
	for {
		switch token.typ {
		case itemDiv:
			token.val = "div"
			fallthrough
		case itemTag, itemTagInline, itemTagVoid, itemTagVoidInline:
			return t.parseTag(token)
		case itemText, itemComment, itemHTMLTag:
			return t.newText(token.pos, []byte(token.val), token.typ)
		case itemCode, itemCodeBuffered, itemCodeUnescaped, itemMixinBlock:
			return t.newCode(token.pos, token.val, token.typ)
		case itemIf, itemUnless:
			return t.parseIf(token)
		case itemFor, itemEach, itemWhile:
			return t.parseFor(token)
		case itemCase:
			return t.parseCase(token)
		case itemBlock, itemBlockPrepend, itemBlockAppend:
			return t.parseBlock(token)
		case itemMixinCall:
			return t.parseMixinUse(token)
		case itemInclude:
			return t.parseInclude(token)
		case itemDoctype:
			return t.newDoctype(token.pos, token.val)
		case itemFilter:
			return t.parseFilter(token)
		case itemError:
			t.errorf("Error lex: %s line: %d\n", token.val, token.line)
		default:
			t.errorf(`Error hub(): unexpected token  "%s"  type  "%s"`, token.val, token.typ)
		}
	}
}

func (t *tree) parseFilter(tk item) node {
	var subf, args, text string
Loop:
	for {
		switch token := t.nextNonSpace(); token.typ {
		case itemFilterSubf:
			subf = token.val
		case itemFilterArgs:
			args = strings.Trim(token.val, " \t\r\n")
		case itemFilterText:
			text = strings.Trim(token.val, " \t\r\n")
		default:
			break Loop
		}
	}
	t.backup()
	switch tk.val {
	case "go":
		filterGo(subf, args, text)
	case "markdown", "markdown-it":
		// TODO: filterMarkdown(subf, args, text)
	}
	return t.newList(tk.pos) // for return nothing
}

func filterGo(subf, args, text string) {
	switch subf {
	case "func":
		goFlt.Name = ""
		switch args {
		case "name":
			goFlt.Name = text
		case "arg", "args":
			if goFlt.Args != "" {
				goFlt.Args += ", " + strings.Trim(text, "()")
			} else {
				goFlt.Args = strings.Trim(text, "()")
			}
		default:
			fn := strings.Split(text, "(")
			if len(fn) == 2 {
				goFlt.Name = strings.Trim(fn[0], " \t\n)")
				goFlt.Args = strings.Trim(fn[1], " \t\n)")
			} else {
				log.Fatal(":go:func filter error in " + text)
			}
		}
	case "import":
		goFlt.Import = text
	}
}

func (t *tree) parseTag(tk item) node {
	var (
		deep = tk.depth
		tag  = t.newTag(tk.pos, tk.val, tk.typ)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			if tag.tagType == itemTagVoid || tag.tagType == itemTagVoidInline {
				break Loop
			}
			tag.append(t.hub(token))
		case token.depth == deep:
			switch token.typ {
			case itemClass:
				tag.attr("class", `"`+token.val+`"`, false)
			case itemID:
				tag.attr("id", `"`+token.val+`"`, false)
			case itemAttrStart:
				t.parseAttributes(tag, `"`)
			case itemTagEnd:
				tag.tagType = itemTagVoid
				return tag
			default:
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return tag
}

type pAttr interface {
	attr(string, string, bool)
}

func (t *tree) parseAttributes(tag pAttr, qw string) {
	var (
		aname string
		equal bool
		unesc bool
		stack = make([]string, 0, 4)
	)
	for {
		switch token := t.next(); token.typ {
		case itemAttrSpace:
			// skip
		case itemAttr:
			switch {
			case aname == "":
				aname = token.val
			case aname != "" && !equal:
				tag.attr(aname, qw+aname+qw, unesc)
				aname = token.val
			case aname != "" && equal:
				stack = append(stack, token.val)
			}
		case itemAttrEqual, itemAttrEqualUn:
			if token.typ == itemAttrEqual {
				unesc = false
			} else {
				unesc = true
			}
			equal = true
			switch len_stack := len(stack); {
			case len_stack == 0 && aname != "":
				// skip
			case len_stack > 1 && aname != "":
				tag.attr(aname, strings.Join(stack[:len(stack)-1], " "), unesc)

				aname = stack[len(stack)-1]
				stack = stack[:0]
			case len_stack == 1 && aname == "":
				aname = stack[0]
				stack = stack[:0]
			default:
				t.errorf("unexpected '='")
			}
		case itemAttrComma:
			equal = false
			switch len_stack := len(stack); {
			case len_stack > 0 && aname != "":
				tag.attr(aname, strings.Join(stack, " "), unesc)
				aname = ""
				stack = stack[:0]
			case len_stack == 0 && aname != "":
				tag.attr(aname, qw+aname+qw, unesc)
				aname = ""
			}
		case itemAttrEnd:
			switch len_stack := len(stack); {
			case len_stack > 0 && aname != "":
				tag.attr(aname, strings.Join(stack, " "), unesc)
			case len_stack > 0 && aname == "":
				for _, a := range stack {
					tag.attr(a, a, unesc)
				}
			case len_stack == 0 && aname != "":
				tag.attr(aname, qw+aname+qw, unesc)
			}
			return
		default:
			t.errorf("unexpected %s", token.val)
		}
	}
}

func (t *tree) parseIf(tk item) node {
	var (
		deep = tk.depth
		cond = t.newCond(tk.pos, tk.val, tk.typ)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			cond.append(t.hub(token))
		case token.depth == deep:
			switch token.typ {
			case itemElse:
				ni := t.peek()
				if ni.typ == itemIf {
					token = t.next()
					cond.append(t.newCode(token.pos, token.val, itemElseIf))
				} else {
					cond.append(t.newCode(token.pos, token.val, token.typ))
				}
			default:
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return cond
}

func (t *tree) parseFor(tk item) node {
	var (
		deep = tk.depth
		cond = t.newCond(tk.pos, tk.val, tk.typ)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			cond.append(t.hub(token))
		case token.depth == deep:
			if token.typ == itemElse {
				cond.condType = itemForIfNotContain
				cond.append(t.newCode(token.pos, token.val, itemForElse))
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return cond
}

func (t *tree) parseCase(tk item) node {
	var (
		deep  = tk.depth
		iCase = t.newCond(tk.pos, tk.val, tk.typ)
	)
	for {
		if token := t.nextNonSpace(); token.depth > deep {
			switch token.typ {
			case itemCaseWhen, itemCaseDefault:
				iCase.append(t.newCode(token.pos, token.val, token.typ))
			default:
				iCase.append(t.hub(token))
			}
		} else {
			break
		}
	}
	t.backup()
	return iCase
}

func (t *tree) parseMixin(tk item) *mixinNode {
	var (
		deep  = tk.depth
		mixin = t.newMixin(tk.pos)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			mixin.append(t.hub(token))
		case token.depth == deep:
			if token.typ == itemAttrStart {
				t.parseAttributes(mixin, "")
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return mixin
}

func (t *tree) parseMixinUse(tk item) node {
	tMix, ok := t.mixin[tk.val]
	if !ok {
		t.errorf(`Mixin "%s" must be declared before use.`, tk.val)
	}
	var (
		deep  = tk.depth
		mixin = tMix.CopyMixin()
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			mixin.appendToBlock(t.hub(token))
		case token.depth == deep:
			if token.typ == itemAttrStart {
				t.parseAttributes(mixin, "")
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()

	use := len(mixin.AttrName)
	tpl := len(tMix.AttrName)
	switch {
	case use < tpl:
		i := 0
		diff := tpl - use
		mixin.AttrCode = append(mixin.AttrCode, make([]string, diff)...) // Extend slice
		for index := 0; index < diff; index++ {
			i = tpl - index - 1
			if tMix.AttrName[i] != tMix.AttrCode[i] {
				mixin.AttrCode[i] = tMix.AttrCode[i]
			} else {
				mixin.AttrCode[i] = `""`
			}
		}
		mixin.AttrName = tMix.AttrName
	case use > tpl:
		if tpl <= 0 {
			break
		}
		if strings.HasPrefix(tMix.AttrName[tpl-1], "...") {
			mixin.AttrRest = mixin.AttrCode[tpl-1:]
		}
		mixin.AttrCode = mixin.AttrCode[:tpl]
		mixin.AttrName = tMix.AttrName
	case use == tpl:
		mixin.AttrName = tMix.AttrName
	}
	return mixin
}

func (t *tree) parseBlock(tk item) *blockNode {
	block := t.newList(tk.pos)
	for {//可以用同一段代码填充多个block？--- 应该是block下可以有多个平级的元素节点？
		token := t.nextNonSpace()
		if token.depth > tk.depth {
			block.append(t.hub(token))
		} else {
			break
		}
	}
	t.backup()
	var suf string
	switch tk.typ {
	case itemBlockPrepend:
		suf = "_prepend"
	case itemBlockAppend:
		suf = "_append"
	}
	t.block[tk.val+suf] = block//将block的内容 挂载到tree的block下面， 这个block是一个map。
	return t.newBlock(tk.pos, tk.val, tk.typ)
}

func (t *tree) parseInclude(tk item) *listNode {
	switch ext := filepath.Ext(tk.val); ext {
	case ".jade", ".pug", "":
		return t.parseSubFile(tk.val)
	case ".js", ".css", ".tpl", ".md":
		ln := t.newList(tk.pos)
		ln.append(t.newText(tk.pos, t.read(tk.val), itemText))
		return ln
	default:
		t.errorf(`file extension  "%s"  is not supported`, ext)
		return nil
	}
}

func (t *tree) parseSubFile(path string) *listNode {
	// log.Println("subtemplate: " + path)
	currentTmplDir, _ := filepath.Split(t.Name)
	var incTree = New(currentTmplDir + path)//处理extend的子文件时开了一个新的tree
	incTree.block = t.block//继承了原tree的mixin和block
	incTree.mixin = t.mixin
	_, err := incTree.Parse(t.read(path))//读取文件调用新的tree的Parse，这里相当于进行的递归的调用。。。。
	if err != nil {
		d, _ := os.Getwd()
		t.errorf(`in '%s' subtemplate '%s': parseSubFile() error: %s`, d, path, err)
	}

	return incTree.Root//返回新tree的root
}

func (t *tree) read(path string) []byte {
	currentTmplDir, _ := filepath.Split(t.Name)
	path = currentTmplDir + path

	bb, err := ReadFunc(path)

	if os.IsNotExist(err) {

		if ext := filepath.Ext(path); ext == "" {
			if _, er := os.Stat(path + ".jade"); os.IsNotExist(er) {
				if _, er = os.Stat(path + ".pug"); os.IsNotExist(er) {
					wd, _ := os.Getwd()
					t.errorf("in '%s' subtemplate '%s': file path error: '.jade' or '.pug' file required", wd, path)
				} else {
					ext = ".pug"
				}
			} else {
				ext = ".jade"
			}
			bb, err = ReadFunc(path + ext)
		}
	}
	if err != nil {
		wd, _ := os.Getwd()
		t.errorf(`%s  work dir: %s `, err, wd)
	}
	return bb
}




func (t *tree) pageParse() {
	//t.Root = t.newList(t.peek().pos) //page是直接在原tree上处理block的内容，不能在这里把原tree手上的root覆盖了，实际在load layout的时候已经加的页面的root节点。
	var (
		ext   bool
		token = t.nextNonSpace()
	)
	/*
	if token.typ == itemExtends {
		ext = true
		t.Root.append(t.parseSubFile(token.val))//parseSubFile最终也会调用topParse方法，但是在有extends的文件里面，逻辑稍有不同。
		//基类文件的根节点，直接挂载到派生文件tree的root下面，这个没问题。
		//派生文件对应的block内容怎么填充到基类文件的节点上的呢？
		//派生文件parseBlock会直接把block的节点挂载到tree的block这个map上。基类文件的block节点内容是空的
		//这里会建立起一个关联关系。  最后在输出代码的时候再使用这个关系
		//mixin应该类似， 但是include是直接将节点挂载到
		//如何动态load？  不从文件读取，预先创建tree，load 所有的mixin和block(layout)，然后再page里面使用mixin和block
		//用户不需要关心block(layout)，默认使用content，直接在page里面使用已有的mixin来定制自己的页面就可以了！！
		//不使用include，使用mixin代替。
		//因为exend默认会去读文件，所以动态load不能让其读取文件，
		//如果page页面不使用extend方法，只有一个block逻辑，那怎么保证layout的block节点正常处理？
		// 				编写一个pageParse的方法，默认启用extend的标记？其他逻辑不变？？？

		token = t.nextNonSpace()
	}*/
	ext = true
	for {
		switch token.typ {
		case itemInclude:
			t.Root.append(t.parseInclude(token))
		case itemBlock, itemBlockPrepend, itemBlockAppend:
			if ext {
				t.parseBlock(token)//有extends的情况，说明当前是在处理派生类。 没看明白处理基类文件和派生类文件的时候，相关的节点是怎么关联上去的？
			} else {
				t.Root.append(t.parseBlock(token))//没有extends的情况，说明是在parseSubFile中，在处理基类，通常是layout文件
				//这里因为递归以及树的结构，实际extends也不一定要在最终的layout上
			}
		case itemMixin:
			t.mixin[token.val] = t.parseMixin(token)//这里应该是mixin的声明， mixin的调用在哪儿呢？？？tree的hub函数中处理itemMixinCall
		case itemEOF:
			return
		case itemExtends:
			t.errorf(`Declaration of template inheritance ("extends") should be the first thing in the file. There can only be one extends statement per file.`)
		case itemError:
			t.errorf("%s line: %d\n", token.val, token.line)
		default:
			if ext {
				t.errorf(`Only import, named blocks and mixins can appear at the top level of an extending template`)
			}
			t.Root.append(t.hub(token))
		}
		token = t.nextNonSpace()
	}
}