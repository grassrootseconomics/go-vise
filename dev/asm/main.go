package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"

	"git.defalsify.org/vise.git/asm"
)

type arg struct {
	One   *string `(@Sym | @NumFirst)`
	Two   *string `((@Sym | @NumFirst) Whitespace?)?`
	Three *string `((@Sym | @NumFirst) Whitespace?)?`
}

type instruction struct {
	OpCode  string `@Ident`
	OpArg   arg    `(Whitespace @@)?`
	Comment string `Comment? EOL`
}

type asmAsm struct {
	Instructions []*instruction `@@*`
}

type processor struct {
	*asm.FlagParser
}

func newProcessor(fp string) (*processor, error) {
	o := &processor{
		asm.NewFlagParser(),
	}
	_, err := o.Load(fp)
	return o, err
}

func (p *processor) processFlag(s []string, one *string, two *string) ([]string, error) {
	_, err := strconv.Atoi(*one)
	if err != nil {
		r, err := p.GetAsString(*one)
		if err != nil {
			return nil, err
		}
		log.Printf("translated flag %s to %s", *one, r)
		s = append(s, r)
	} else {
		s = append(s, *one)
	}
	return append(s, *two), nil
}

func (p *processor) pass(s []string, a arg) []string {
	for _, r := range []*string{a.One, a.Two, a.Three} {
		if r == nil {
			break
		}
		s = append(s, *r)
	}
	return s
}

func (pp *processor) run(b []byte) ([]byte, error) {
	asmLexer := lexer.MustSimple([]lexer.SimpleRule{
		{"Comment", `(?:#)[^\n]*`},
		{"Ident", `^[A-Z]+`},
		{"NumFirst", `[0-9][a-zA-Z0-9]*`},
		{"Sym", `[a-zA-Z_\*\.\^\<\>][a-zA-Z0-9_]*`},
		{"Whitespace", `[ \t]+`},
		{"EOL", `[\n\r]+`},
		{"Quote", `["']`},
	})
	asmParser := participle.MustBuild[asmAsm](
		participle.Lexer(asmLexer),
		participle.Elide("Comment", "Whitespace"),
	)
	ast, err := asmParser.ParseString("preprocessor", string(b))
	if err != nil {
		return nil, err
	}

	b = []byte{}
	for _, v := range ast.Instructions {
		s := []string{v.OpCode}
		if v.OpArg.One != nil {
			switch v.OpCode {
			case "CATCH":
				s = append(s, *v.OpArg.One)
				s, err = pp.processFlag(s, v.OpArg.Two, v.OpArg.Three)
				if err != nil {
					return nil, err
				}
			case "CROAK":
				s, err = pp.processFlag(s, v.OpArg.One, v.OpArg.Two)
				if err != nil {
					return nil, err
				}
			default:
				s = pp.pass(s, v.OpArg)
			}
		}
		b = append(b, []byte(strings.Join(s, " "))...)
		b = append(b, 0x0a)
	}

	return b, nil
}

func main() {
	var ppfp string
	flag.StringVar(&ppfp, "f", "", "preprocessor data to load")
	flag.Parse()
	if len(flag.Args()) < 1 {
		os.Exit(1)
	}
	fp := flag.Arg(0)
	v, err := ioutil.ReadFile(fp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}

	if len(ppfp) > 0 {
		pp, err := newProcessor(ppfp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "preprocessor load error: %v\n", err)
			os.Exit(1)
		}

		v, err = pp.run(v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "preprocess error: %v\n", err)
			os.Exit(1)
		}
	}
	log.Printf("preprocessor done")

	n, err := asm.Parse(string(v), os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	log.Printf("parsed total %v bytes", n)
}
