package processor

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/nghiack7/game-ad-service/internal/domain/models"
)

// Analyzer performs analysis on game ads
type Analyzer struct {
	// Configuration options
	minScore      float64
	maxScore      float64
	analysisDelay time.Duration
}

// AnalyzerOptions configures the analyzer
type AnalyzerOptions struct {
	// MinScore is the minimum score that can be assigned (default: 1.0)
	MinScore float64

	// MaxScore is the maximum score that can be assigned (default: 10.0)
	MaxScore float64

	// AnalysisDelay is additional processing time for each analysis (default: 0)
	AnalysisDelay time.Duration
}

// DefaultAnalyzerOptions returns sensible defaults
func DefaultAnalyzerOptions() AnalyzerOptions {
	return AnalyzerOptions{
		MinScore:      1.0,
		MaxScore:      10.0,
		AnalysisDelay: 0,
	}
}

// NewAnalyzer creates a new ad analyzer
func NewAnalyzer(options AnalyzerOptions) *Analyzer {
	return &Analyzer{
		minScore:      options.MinScore,
		maxScore:      options.MaxScore,
		analysisDelay: options.AnalysisDelay,
	}
}

// Analyze generates analysis results for an ad
// In a real implementation, this would use ML models or complex rules
func (a *Analyzer) Analyze(ad *models.Ad) (models.AdAnalysis, error) {
	// Add configured delay if any
	if a.analysisDelay > 0 {
		time.Sleep(a.analysisDelay)
	}

	// Initialize randomization
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Mock analysis with randomized but meaningful results
	score := a.minScore + r.Float64()*(a.maxScore-a.minScore)

	// Round to one decimal place
	score = float64(int(score*10)) / 10

	analysis := models.AdAnalysis{
		EffectivenessScore:     score,
		Strengths:              []string{},
		ImprovementSuggestions: []string{},
	}

	// Generate relevant strengths based on ad content
	if len(ad.CallToAction) > 0 {
		analysis.Strengths = append(analysis.Strengths,
			fmt.Sprintf("Strong call to action with clear incentive: %s", ad.CallToAction))
	}

	if len(ad.TargetAudience) > 0 {
		analysis.Strengths = append(analysis.Strengths,
			"Appeals to target audience's desire for progression")
	}

	if len(ad.VisualElements) > 0 {
		analysis.Strengths = append(analysis.Strengths,
			fmt.Sprintf("Effective visual elements: %s", ad.VisualElements[0]))
	}

	// Generate improvement suggestions
	if score < 9.0 {
		analysis.ImprovementSuggestions = append(analysis.ImprovementSuggestions,
			"Consider adding social proof elements")
	}

	if score < 8.0 {
		analysis.ImprovementSuggestions = append(analysis.ImprovementSuggestions,
			"Highlight immediate gameplay satisfaction")
	}

	if score < 7.0 {
		analysis.ImprovementSuggestions = append(analysis.ImprovementSuggestions,
			"Optimize creative elements for better engagement")
	}

	return analysis, nil
}

// BatchAnalyze analyzes multiple ads in a batch
// This could be more efficient in a real implementation
func (a *Analyzer) BatchAnalyze(ads []*models.Ad) (map[string]models.AdAnalysis, error) {
	results := make(map[string]models.AdAnalysis)

	for _, ad := range ads {
		analysis, err := a.Analyze(ad)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze ad %s: %w", ad.AdID, err)
		}

		results[ad.AdID] = analysis
	}

	return results, nil
}
