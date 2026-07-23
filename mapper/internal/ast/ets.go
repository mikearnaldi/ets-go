package ast

// ETS AST nodes. This file is NOT generated; it is a local additive patch
// (see patches/ in the ets-go repository). ETS nodes are produced only by the
// ETS-aware parser vendored in the ets-go mapper and are consumed by the ETS
// transform before any TypeScript compiler machinery sees the tree.
//
// Kind values are offset from KindCount so upstream kind additions never
// collide with ETS kinds.

const (
	KindETSGenBlock Kind = KindCount + 1 + iota
	KindETSRunExpression
)

// ETSGenBlock is a trailing block call: `<callee> { statements }`, which the
// ETS transform rewrites to `<callee>(function* () { statements })`.
type ETSGenBlock struct {
	ExpressionBase
	CompositeBase
	Expression *Expression // the callee expression
	Block      *Node       // a Block node with the body statements
}

func (f *NodeFactory) NewETSGenBlock(expression *Expression, block *Node) *Node {
	data := &ETSGenBlock{}
	data.Expression = expression
	data.Block = block
	return f.newNode(KindETSGenBlock, data)
}

func (node *ETSGenBlock) ForEachChild(v Visitor) bool {
	return visit(v, node.Expression) ||
		visit(v, node.Block)
}

func (node *ETSGenBlock) VisitEachChild(v *NodeVisitor) *Node {
	return v.Factory.NewETSGenBlock(v.visitNode(node.Expression), v.visitNode(node.Block))
}

func (node *ETSGenBlock) Clone(f NodeFactoryCoercible) *Node {
	return cloneNode(f.AsNodeFactory().NewETSGenBlock(node.Expression, node.Block), node.AsNode(), f.AsNodeFactory().hooks)
}

func IsETSGenBlock(node *Node) bool {
	return node.Kind == KindETSGenBlock
}

func (node *Node) AsETSGenBlock() *ETSGenBlock {
	return node.data.(*ETSGenBlock)
}

// ETSRunExpression is `run <operand>` inside a trailing block body, which the
// ETS transform rewrites to `yield* <operand>`. The `run` keyword itself is
// not stored; its span is [node.Pos()+trivia, +3).
type ETSRunExpression struct {
	ExpressionBase
	CompositeBase
	Expression *Expression // the operand
}

func (f *NodeFactory) NewETSRunExpression(expression *Expression) *Node {
	data := &ETSRunExpression{}
	data.Expression = expression
	return f.newNode(KindETSRunExpression, data)
}

func (node *ETSRunExpression) ForEachChild(v Visitor) bool {
	return visit(v, node.Expression)
}

func (node *ETSRunExpression) VisitEachChild(v *NodeVisitor) *Node {
	return v.Factory.NewETSRunExpression(v.visitNode(node.Expression))
}

func (node *ETSRunExpression) Clone(f NodeFactoryCoercible) *Node {
	return cloneNode(f.AsNodeFactory().NewETSRunExpression(node.Expression), node.AsNode(), f.AsNodeFactory().hooks)
}

func IsETSRunExpression(node *Node) bool {
	return node.Kind == KindETSRunExpression
}

func (node *Node) AsETSRunExpression() *ETSRunExpression {
	return node.data.(*ETSRunExpression)
}
