package security

import "hexajobs.dev/hexajobs-cli/internal/models"

// FilterGhosting hides rates strictly above 75%. Unknown (-1) remains visible.
// The returned slice is independent and never reorders or mutates engine data.
func FilterGhosting(jobs []models.JobListing) []models.JobListing {
	out := make([]models.JobListing, 0, len(jobs))
	for _, job := range jobs {
		if job.GhostingRate <= 0.75 || job.GhostingRate != job.GhostingRate {
			out = append(out, job)
		}
	}
	return out
}
