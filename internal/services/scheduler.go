package services

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ScheduledJob represents a registered scheduled job.
type ScheduledJob struct {
	Name     string
	Hour     int
	Minute   int
	Fn       func()
	LastRun  time.Time
	LastErr  string
	NextRun  time.Time
	stop     chan struct{}
}

// Scheduler manages time-based scheduled jobs (daily at specific hour:minute).
type Scheduler struct {
	jobs []*ScheduledJob
	mu   sync.Mutex
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// AddDailyJob registers a job that runs every day at the given hour:minute (server local time).
func (s *Scheduler) AddDailyJob(name string, hour, minute int, fn func()) {
	j := &ScheduledJob{
		Name:   name,
		Hour:   hour,
		Minute: minute,
		Fn:     fn,
		stop:   make(chan struct{}),
	}
	s.mu.Lock()
	s.jobs = append(s.jobs, j)
	s.mu.Unlock()

	go s.runDaily(j)
	log.Printf("[scheduler] registered job %q (daily %02d:%02d)", name, hour, minute)
}

// Status returns the current state of all registered jobs.
func (s *Scheduler) Status() []map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []map[string]interface{}
	for _, j := range s.jobs {
		result = append(result, map[string]interface{}{
			"name":     j.Name,
			"schedule": fmt.Sprintf("%02d:%02d daily", j.Hour, j.Minute),
			"next_run": j.NextRun.Format(time.RFC3339),
			"last_run": j.LastRun.Format(time.RFC3339),
			"last_err": j.LastErr,
		})
	}
	return result
}

// UpdateJobTime changes the schedule of a running job. The old goroutine is
// stopped and a new one is started with the updated hour:minute.
func (s *Scheduler) UpdateJobTime(name string, hour, minute int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, j := range s.jobs {
		if j.Name == name {
			close(j.stop) // stop old goroutine
			newJob := &ScheduledJob{
				Name:   name,
				Hour:   hour,
				Minute: minute,
				Fn:     j.Fn,
				stop:   make(chan struct{}),
			}
			s.jobs[i] = newJob
			go s.runDaily(newJob)
			log.Printf("[scheduler] updated job %q to %02d:%02d", name, hour, minute)
			return nil
		}
	}
	return fmt.Errorf("job %q not found", name)
}

// RemoveJob stops and removes a job by name.
func (s *Scheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, j := range s.jobs {
		if j.Name == name {
			close(j.stop)
			s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
			log.Printf("[scheduler] removed job %q", name)
			return nil
		}
	}
	return fmt.Errorf("job %q not found", name)
}

// AddJobIfExists updates a job's schedule if it already exists, otherwise adds it.
// Useful for re-registering jobs after config changes.
func (s *Scheduler) AddOrUpdateJob(name string, hour, minute int, fn func()) {
	s.mu.Lock()
	for _, j := range s.jobs {
		if j.Name == name {
			s.mu.Unlock()
			_ = s.UpdateJobTime(name, hour, minute)
			return
		}
	}
	s.mu.Unlock()
	s.AddDailyJob(name, hour, minute, fn)
}

// StopAll stops all scheduled jobs.
func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		close(j.stop)
	}
	log.Printf("[scheduler] stopped all %d jobs", len(s.jobs))
}

func (s *Scheduler) runDaily(j *ScheduledJob) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), j.Hour, j.Minute, 0, 0, now.Location())
		if next.Before(now) || next.Equal(now) {
			next = next.Add(24 * time.Hour)
		}
		j.NextRun = next
		log.Printf("[scheduler] %s: next run at %s", j.Name, next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(time.Until(next)):
			log.Printf("[scheduler] %s: running...", j.Name)
			j.LastRun = time.Now()

			// Run with panic recovery
			err := safeRun(j.Fn)
			if err != nil {
				j.LastErr = err.Error()
				log.Printf("[scheduler] %s: error: %v", j.Name, err)
			} else {
				j.LastErr = ""
				log.Printf("[scheduler] %s: completed", j.Name)
			}

		case <-j.stop:
			log.Printf("[scheduler] %s: stopped", j.Name)
			return
		}
	}
}

func safeRun(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	fn()
	return nil
}
