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

func TestRunEach_StopsEarly(t *testing.T) {
	p, err := Compile(`.[]`)
	require.NoError(t, err)

	var got []interface{}
	err = p.RunEach([]interface{}{"a", "b", "c", "d"}, func(v interface{}) (bool, error) {
		got = append(got, v)
		return len(got) == 2, nil // stop after two
	})

	require.NoError(t, err)
	assert.Equal(t, []interface{}{"a", "b"}, got)
}

func TestRunEach_PropagatesEmitError(t *testing.T) {
	p, err := Compile(`.[]`)
	require.NoError(t, err)

	err = p.RunEach([]interface{}{"a"}, func(v interface{}) (bool, error) {
		return false, assert.AnError
	})

	assert.ErrorIs(t, err, assert.AnError)
}

func TestCompileWithVariables_ParentAccessible(t *testing.T) {
	p, err := Compile(`{q: .q, chunk: $parent.chop.chunk}`, "$parent")
	require.NoError(t, err)

	out, err := p.Run(
		map[string]interface{}{"q": "why?"},
		map[string]interface{}{"chop": map[string]interface{}{"chunk": "source text"}},
	)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{map[string]interface{}{"q": "why?", "chunk": "source text"}}, out)
}

func TestCompile_UndeclaredVariableFailsAtCompile(t *testing.T) {
	_, err := Compile(`{chunk: $parent.chunk}`) // $parent not declared
	assert.Error(t, err, "using $parent without declaring it must be a compile-time error")
	assert.Contains(t, err.Error(), "$parent")
}

func TestCompileWithVariables_NullParentIsFine(t *testing.T) {
	p, err := Compile(`.v`, "$parent")
	require.NoError(t, err)

	out, err := p.Run(map[string]interface{}{"v": 1}, nil)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{1}, out)
}
