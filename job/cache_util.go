package job

func enable(j *Job, cache JobCache) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	shouldStartWaiting := j.jobTimer != nil && j.Disabled
	j.Disabled = false

	j.lock.Unlock()
	if err := cache.Set(j); err != nil {
		j.lock.Lock()
		j.Disabled = true
		return err
	}
	j.lock.Lock()

	if shouldStartWaiting {
		go j.StartWaiting(cache, false)
	}

	return nil
}

func disable(j *Job, cache JobCache) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.Disabled = true

	j.lock.Unlock()
	if err := cache.Set(j); err != nil {
		j.lock.Lock()
		j.Disabled = false
		return err
	}
	j.lock.Lock()

	if j.jobTimer != nil {
		j.jobTimer.Stop()
	}

	return nil
}
