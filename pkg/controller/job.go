package controller

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func shouldDeleteJob(job *batchv1.Job, deleteSuccessfulAfter, deleteFailedAfter time.Duration, ignoreCronJobs bool) bool {
	// skip the job if it has any active pods
	if job.Status.Active > 0 {
		return false
	}

	if ignoreCronJobs {
		owners := getJobOwnerKinds(job)
		if isOwnedByCronJob(owners) {
			return false
		}
	}

	finishTime := jobFinishTime(job)

	if finishTime.IsZero() {
		return false
	}

	timeSinceFinish := time.Since(finishTime)

	if job.Status.Succeeded > 0 {
		if deleteSuccessfulAfter > 0 && timeSinceFinish > deleteSuccessfulAfter {
			return true
		}
	}
	if job.Status.Failed > 0 {
		if deleteFailedAfter > 0 && timeSinceFinish >= deleteFailedAfter {
			return true
		}
	}
	return false
}

func getJobOwnerKinds(job *batchv1.Job) []string {
	var kinds []string
	for _, ow := range job.OwnerReferences {
		kinds = append(kinds, ow.Kind)
	}
	return kinds
}

// Can return "zero" time, caller must check
func jobFinishTime(jobObj *batchv1.Job) time.Time {
	if !jobObj.Status.CompletionTime.IsZero() {
		return jobObj.Status.CompletionTime.Time
	}

	for _, jc := range jobObj.Status.Conditions {
		// Looking for the time when job's condition "Failed" became "true" (equals end of execution)
		if jc.Type == batchv1.JobFailed && jc.Status == corev1.ConditionTrue {
			return jc.LastTransitionTime.Time
		}
	}

	return time.Time{}
}

// isOwnedByCronJob returns true if and only if job has a single owner CronJob
// and this owners kind is CronJob
func isOwnedByCronJob(ownerKinds []string) bool {
	if len(ownerKinds) == 1 && ownerKinds[0] == "CronJob" {
		return true
	}
	return false
}
