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
