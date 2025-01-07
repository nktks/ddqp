package ddqp

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type MetricQuery struct {
	Pos           lexer.Position
	QueryFunction []*QueryFunction `@@*`
	Query         []*Query         `@@*`
}

func (mq *MetricQuery) String() string {
	if len(mq.Query) > 0 {
		return mq.Query[0].String()
	}
	return mq.QueryFunction[0].String()

}

type QueryFunction struct {
	Name string `@Ident?`
	Arg  *Query `"("? @@ ")"?`
}

func (q *QueryFunction) String() string {
	return fmt.Sprintf("%s(%s)", q.Name, q.Arg.String())
}

type OpIdent struct {
	Operator Operator `@("+" | "-" | "/" | "*")`
	Ident    string   `@Ident`
}

func (o *OpIdent) String() string {
	return fmt.Sprintf(" %s %s", o.Operator, o.Ident)
}

type Query struct {
	Pos lexer.Position

	Aggregator                string        `parser:"@Ident"`
	SpaceAggregationCondition string        `parser:"( '(' @SpaceAggregatorCondition ')' )?"`
	Separator                 string        `parser:"':'"`
	MetricName                string        `parser:"@Ident( @'.' @Ident)*"`
	Filters                   *MetricFilter `parser:"'{' @@ '}'"`
	OpIdent                   *OpIdent      `parser:"( @@)?"`
	By                        string        `parser:"Ident?"`
	Grouping                  []string      `parser:"'{'? ( @Ident ( ',' @Ident )* )? '}'?"`
	Function                  []*Function   `parser:"( @@ ( '.' @@ )* )?"`
}

func (q *Query) String() string {
	base := q.Aggregator

	if q.SpaceAggregationCondition != "" {
		base = fmt.Sprintf("%s(%s)", base, q.SpaceAggregationCondition)
	}

	base = fmt.Sprintf("%s:%s{%s}", base, q.MetricName, q.Filters.String())

	if q.OpIdent != nil {
		base = fmt.Sprintf("%s%s", base, q.OpIdent.String())
	}
	if len(q.Grouping) > 0 {
		base = fmt.Sprintf("%s by {%s}", base, strings.Join(q.Grouping, ","))
	}

	if len(q.Function) > 0 {
		funcs := []string{}
		for _, v := range q.Function {
			funcs = append(funcs, v.String())
		}
		return fmt.Sprintf("%s.%s", base, strings.Join(funcs, "."))
	}

	return base
}

type Function struct {
	Name string   `"." @Ident`
	Args []*Value `"(" ( @@ ( "," @@ )* )? ")"`
}

func (f *Function) String() string {
	args := []string{}
	for _, v := range f.Args {
		args = append(args, v.String())
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(args, ","))
}

type Bool bool

func (b *Bool) Capture(v []string) error { *b = v[0] == "true"; return nil }
func (b *Bool) String() string           { return fmt.Sprintf("%v", *b) }

// NewMetricQueryParser returns a Parser which is capable of interpretting
// a metric query.
func NewMetricQueryParser() *MetricQueryParser {
	mqp := &MetricQueryParser{
		parser: participle.MustBuild[MetricQuery](
			participle.Lexer(lex),
			participle.Unquote("String"),
		),
	}

	return mqp
}

// MetricQueryParser is parser returned when calling NewMetricQueryParser.
type MetricQueryParser struct {
	parser *participle.Parser[MetricQuery]
}

// Parse sanitizes the query string and returns the AST and any error.
func (mqp *MetricQueryParser) Parse(query string) (*MetricQuery, error) {
	// the parser doesn't handle queries that are split up across multiple lines
	sanitized := strings.ReplaceAll(query, "\n", "")
	// return the raw parsed outpu
	return mqp.parser.ParseString("", sanitized)
}
