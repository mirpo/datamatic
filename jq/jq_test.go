package jq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_InvalidProgram(t *testing.T) {
	_, err := Compile(".foo | select(")
	assert.Error(t, err)
}

func TestRun_Identity(t *testing.T) {
	p, err := Compile(".")
	require.NoError(t, err)

	out, err := p.Run(map[string]interface{}{"a": float64(1)})
	require.NoError(t, err)
	assert.Equal(t, []interface{}{map[string]interface{}{"a": float64(1)}}, out)
}

func TestRun_SelectFiltersToZeroResults(t *testing.T) {
	p, err := Compile(`select(.keep == true)`)
	require.NoError(t, err)

	out, err := p.Run(map[string]interface{}{"keep": false})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRun_ArrayExplodeFansOut(t *testing.T) {
	p, err := Compile(`.items[]`)
	require.NoError(t, err)

	out, err := p.Run(map[string]interface{}{"items": []interface{}{"a", "b", "c"}})
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"a", "b", "c"}, out)
}

func TestRun_ObjectConstruction(t *testing.T) {
	p, err := Compile(`{q: .question, a: .answer}`)
	require.NoError(t, err)

	out, err := p.Run(map[string]interface{}{"question": "x", "answer": "y", "noise": "z"})
	require.NoError(t, err)
	assert.Equal(t, []interface{}{map[string]interface{}{"q": "x", "a": "y"}}, out)
}

func TestRun_RuntimeErrorSurfaces(t *testing.T) {
	p, err := Compile(`.a + 1`) // works on numbers, errors on strings
	require.NoError(t, err)

	_, err = p.Run(map[string]interface{}{"a": "not-a-number"})
	assert.Error(t, err)
}
