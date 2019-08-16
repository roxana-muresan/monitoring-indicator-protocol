package indicator_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/api_versions"
	v1 "github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/apis/indicatordocument/v1"

	"github.com/cppforlife/go-patch/patch"

	"github.com/pivotal/monitoring-indicator-protocol/pkg/indicator"
	"github.com/pivotal/monitoring-indicator-protocol/test_fixtures"
)

var (
	v0DocumentBytes = []byte(`---
apiVersion: v0
metadata:
  deployment: test-deployment

product:
  name: testing
  version: 123

indicators:
- name: test_indicator
  promql: test_expr
`)

	v0Match = indicator.Match{
		Name:    test_fixtures.StrPtr("testing"),
		Version: test_fixtures.StrPtr("123"),
		Metadata: map[string]string{
			"deployment": "test-deployment",
		},
	}

	v0Patch = indicator.Patch{
		APIVersion: api_versions.V0Patch,
		Match:      v0Match,
		Operations: nil,
	}

	v1DocumentBytes = []byte(`---
apiVersion: indicatorprotocol.io/v1
kind: IndicatorDocument
metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
`)

	v1Match = indicator.Match{
		Name:    test_fixtures.StrPtr("testing"),
		Version: test_fixtures.StrPtr("123"),
		Metadata: map[string]string{
			"deployment": "test-deployment",
		},
	}

	v1Patch = indicator.Patch{
		APIVersion: api_versions.V1,
		Match:      v1Match,
		Operations: nil,
	}
)

func matchToPatch(apiVersion string, m indicator.Match) indicator.Patch {
	return indicator.Patch{
		APIVersion: apiVersion,
		Match:      m,
		Operations: nil,
	}
}

func TestDocumentMatching(t *testing.T) {
	name1 := "testing"
	version1 := "123"
	matchProduct123 := matchToPatch(api_versions.V1, indicator.Match{
		Name:    &name1,
		Version: &version1,
	})

	matchDeploymentTest := matchToPatch(api_versions.V1, indicator.Match{
		Name:    nil,
		Version: nil,
		Metadata: map[string]string{
			"deployment": "test-deployment",
		},
	})

	name2 := "other-testing"
	version2 := "456"
	matchDeploymentOtherTest := matchToPatch(api_versions.V1, indicator.Match{
		Name:    &name2,
		Version: &version2,
		Metadata: map[string]string{
			"deployment": "other-test-deployment",
		},
	})

	t.Run("name and version", func(t *testing.T) {
		g := NewGomegaWithT(t)

		documentBytes := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: non-matching-test-deployment

spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
  `)

		g.Expect(indicator.MatchDocument(matchProduct123, documentBytes)).To(BeTrue())
		g.Expect(indicator.MatchDocument(matchDeploymentTest, documentBytes)).To(BeFalse())
		g.Expect(indicator.MatchDocument(matchDeploymentOtherTest, documentBytes)).To(BeFalse())
	})

	t.Run("metadata", func(t *testing.T) {
		g := NewGomegaWithT(t)

		documentBytes := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing-foo-foo
    version: 123456
  
  indicators:
  - name: test_indicator
    promql: test_expr
`)

		g.Expect(indicator.MatchDocument(matchProduct123, documentBytes)).To(BeFalse())
		g.Expect(indicator.MatchDocument(matchDeploymentTest, documentBytes)).To(BeTrue())
		g.Expect(indicator.MatchDocument(matchDeploymentOtherTest, documentBytes)).To(BeFalse())
	})

	t.Run("name and version and metadata", func(t *testing.T) {
		g := NewGomegaWithT(t)

		documentBytes := []byte(`
---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: other-test-deployment

spec:
  product:
    name: other-testing
    version: 456

  indicators:
  - name: test_indicator
    promql: test_expr
`)

		g.Expect(indicator.MatchDocument(matchProduct123, documentBytes)).To(BeFalse())
		g.Expect(indicator.MatchDocument(matchDeploymentTest, documentBytes)).To(BeFalse())
		g.Expect(indicator.MatchDocument(matchDeploymentOtherTest, documentBytes)).To(BeTrue())
	})
}

func TestPatching(t *testing.T) {
	t.Run("patches files that match", func(t *testing.T) {
		g := NewGomegaWithT(t)

		matchingDocument := []byte(`
---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment

spec:
  indicators:
    - name: test_indicator
      promql: test_expr
  
  product:
    name: testing
    version: 123
`)
		var val interface{} = "patched_promql"
		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/promql"),
					Value: &val,
				},
			},
		}}

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, matchingDocument)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].PromQL).To(BeEquivalentTo("patched_promql"))
	})

	t.Run("patches v0 thresholds", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var patchedThreshold interface{} = map[interface{}]interface{}{
			"level": "warning",
			"gt":    1000,
		}

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V0Patch,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/indicators/1/thresholds/1"),
					Value: &patchedThreshold,
				},
			},
		}}
		doc := []byte(`---
apiVersion: v0

metadata:
  deployment: test-deployment

product:
  name: testing
  version: 123

indicators:
- name: test_indicator
  promql: test_expr
- name: test_indicator_2
  promql: test_expr
  thresholds: 
  - level: critical
    gt: 1500
  - level: warning
    gt: 500
`)

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[1].Thresholds[1]).To(BeEquivalentTo(v1.Threshold{
			Level:    "warning",
			Operator: v1.GreaterThan,
			Value:    1000,
		}))
	})

	t.Run("does not patch files that do not match", func(t *testing.T) {
		g := NewGomegaWithT(t)

		nonMatchingDocument := []byte(`---
apiVersion: indicatorprotocol.io/v1
metadata:
  label:
    deployment: not-test-deployment
spec:
  product:
    name: testing
    version: 123
  
  
  indicators:
  - name: test_indicator
    promql: test_expr
`)
		var val interface{} = "patched_promql"
		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/promql"),
					Value: &val,
				},
			},
		}}

		unpatchedBytes, err := indicator.ApplyPatches(indicatorPatch, nonMatchingDocument)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(unpatchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].PromQL).To(BeEquivalentTo("test_expr"))
	})

	t.Run("replaces by index", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var patchedThreshold interface{} = map[interface{}]interface{}{
			"level":    "warning",
			"operator": "gt",
			"value":    1000,
		}

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/1/thresholds/1"),
					Value: &patchedThreshold,
				},
			},
		}}
		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
  - name: test_indicator_2
    promql: test_expr
    thresholds: 
    - level: critical
      operator: gt
      value: 1500
    - level: warning
      operator: gt
      value: 500
`)

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[1].Thresholds[1]).To(BeEquivalentTo(v1.Threshold{
			Level:    "warning",
			Operator: v1.GreaterThan,
			Value:    1000,
		}))
	})

	t.Run("replaces by attribute value", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var patchedThreshold interface{} = map[interface{}]interface{}{
			"level":    "warning",
			"operator": "gt",
			"value":    800,
		}

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/name=test_indicator/thresholds/level=warning"),
					Value: &patchedThreshold,
				},
			},
		}}
		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment
spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
    thresholds:
    - level: warning
      operator: gt
      value: 500    
    - level: critical
      operator: gt
      value: 1000
`)

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].Thresholds[0]).To(BeEquivalentTo(v1.Threshold{
			Level:    "warning",
			Operator: v1.GreaterThan,
			Value:    800,
		}))
	})

	t.Run("removes", func(t *testing.T) {
		g := NewGomegaWithT(t)

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "remove",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/thresholds/level=warning"),
					Value: nil,
				},
			},
		}}
		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing
    version: 123
  indicators:
  - name: test_indicator
    promql: test_expr
    thresholds:
    - level: warning
      operator: gt
      value: 500
    - level: critical
      operator: gt
      value: 1000
`)

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].Thresholds).To(HaveLen(1))
	})

	t.Run("ignores `test` operation", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var testVal interface{} = "not_test_indicator"
		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "test",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/name"),
					Value: &testVal,
				},
				{
					Type:  "remove",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/thresholds/level=warning"),
					Value: nil,
				},
			},
		}}
		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing
    version: 123
  indicators:
  - name: test_indicator
    promql: test_expr
    thresholds:
    - level: warning
      operator: gt
      value: 500
`)

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].Thresholds).To(HaveLen(0))
	})

	t.Run("adds by replacing", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var newThresholds interface{} = map[interface{}]interface{}{
			"level":    "warning",
			"operator": "gt",
			"value":    10,
		}

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/name=test_indicator/thresholds?/-"),
					Value: &newThresholds,
				},
			},
		}}
		doc := v1DocumentBytes

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].Thresholds).To(HaveLen(1))
	})

	t.Run("does not error when patch fails due to invalid operation", func(t *testing.T) {
		g := NewGomegaWithT(t)

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type: "replace",
					Path: test_fixtures.StrPtr("/spec/indicators/name=test_indicator/thresholds?/-"),
				},
			},
		}}

		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1
kind: IndicatorDocument

metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
`)

		patchedBytes, err := indicator.ProcessDocument(indicatorPatch, doc)
		g.Expect(err).To(BeEmpty())

		reader := ioutil.NopCloser(bytes.NewReader(doc))
		d, err2 := indicator.DocumentFromYAML(reader)
		g.Expect(patchedBytes).To(Equal(d))
		g.Expect(err2).ToNot(HaveOccurred())
	})

	t.Run("does not error when patch fails due to invalid path", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var val interface{} = "patched_threshold"
		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/35/thresholds/0"),
					Value: &val,
				},
			},
		}}

		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1
kind: IndicatorDocument

metadata:
  labels:
    deployment: test-deployment

spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
`)

		patchedBytes, err := indicator.ProcessDocument(indicatorPatch, doc)
		g.Expect(err).To(BeEmpty())

		reader := ioutil.NopCloser(bytes.NewReader(doc))
		d, err2 := indicator.DocumentFromYAML(reader)
		g.Expect(patchedBytes).To(Equal(d))
		g.Expect(err2).ToNot(HaveOccurred())
	})

	t.Run("applies partially successful patches", func(t *testing.T) {
		g := NewGomegaWithT(t)

		var patchedWarningThreshold interface{} = map[interface{}]interface{}{
			"level":    "warning",
			"operator": "gt",
			"value":    800,
		}
		var patchedCriticalThreshold interface{} = map[interface{}]interface{}{
			"level":    "critical",
			"operator": "gt",
			"value":    5000,
		}
		var patchedPromql interface{} = "foo"

		indicatorPatch := []indicator.Patch{{
			APIVersion: api_versions.V1,
			Match: indicator.Match{
				Metadata: map[string]string{
					"deployment": "test-deployment",
				},
			},
			Operations: []patch.OpDefinition{
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/thresholds/level=warning"),
					Value: &patchedWarningThreshold,
				},
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/1/promql"),
					Value: &patchedPromql,
				},
				{
					Type:  "replace",
					Path:  test_fixtures.StrPtr("/spec/indicators/0/thresholds/level=critical"),
					Value: &patchedCriticalThreshold,
				},
			},
		}}
		doc := []byte(`---
apiVersion: indicatorprotocol.io/v1

metadata:
  labels:
    deployment: test-deployment


spec:
  product:
    name: testing
    version: 123
  
  indicators:
  - name: test_indicator
    promql: test_expr
    thresholds:
    - level: warning
      operator: gt
      value: 500    
    - level: critical
      operator: gt
      value: 1000
`)

		patchedBytes, err := indicator.ApplyPatches(indicatorPatch, doc)
		g.Expect(err).ToNot(HaveOccurred())

		reader := ioutil.NopCloser(bytes.NewReader(patchedBytes))
		d, err := indicator.DocumentFromYAML(reader)
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(d.Spec.Indicators[0].PromQL).To(Equal("test_expr"))
		g.Expect(d.Spec.Indicators[0].Thresholds).To(BeEquivalentTo([]v1.Threshold{
			{
				Level:    "warning",
				Operator: v1.GreaterThan,
				Value:    800,
			},
			{
				Level:    "critical",
				Operator: v1.GreaterThan,
				Value:    5000,
			},
		}))
	})
}

func TestPatchingApiCompatibility(t *testing.T) {

	t.Run("v0 patches don't match v1 docs", func(t *testing.T) {
		g := NewGomegaWithT(t)
		g.Expect(indicator.MatchDocument(v0Patch, v1DocumentBytes)).To(BeFalse())
	})

	t.Run("v0 patches match v0 docs", func(t *testing.T) {
		g := NewGomegaWithT(t)
		g.Expect(indicator.MatchDocument(v0Patch, v0DocumentBytes)).To(BeTrue())
	})

	t.Run("v1 patches match v1 docs", func(t *testing.T) {
		g := NewGomegaWithT(t)
		g.Expect(indicator.MatchDocument(v1Patch, v1DocumentBytes)).To(BeTrue())

	})

	t.Run("v1 patches don't match v0 docs", func(t *testing.T) {
		g := NewGomegaWithT(t)
		g.Expect(indicator.MatchDocument(v1Patch, v0DocumentBytes)).To(BeFalse())
	})
}
