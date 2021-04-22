package dtlabels

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	testLabel        = "test-label"
	testValue        = "test-value"
	anotherTestLabel = "another-test-label"
	anotherTestValue = "another-test-value"
)

func TestAreExpressionsMatching(t *testing.T) {
	t.Run(`Test In Operator`, func(t *testing.T) {
		requirements := []metav1.LabelSelectorRequirement{
			{Key: testLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{testValue, anotherTestValue}},
		}
		matches, err := AreExpressionsMatching(requirements, map[string]string{
			testLabel: testValue,
		})
		assert.NoError(t, err)
		assert.True(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: anotherTestValue,
		})
		assert.NoError(t, err)
		assert.True(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: "yet-another-value",
		})
		assert.NoError(t, err)
		assert.False(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{})
		assert.NoError(t, err)
		assert.False(t, matches)
	})
	t.Run(`Test NotIn operator`, func(t *testing.T) {
		requirements := []metav1.LabelSelectorRequirement{
			{Key: testLabel, Operator: metav1.LabelSelectorOpNotIn, Values: []string{testValue, anotherTestValue}},
		}
		matches, err := AreExpressionsMatching(requirements, map[string]string{
			testLabel: testValue,
		})
		assert.NoError(t, err)
		assert.False(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: anotherTestValue,
		})
		assert.NoError(t, err)
		assert.False(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: "yet-another-value",
		})
		assert.NoError(t, err)
		assert.True(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{})
		assert.NoError(t, err)
		assert.True(t, matches)
	})
	t.Run(`Test Exists operator`, func(t *testing.T) {
		requirements := []metav1.LabelSelectorRequirement{
			{Key: testLabel, Operator: metav1.LabelSelectorOpExists},
		}
		matches, err := AreExpressionsMatching(requirements, map[string]string{
			testLabel: testValue,
		})
		assert.NoError(t, err)
		assert.True(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: anotherTestValue,
		})
		assert.NoError(t, err)
		assert.True(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			"another-label": testValue,
		})
		assert.NoError(t, err)
		assert.False(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{})
		assert.NoError(t, err)
		assert.False(t, matches)
	})
	t.Run(`Test DoesNotExist operator`, func(t *testing.T) {
		requirements := []metav1.LabelSelectorRequirement{
			{Key: testLabel, Operator: metav1.LabelSelectorOpDoesNotExist},
		}
		matches, err := AreExpressionsMatching(requirements, map[string]string{
			testLabel: testValue,
		})
		assert.NoError(t, err)
		assert.False(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: anotherTestValue,
		})
		assert.NoError(t, err)
		assert.False(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{
			"another-label": testValue,
		})
		assert.NoError(t, err)
		assert.True(t, matches)

		matches, err = AreExpressionsMatching(requirements, map[string]string{})
		assert.NoError(t, err)
		assert.True(t, matches)
	})
	t.Run(`Error on incorrect expression`, func(t *testing.T) {
		requirements := []metav1.LabelSelectorRequirement{
			{Key: testLabel, Operator: metav1.LabelSelectorOperator("not-an-operator"), Values: []string{testValue, anotherTestValue}},
		}
		matches, err := AreExpressionsMatching(requirements, map[string]string{
			testLabel: testValue,
		})
		assert.Error(t, err)
		assert.False(t, matches)

		requirements = []metav1.LabelSelectorRequirement{
			{Key: testLabel, Operator: metav1.LabelSelectorOpIn},
		}
		matches, err = AreExpressionsMatching(requirements, map[string]string{
			testLabel: testValue,
		})
		assert.Error(t, err)
		assert.False(t, matches)
	})
}

func TestAreLabelsMatching(t *testing.T) {
	t.Run(`Fails with empty labels`, func(t *testing.T) {
		labelsToMatch := map[string]string{
			testLabel: testValue,
		}
		labelsMatch := AreLabelsMatching(labelsToMatch, map[string]string{})
		assert.False(t, labelsMatch)

		labelsMatch = AreLabelsMatching(map[string]string{}, map[string]string{})
		assert.False(t, labelsMatch)
	})
	t.Run(`Fails partial match`, func(t *testing.T) {
		labelsToMatch := map[string]string{
			testLabel:        testValue,
			anotherTestLabel: anotherTestValue,
		}
		labelsMatch := AreLabelsMatching(labelsToMatch, map[string]string{
			testLabel: testValue,
		})
		assert.False(t, labelsMatch)

		labelsMatch = AreLabelsMatching(labelsToMatch, map[string]string{
			anotherTestLabel: anotherTestValue,
		})
		assert.False(t, labelsMatch)
	})
	t.Run(`Matches full match`, func(t *testing.T) {
		labelsToMatch := map[string]string{
			testLabel:        testValue,
			anotherTestLabel: anotherTestValue,
		}
		labelsMatch := AreLabelsMatching(labelsToMatch, map[string]string{
			testLabel:        testValue,
			anotherTestLabel: anotherTestValue,
		})
		assert.True(t, labelsMatch)

		labelsMatch = AreLabelsMatching(labelsToMatch, map[string]string{
			testLabel:           testValue,
			anotherTestLabel:    anotherTestValue,
			"yet-another-label": "yet-another-value",
		})
		assert.True(t, labelsMatch)
	})
}

func TestRequirementOperatorToSelectionOperator(t *testing.T) {
	assert.Equal(t, selection.In, requirementOperatorToSelectionOperator(metav1.LabelSelectorOpIn))
	assert.Equal(t, selection.NotIn, requirementOperatorToSelectionOperator(metav1.LabelSelectorOpNotIn))
	assert.Equal(t, selection.Exists, requirementOperatorToSelectionOperator(metav1.LabelSelectorOpExists))
	assert.Equal(t, selection.DoesNotExist, requirementOperatorToSelectionOperator(metav1.LabelSelectorOpDoesNotExist))
}