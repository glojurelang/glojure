package ast

import "fmt"

// Transform rewrites an AST in post-order. Children are transformed before
// their parent, and replacements are installed in the parent before fn is
// called. Transform mutates the freshly analyzed tree in place; fn may return
// either the supplied node or a replacement.
func Transform(root *Node, fn func(*Node) (*Node, error)) (*Node, error) {
	if root == nil {
		return nil, nil
	}
	if fn == nil {
		return nil, fmt.Errorf("ast transform function is nil")
	}
	if err := transformChildren(root, fn); err != nil {
		return nil, err
	}
	result, err := fn(root)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("ast transform returned a nil node")
	}
	return result, nil
}

func transformNode(node **Node, fn func(*Node) (*Node, error)) error {
	if *node == nil {
		return nil
	}
	transformed, err := Transform(*node, fn)
	if err != nil {
		return err
	}
	*node = transformed
	return nil
}

func transformNodes(nodes []*Node, fn func(*Node) (*Node, error)) error {
	for i := range nodes {
		if err := transformNode(&nodes[i], fn); err != nil {
			return err
		}
	}
	return nil
}

func transformChildren(node *Node, fn func(*Node) (*Node, error)) error {
	switch sub := node.Sub.(type) {
	case *ConstNode:
		return transformNode(&sub.Meta, fn)
	case *DefNode:
		if err := transformNode(&sub.Meta, fn); err != nil {
			return err
		}
		return transformNode(&sub.Init, fn)
	case *SetBangNode:
		if err := transformNode(&sub.Target, fn); err != nil {
			return err
		}
		return transformNode(&sub.Val, fn)
	case *WithMetaNode:
		if err := transformNode(&sub.Expr, fn); err != nil {
			return err
		}
		return transformNode(&sub.Meta, fn)
	case *FnNode:
		if err := transformNodes(sub.Methods, fn); err != nil {
			return err
		}
		return transformNode(&sub.Local, fn)
	case *FnMethodNode:
		if err := transformNodes(sub.Params, fn); err != nil {
			return err
		}
		return transformNode(&sub.Body, fn)
	case *MapNode:
		if err := transformNodes(sub.Keys, fn); err != nil {
			return err
		}
		return transformNodes(sub.Vals, fn)
	case *VectorNode:
		return transformNodes(sub.Items, fn)
	case *SetNode:
		return transformNodes(sub.Items, fn)
	case *DoNode:
		if err := transformNodes(sub.Statements, fn); err != nil {
			return err
		}
		return transformNode(&sub.Ret, fn)
	case *LetNode:
		if err := transformNodes(sub.Bindings, fn); err != nil {
			return err
		}
		return transformNode(&sub.Body, fn)
	case *BindingNode:
		return transformNode(&sub.Init, fn)
	case *InvokeNode:
		if err := transformNode(&sub.Fn, fn); err != nil {
			return err
		}
		return transformNodes(sub.Args, fn)
	case *IfNode:
		if err := transformNode(&sub.Test, fn); err != nil {
			return err
		}
		if err := transformNode(&sub.Then, fn); err != nil {
			return err
		}
		return transformNode(&sub.Else, fn)
	case *NewNode:
		if err := transformNode(&sub.Class, fn); err != nil {
			return err
		}
		return transformNodes(sub.Args, fn)
	case *QuoteNode:
		return transformNode(&sub.Expr, fn)
	case *TryNode:
		if err := transformNode(&sub.Body, fn); err != nil {
			return err
		}
		if err := transformNodes(sub.Catches, fn); err != nil {
			return err
		}
		return transformNode(&sub.Finally, fn)
	case *CatchNode:
		if err := transformNode(&sub.Class, fn); err != nil {
			return err
		}
		if err := transformNode(&sub.Local, fn); err != nil {
			return err
		}
		return transformNode(&sub.Body, fn)
	case *ThrowNode:
		return transformNode(&sub.Exception, fn)
	case *HostCallNode:
		if err := transformNode(&sub.Target, fn); err != nil {
			return err
		}
		return transformNodes(sub.Args, fn)
	case *HostFieldNode:
		return transformNode(&sub.Target, fn)
	case *HostInteropNode:
		return transformNode(&sub.Target, fn)
	case *LetFnNode:
		if err := transformNodes(sub.Bindings, fn); err != nil {
			return err
		}
		return transformNode(&sub.Body, fn)
	case *RecurNode:
		return transformNodes(sub.Exprs, fn)
	case *GoNode:
		return transformNode(&sub.Invoke, fn)
	case *CaseNode:
		if err := transformNode(&sub.Test, fn); err != nil {
			return err
		}
		if err := transformNode(&sub.Default, fn); err != nil {
			return err
		}
		for i := range sub.Entries {
			if err := transformNode(&sub.Entries[i].TestConstant, fn); err != nil {
				return err
			}
			if err := transformNode(&sub.Entries[i].ResultExpr, fn); err != nil {
				return err
			}
		}
	case nil, *LocalNode, *VarNode, *GoBuiltinNode, *MaybeHostFormNode,
		*MaybeClassNode, *TheVarNode:
		return nil
	default:
		return fmt.Errorf("unsupported AST payload %T", node.Sub)
	}
	return nil
}
