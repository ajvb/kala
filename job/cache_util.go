package job

// enable clears the job's Disabled flag and, if the job had a timer, starts a
// new one before returning.
func enable(j *Job, cache JobCache, persist bool) error {
	j.lock.Lock()
	shouldStartWaiting := j.jobTimer != nil && j.Disabled
	j.Disabled = false
	j.lock.Unlock()

	if persist {
		if err := cache.Set(j); err != nil {
			j.lock.Lock()
			j.Disabled = true
			j.lock.Unlock()
			return err
		}
	}

	// StartWaiting takes the job's lock itself.
	if shouldStartWaiting {
		j.StartWaiting(cache, false)
	}

	return nil
}

func disable(j *Job, cache JobCache, persist bool) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.Disabled = true
	if persist {
		j.lock.Unlock()
		if err := cache.Set(j); err != nil {
			j.lock.Lock()
			j.Disabled = false
			return err
		}
		j.lock.Lock()
	}

	if j.jobTimer != nil {
		j.jobTimer.Stop()
	}

	return nil
}
