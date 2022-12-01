package straightexit_test

import (
	"testing"

	straightExit "github.com/GermanVor/devops-pet-project/staticlint/straightExit"
	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestMyAnalyzer we are waiting to receive an error message
func TestMyAnalyzer(t *testing.T) {
	t.Run("Expected error", func(t *testing.T) {
		res := analysistest.Run(&testing.T{}, analysistest.TestData(), straightExit.Analyzer, "./...")
		require.Equal(t, 2, len(res))

		// it is about ./testdata/qwe package
		assert.Equal(t, 0, len(res[1].Diagnostics))

		err := res[0]

		require.Equal(t, 1, len(err.Diagnostics))
		assert.Equal(t, straightExit.Category, err.Diagnostics[0].Category)
	})
}
