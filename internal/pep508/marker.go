package pep508

import (
	"fmt"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/pep440"
)

type MarkerEnvironment struct {
	OSName                string
	SysPlatform           string
	PlatformMachine       string
	PlatformRelease       string
	PlatformSystem        string
	PlatformVersion       string
	PythonVersion         string
	PythonFullVersion     string
	ImplementationName    string
	ImplementationVersion string
	Extra                 string
}

type MarkerTree interface {
	Evaluate(env MarkerEnvironment) bool
	String() string
}

type MarkerAnd struct {
	Left  MarkerTree
	Right MarkerTree
}

func (m *MarkerAnd) Evaluate(env MarkerEnvironment) bool {
	return m.Left.Evaluate(env) && m.Right.Evaluate(env)
}

func (m *MarkerAnd) String() string {
	return m.Left.String() + " and " + m.Right.String()
}

type MarkerOr struct {
	Left  MarkerTree
	Right MarkerTree
}

func (m *MarkerOr) Evaluate(env MarkerEnvironment) bool {
	return m.Left.Evaluate(env) || m.Right.Evaluate(env)
}

func (m *MarkerOr) String() string {
	return m.Left.String() + " or " + m.Right.String()
}

type MarkerOp int

const (
	MarkerOpEqual MarkerOp = iota
	MarkerOpNotEqual
	MarkerOpLess
	MarkerOpLessEqual
	MarkerOpGreater
	MarkerOpGreaterEqual
	MarkerOpIn
	MarkerOpNotIn
)

func (op MarkerOp) String() string {
	switch op {
	case MarkerOpEqual:
		return "=="
	case MarkerOpNotEqual:
		return "!="
	case MarkerOpLess:
		return "<"
	case MarkerOpLessEqual:
		return "<="
	case MarkerOpGreater:
		return ">"
	case MarkerOpGreaterEqual:
		return ">="
	case MarkerOpIn:
		return "in"
	case MarkerOpNotIn:
		return "not in"
	default:
		return "?"
	}
}

type MarkerExpr struct {
	Variable string
	Op       MarkerOp
	Value    string
}

func (m *MarkerExpr) Evaluate(env MarkerEnvironment) bool {
	varValue := lookupMarkerVar(m.Variable, env)
	return evaluateOp(varValue, m.Op, m.Value)
}

func (m *MarkerExpr) String() string {
	return fmt.Sprintf("%s %s %q", m.Variable, m.Op, m.Value)
}

func lookupMarkerVar(name string, env MarkerEnvironment) string {
	switch name {
	case "os_name", "os.name":
		return env.OSName
	case "sys_platform", "sys.platform":
		return env.SysPlatform
	case "platform_machine", "platform.machine":
		return env.PlatformMachine
	case "platform_release", "platform.release":
		return env.PlatformRelease
	case "platform_system", "platform.system":
		return env.PlatformSystem
	case "platform_version", "platform.version":
		return env.PlatformVersion
	case "python_version", "python.version":
		return env.PythonVersion
	case "python_full_version":
		return env.PythonFullVersion
	case "implementation_name":
		return env.ImplementationName
	case "implementation_version":
		return env.ImplementationVersion
	case "extra":
		return env.Extra
	default:
		return ""
	}
}

func evaluateOp(left string, op MarkerOp, right string) bool {
	switch op {
	case MarkerOpIn:
		return strings.Contains(right, left)
	case MarkerOpNotIn:
		return !strings.Contains(right, left)
	case MarkerOpEqual:
		return left == right
	case MarkerOpNotEqual:
		return left != right
	}

	// For ordering operators, try version comparison first
	leftVer, leftErr := pep440.Parse(left)
	rightVer, rightErr := pep440.Parse(right)

	if leftErr == nil && rightErr == nil {
		cmp := pep440.Compare(leftVer, rightVer)
		switch op {
		case MarkerOpLess:
			return cmp < 0
		case MarkerOpLessEqual:
			return cmp <= 0
		case MarkerOpGreater:
			return cmp > 0
		case MarkerOpGreaterEqual:
			return cmp >= 0
		}
	}

	// Fall back to string comparison
	switch op {
	case MarkerOpLess:
		return left < right
	case MarkerOpLessEqual:
		return left <= right
	case MarkerOpGreater:
		return left > right
	case MarkerOpGreaterEqual:
		return left >= right
	default:
		return false
	}
}
