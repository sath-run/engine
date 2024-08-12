package scheduler

type Container struct {
	imageUrl    string
	containerId string
	dir         string
	currentJob  *Job
}
