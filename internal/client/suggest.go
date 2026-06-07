package client

import "strings"

// popularPackages is a small set of commonly-mistyped popular Python packages.
var popularPackages = []string{
	"requests", "numpy", "pandas", "flask", "django", "scipy",
	"matplotlib", "tensorflow", "pytorch", "torch", "scikit-learn",
	"sqlalchemy", "celery", "redis", "boto3", "pillow",
	"beautifulsoup4", "selenium", "fastapi", "uvicorn", "gunicorn",
	"pytest", "black", "ruff", "mypy", "pylint", "httpx",
	"pydantic", "aiohttp", "click", "typer", "rich",
	"cryptography", "paramiko", "psycopg2", "pymongo", "elasticsearch",
}

// SuggestPackage returns a likely correct package name given a mistyped name.
// Returns empty string if no good suggestion found.
func SuggestPackage(name string) string {
	lower := strings.ToLower(name)

	// Check common suffix/prefix mistakes
	candidates := []string{
		lower + "s",         // request → requests
		strings.TrimSuffix(lower, "s"), // requestss → requests
		strings.TrimSuffix(lower, "-py"),
		strings.TrimSuffix(lower, "-python"),
		strings.TrimPrefix(lower, "python-"),
		strings.TrimPrefix(lower, "py-"),
		strings.ReplaceAll(lower, "_", "-"),
		strings.ReplaceAll(lower, "-", "_"),
	}

	for _, candidate := range candidates {
		if candidate == lower || candidate == "" {
			continue
		}
		for _, pkg := range popularPackages {
			if candidate == pkg {
				return pkg
			}
		}
	}

	// Levenshtein distance check against popular packages
	for _, pkg := range popularPackages {
		if levenshtein(lower, pkg) <= 2 && lower != pkg {
			return pkg
		}
	}

	return ""
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
