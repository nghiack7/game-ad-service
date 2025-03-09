package mapper

import (
	"github.com/nghiack7/game-ad-service/internal/domain/models"
	"github.com/nghiack7/game-ad-service/internal/interfaces/dtos"
)

func MapAdSubmissionToAdSubmission(submission dtos.AdSubmissionDTO) models.AdSubmission {
	return models.AdSubmission{
		Title:          submission.Title,
		Description:    submission.Description,
		Genre:          submission.Genre,
		TargetAudience: submission.TargetAudience,
		VisualElements: submission.VisualElements,
		CallToAction:   submission.CallToAction,
		Duration:       submission.Duration,
		Priority:       submission.Priority,
	}
}

func MapAdToAdDTO(ad *models.Ad) *dtos.AdDTO {
	return &dtos.AdDTO{
		AdID:           ad.AdID,
		Title:          ad.Title,
		Description:    ad.Description,
		Genre:          ad.Genre,
		TargetAudience: ad.TargetAudience,
		VisualElements: ad.VisualElements,
		CallToAction:   ad.CallToAction,
		Duration:       ad.Duration,
		Priority:       ad.Priority,
		Status:         ad.Status,
		CreatedAt:      ad.CreatedAt,
		CompletedAt:    ad.CompletedAt,
		Analysis:       MapAdAnalysisToAdAnalysisDTO(ad.Analysis),
		Error:          ad.Error,
	}
}

func MapAdAnalysisToAdAnalysisDTO(analysis *models.AdAnalysis) *dtos.AdAnalysisDTO {
	if analysis == nil {
		return nil
	}
	return &dtos.AdAnalysisDTO{
		EffectivenessScore:     analysis.EffectivenessScore,
		Strengths:              analysis.Strengths,
		ImprovementSuggestions: analysis.ImprovementSuggestions,
	}
}

func MapAdSubmissionToAdSubmissionDTO(submission models.AdSubmission) dtos.AdSubmissionDTO {
	return dtos.AdSubmissionDTO{
		Title:          submission.Title,
		Description:    submission.Description,
		Genre:          submission.Genre,
		TargetAudience: submission.TargetAudience,
		VisualElements: submission.VisualElements,
		CallToAction:   submission.CallToAction,
		Duration:       submission.Duration,
		Priority:       submission.Priority,
	}
}
